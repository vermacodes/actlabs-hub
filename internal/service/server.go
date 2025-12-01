package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
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

func (s *serverService) RegisterSubscription(ctx context.Context, server entity.Server) error {
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

	if err := s.Validate(ctx, server); err != nil { // Validate object. handles logging.
		return err
	}

	if err := s.UpsertServerInDatabase(ctx, server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) Unregister(ctx context.Context, userPrincipalName string) error {
	logger.LogInfo(ctx, "unregistering server")

	// get server from db.
	server, err := s.GetServerFromDatabase(ctx, userPrincipalName)
	if err != nil {
		return err
	}

	// delete server from db.
	if err := s.serverRepository.DeleteServerFromDatabase(ctx, server); err != nil {
		logger.LogError(ctx, "failed to delete server from database",
			"subscription_id", server.SubscriptionId,
			"error", err,
		)

		return fmt.Errorf("resources were destroyed, but got error deleting server from db")
	}

	return nil
}

// func (s *serverService) createEvent(server entity.Server, eventType, reason, message string) {
// 	s.eventService.CreateEvent(context.TODO(), entity.Event{
// 		Type:      eventType,
// 		Reason:    reason,
// 		Message:   message,
// 		Reporter:  "actlabs-hub",
// 		Object:    server.UserPrincipalName,
// 		TimeStamp: time.Now().Format(time.RFC3339),
// 	})
// }

func (s *serverService) GetServer(ctx context.Context, userPrincipalName string) (entity.Server, error) {
	// get server from db.
	server, err := s.serverRepository.GetServerFromDatabase(ctx, "actlabs", userPrincipalName)
	if err != nil {
		logger.LogError(ctx, "failed to get server from database",
			"error", err,
		)

		if strings.Contains(err.Error(), "404 Not Found") {
			s.ServerDefaults(&server)
			server.Status = entity.ServerStatusUnregistered
			return server, nil
		}

		return server, errors.New("not able to get server status from database")
	}

	// update endpoint to accommodate new changes.
	server.Endpoint = s.appConfig.ActlabsServerEndpointExternal

	return server, nil
}

func (s *serverService) UpdateActivityStatus(ctx context.Context, userPrincipalName string) error {
	server, err := s.GetServerFromDatabase(ctx, userPrincipalName)
	if err != nil {
		return errors.New("error getting server from database")
	}

	server.LastUserActivityTime = time.Now().Format(time.RFC3339)

	if err := s.UpsertServerInDatabase(ctx, server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) GetServerFromDatabase(ctx context.Context, userPrincipalName string) (entity.Server, error) {
	server, err := s.serverRepository.GetServerFromDatabase(ctx, "actlabs", userPrincipalName)
	if err != nil {
		logger.LogError(ctx, "failed to get server from database",
			"error", err,
		)
		return server, fmt.Errorf(
			"not able to find server for %s in database, is it registered?",
			userPrincipalName,
		)
	}

	return server, nil
}

func (s *serverService) GetAllServers(ctx context.Context) ([]entity.Server, error) {
	servers, err := s.serverRepository.GetAllServersFromDatabase(ctx)
	if err != nil {
		logger.LogError(ctx, "failed to get all servers from database",
			"error", err,
		)
		return servers, fmt.Errorf("not able to find servers in database")
	}

	return servers, nil
}

func (s *serverService) UpsertServerInDatabase(ctx context.Context, server entity.Server) error {
	// Update server in database.
	if err := s.serverRepository.UpsertServerInDatabase(ctx, server); err != nil {
		logger.LogError(ctx, "failed to upsert server in database",
			"subscription_id", server.SubscriptionId,
			"error", err,
		)
		return err
	}

	return nil
}

func (s *serverService) Validate(ctx context.Context, server entity.Server) error {
	if server.UserPrincipalName == "" ||
		(server.UserPrincipalId == "" && server.FdpoUserPrincipalId == "") ||
		server.SubscriptionId == "" {
		logger.LogError(ctx, "server validation failed: required fields missing",
			"user_principal_name", server.UserPrincipalName,
			"user_principal_id", server.UserPrincipalId,
			"fdpo_user_principal_id", server.FdpoUserPrincipalId,
			"subscription_id", server.SubscriptionId,
		)
		return errors.New(
			"userPrincipalName, userPrincipalId or FdpoUserPrincipalId, and subscriptionId are all required",
		)
	}

	if server.UserAlias == "" {
		server.UserAlias = strings.Split(server.UserPrincipalName, "@")[0]
	}

	if s.appConfig.ActlabsEnvironmentName == "local" {
		// Skip authorization check in local environment.
		return nil
	}

	ok, err := s.serverRepository.IsUserAuthorized(ctx, server)
	if err != nil {
		logger.LogError(ctx, "failed to verify user authorization for subscription",
			"subscription_id", server.SubscriptionId,
			"server_version", server.Version,
			"error", err,
		)
		return errors.New("failed to verify if user is the owner or contributor of subscription")
	}
	if !ok {
		logger.LogError(ctx, "user is not the owner or contributor of subscription",
			"subscription_id", server.SubscriptionId,
			"server_version", server.Version,
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

	server.Endpoint = s.appConfig.ActlabsServerEndpointExternal
}
