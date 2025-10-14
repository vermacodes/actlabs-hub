package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/slog"

	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
)

type serverService struct {
	serverRepository entity.ServerRepository
	appConfig        *config.Config
	eventService     entity.EventService
}

func NewServerService(
	serverRepository entity.ServerRepository,
	appConfig *config.Config,
	eventService entity.EventService,
) entity.ServerService {
	return &serverService{
		serverRepository: serverRepository,
		appConfig:        appConfig,
		eventService:     eventService,
	}
}

func (s *serverService) RegisterSubscription(server entity.Server) error {
	server.UserPrincipalName = server.UserAlias + "@microsoft.com"
	server.PartitionKey = "actlabs"
	server.RowKey = server.UserPrincipalName
	server.Version = "V2"
	server.Region = "eastus"

	// if tenant is fdpo, set version to V3.
	// set FdpoUserPrincipalId to userPrincipalId.
	// set UserPrincipalId to empty.
	if server.TenantID == s.appConfig.FdpoTenantID {
		server.Version = "V3"
		server.FdpoUserPrincipalId = server.UserPrincipalId
		server.UserPrincipalId = ""
	}

	s.ServerDefaults(&server) // Set defaults.

	if err := s.Validate(server); err != nil { // Validate object. handles logging.
		return err
	}

	if err := s.UpsertServerInDatabase(server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) Unregister(ctx context.Context, userPrincipalName string) error {
	slog.Info("un-registering server",
		slog.String("userPrincipalName", userPrincipalName),
	)

	// get server from db.
	server, err := s.GetServerFromDatabase(userPrincipalName)
	if err != nil {
		return err
	}

	// delete server from db.
	if err := s.serverRepository.DeleteServerFromDatabase(ctx, server); err != nil {
		slog.Error("error deleting server from db",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("resources were destroyed, but got error deleting server from db")
	}

	return nil
}

func (s *serverService) createEvent(server entity.Server, eventType, reason, message string) {
	s.eventService.CreateEvent(context.TODO(), entity.Event{
		Type:      eventType,
		Reason:    reason,
		Message:   message,
		Reporter:  "actlabs-hub",
		Object:    server.UserPrincipalName,
		TimeStamp: time.Now().Format(time.RFC3339),
	})
}

func (s *serverService) GetServer(userPrincipalName string) (entity.Server, error) {
	slog.Info("getting server",
		slog.String("userPrincipalName", userPrincipalName),
	)

	// get server from db.
	server, err := s.serverRepository.GetServerFromDatabase("actlabs", userPrincipalName)
	if err != nil {
		slog.Error("error getting server from db",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)

		if strings.Contains(err.Error(), "404 Not Found") {
			s.ServerDefaults(&server)
			server.Status = entity.ServerStatusUnregistered
			return server, nil
		}

		return server, errors.New("not able to get server status from database")
	}
	return server, nil
}

func (s *serverService) UpdateActivityStatus(userPrincipalName string) error {
	slog.Info("updating server activity status",
		slog.String("userPrincipalName", userPrincipalName),
	)

	server, err := s.GetServerFromDatabase(userPrincipalName)
	if err != nil {
		return errors.New("error getting server from database")
	}

	server.LastUserActivityTime = time.Now().Format(time.RFC3339)

	if err := s.UpsertServerInDatabase(server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) GetServerFromDatabase(userPrincipalName string) (entity.Server, error) {
	server, err := s.serverRepository.GetServerFromDatabase("actlabs", userPrincipalName)
	if err != nil {
		slog.Error("error getting server from db",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)
		return server, fmt.Errorf(
			"not able to find server for %s in database, is it registered?",
			userPrincipalName,
		)
	}

	return server, nil
}

func (s *serverService) GetAllServers(ctx context.Context) ([]entity.Server, error) {
	slog.Info("getting all servers")
	servers, err := s.serverRepository.GetAllServersFromDatabase(ctx)
	if err != nil {
		slog.Error("error getting all servers from db",
			slog.String("error", err.Error()),
		)
		return servers, fmt.Errorf("not able to find servers in database")
	}

	return servers, nil
}

func (s *serverService) UpsertServerInDatabase(server entity.Server) error {
	// Update server in database.
	if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {
		slog.Error("error upserting server in db",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (s *serverService) Validate(server entity.Server) error {
	if server.UserPrincipalName == "" ||
		(server.UserPrincipalId == "" && server.FdpoUserPrincipalId == "") ||
		server.SubscriptionId == "" {
		slog.Error(
			"Server validation failed. userPrincipalName, userPrincipalId or FdpoUserPrincipalId, and subscriptionId are all required",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("userPrincipalId", server.UserPrincipalId),
			slog.String("FdpoUserPrincipalId", server.FdpoUserPrincipalId),
			slog.String("subscriptionId", server.SubscriptionId),
		)
		return errors.New(
			"userPrincipalName, userPrincipalId or FdpoUserPrincipalId, and subscriptionId are all required",
		)
	}

	if server.UserAlias == "" {
		server.UserAlias = strings.Split(server.UserPrincipalName, "@")[0]
	}

	ok, err := s.serverRepository.IsUserAuthorized(server)
	if err != nil {
		slog.Error("failed to verify if user is the owner or contributor of subscription:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("serverVersion", server.Version),
			slog.String("error", err.Error()),
		)
		return errors.New("failed to verify if user is the owner or contributor of subscription")
	}
	if !ok {
		slog.Error("user is not the owner or contributor of subscription:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("serverVersion", server.Version),
		)
		return errors.New("insufficient permissions")
	}

	return nil
}

func (s *serverService) ServerDefaults(server *entity.Server) {
	if server.UserAlias == "" {
		server.UserAlias = helper.UserAlias(server.UserPrincipalName)
	}

	if server.LogLevel == "" {
		server.LogLevel = "0"
	}

	if server.ResourceGroup == "" {
		server.ResourceGroup = "repro-project"
	}

	if server.InactivityDurationInSeconds == 0 {
		server.InactivityDurationInSeconds = 900
	}

	if !server.AutoCreate {
		server.AutoCreate = true
	}

	if !server.AutoDestroy {
		server.AutoDestroy = true
	}

	if server.Status == "" {
		server.Status = entity.ServerStatusRegistered
	}
}
