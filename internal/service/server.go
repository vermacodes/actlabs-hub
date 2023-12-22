package service

import (
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slog"
)

type serverService struct {
	serverRepository entity.ServerRepository
	appConfig        *config.Config
}

func NewServerService(
	serverRepository entity.ServerRepository,
	appConfig *config.Config,
) entity.ServerService {
	return &serverService{
		serverRepository: serverRepository,
		appConfig:        appConfig,
	}
}

func (s *serverService) RegisterSubscription(subscriptionId string, userPrincipalName string, userPrincipalId string) error {
	server := entity.Server{
		PartitionKey:      "actlabs",
		RowKey:            userPrincipalName,
		SubscriptionId:    subscriptionId,
		UserPrincipalId:   userPrincipalId,
		UserPrincipalName: userPrincipalName,
	}

	s.ServerDefaults(&server) // Set defaults.

	if err := s.Validate(server); err != nil {
		slog.Error("Error:", err)
		return err
	}

	if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {
		slog.Error("Error:", err)
		return err
	}

	return nil
}

func (s *serverService) UpdateServer(server entity.Server) error {
	// get server from db.
	serverFromDB, err := s.serverRepository.GetServerFromDatabase("actlabs", server.UserPrincipalName)
	if err != nil {
		slog.Error("Error:", err)
		return err
	}

	// only update some properties
	serverFromDB.AutoCreate = server.AutoCreate
	serverFromDB.AutoDestroy = server.AutoDestroy
	serverFromDB.InactivityDurationInSeconds = server.InactivityDurationInSeconds

	// Validate object.
	if err := s.Validate(serverFromDB); err != nil {
		slog.Error("Error:", err)
		return err
	}

	// Update server in database.
	if err := s.serverRepository.UpsertServerInDatabase(serverFromDB); err != nil {
		slog.Error("Error:", err)
		return err
	}

	return nil
}

func (s *serverService) DeployServer(server entity.Server) (entity.Server, error) {
	// get server from db.
	serverFromDB, err := s.serverRepository.GetServerFromDatabase("actlabs", server.UserPrincipalName)
	if err != nil {
		slog.Error("Error:", err)
		return server, err
	}

	// if server is already deployed or deploying, return.
	if serverFromDB.Status == entity.ServerStatusDeploying || serverFromDB.Status == entity.ServerStatusRunning {
		slog.Info("Server is already deployed or deploying")
		return serverFromDB, nil
	}

	server.SubscriptionId = serverFromDB.SubscriptionId

	// Validate input.
	if err := s.Validate(server); err != nil {
		slog.Error("Error:", err)
		return server, err
	}

	s.ServerDefaults(&server) // Set defaults.

	// Managed Identity
	if err := s.UserAssignedIdentity(&server); err != nil {
		return server, err
	}

	// Before deploying, update the status in db.
	server.Status = entity.ServerStatusDeploying
	// Update server in database.
	if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {
		slog.Error("not able to update server in database", err)
		return server, fmt.Errorf("server deployment interrupted because not able to update status in db : %w", err)
	}

	server, err = s.serverRepository.DeployAzureContainerGroup(server)
	if err != nil {
		slog.Error("Error:", err)
		return server, err
	}

	// convert to int
	waitTimeSeconds, err := strconv.Atoi(s.appConfig.ActlabsServerUPWaitTimeSeconds)
	if err != nil {
		slog.Error("Error:", err)
		return server, err
	}

	// Ensure server is up and running. check every 5 seconds for 3 minutes.
	for i := 0; i < waitTimeSeconds/5; i++ {
		if err := s.serverRepository.EnsureServerUp(server); err == nil {
			slog.Info("Server is up and running")

			server.Status = entity.ServerStatusRunning
			server.LastUserActivityTime = time.Now().Format(time.RFC3339)
			server.DeployedAtTime = time.Now().Format(time.RFC3339)

			// Update server in database.
			if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {
				slog.Error("not able to update server in database", err)
				return server, fmt.Errorf("server has been deployed but failed to update status in database: %w", err)
			}

			return server, err
		}
		time.Sleep(5 * time.Second)
	}

	server.Status = entity.ServerStatusFailed

	return server, nil
}

func (s *serverService) DestroyServer(userPrincipalName string) error {

	// get server from db.
	server, err := s.serverRepository.GetServerFromDatabase("actlabs", userPrincipalName)
	if err != nil {
		slog.Error("Error:", err)
		return err
	}

	// Validate object.
	if err := s.Validate(server); err != nil {
		slog.Error("Error:", err)
		return err
	}

	s.ServerDefaults(&server)

	if err := s.serverRepository.DestroyAzureContainerGroup(server); err != nil {
		slog.Error("error destroying server:", err)
		return err
	}

	server.Status = entity.ServerStatusDestroyed
	server.DestroyedAtTime = time.Now().Format(time.RFC3339)

	if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {
		slog.Error("error updating server status after destroy:", err)
		return fmt.Errorf("server has been destroyed but failed to update status in database: %w", err)
	}

	return nil
}

func (s *serverService) GetServer(userPrincipalName string) (entity.Server, error) {
	// get server from db.
	server, err := s.serverRepository.GetServerFromDatabase("actlabs", userPrincipalName)
	if err != nil {
		slog.Error("error getting server status from db:", err)

		if strings.Contains(err.Error(), "404 Not Found") {
			server.Status = entity.ServerStatusUnregistered
			return server, nil
		}

		return server, fmt.Errorf("not able to get server status from database: %w", err)
	}
	return server, nil
}

func (s *serverService) UpdateActivityStatus(userPrincipalName string) error {
	server, err := s.serverRepository.GetServerFromDatabase("actlabs", userPrincipalName)
	if err != nil {
		slog.Error("Error getting server from database:", err)
		return fmt.Errorf("error getting server from database: %w", err)
	}

	server.LastUserActivityTime = time.Now().Format(time.RFC3339)

	if err := s.serverRepository.UpsertServerInDatabase(server); err != nil {
		slog.Error("Error updating server in database:", err)
		return fmt.Errorf("error updating server in database: %w", err)
	}

	return nil
}

func (s *serverService) Validate(server entity.Server) error {
	if server.UserPrincipalName == "" || server.UserPrincipalId == "" || server.SubscriptionId == "" {
		return errors.New("userPrincipalName, userPrincipalId, and subscriptionId are all required")
	}

	if server.UserAlias == "" {
		server.UserAlias = strings.Split(server.UserPrincipalName, "@")[0]
	}

	ok, err := s.serverRepository.IsUserOwner(server)
	if err != nil {
		slog.Error("failed to verify if user is the owner of subscription:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to verify if user is the owner of subscription")
	}
	if !ok {
		slog.Error("Error: user is not the owner of the subscription")
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

	if server.Region == "" {
		server.Region = "East US"
	}

	if server.ResourceGroup == "" {
		server.ResourceGroup = "repro-project"
	}

	if server.InactivityDurationInSeconds == 0 {
		server.InactivityDurationInSeconds = 3600
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

func (s *serverService) UserAssignedIdentity(server *entity.Server) error {

	var err error
	*server, err = s.serverRepository.GetUserAssignedManagedIdentity(*server)
	if err != nil {
		slog.Error("Managed Identity not found...",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("status", string(server.Status)),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("managed identity not found. please register your subscription")
	}

	return nil
}
