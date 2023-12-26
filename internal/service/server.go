package service

import (
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"errors"
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

	if err := s.Validate(server); err != nil { // Validate object. handles logging.
		return err
	}

	if err := s.UpsertServerInDatabase(server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) UpdateServer(server entity.Server) error {
	slog.Debug("updating server",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
	)

	// get server from db.
	serverFromDB, err := s.GetServerFromDatabase(server.UserPrincipalName)
	if err != nil {
		return errors.New("not able to get server from database")
	}

	// only update some properties
	serverFromDB.AutoCreate = server.AutoCreate
	serverFromDB.AutoDestroy = server.AutoDestroy
	serverFromDB.InactivityDurationInSeconds = server.InactivityDurationInSeconds

	// Validate object.
	if err := s.Validate(serverFromDB); err != nil {
		return err
	}

	// Update server in database.
	if err := s.UpsertServerInDatabase(serverFromDB); err != nil {
		return err
	}

	return nil
}

func (s *serverService) DeployServer(server entity.Server) (entity.Server, error) {
	slog.Debug("deploying server",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
	)

	// get server from db.
	serverFromDB, err := s.GetServerFromDatabase(server.UserPrincipalName)
	if err != nil {
		return server, err
	}

	// if server is already deployed or deploying, return.
	if serverFromDB.Status == entity.ServerStatusDeploying || serverFromDB.Status == entity.ServerStatusRunning {
		slog.Info("server is already deployed or deploying",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("status", string(server.Status)),
		)
		return serverFromDB, nil
	}

	server.SubscriptionId = serverFromDB.SubscriptionId

	// Validate input.
	if err := s.Validate(server); err != nil {
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
	if err := s.UpsertServerInDatabase(server); err != nil {
		return server, err
	}

	server, err = s.serverRepository.DeployAzureContainerGroup(server)
	if err != nil {
		slog.Error("deploying server failed",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	// convert to int
	waitTimeSeconds, err := strconv.Atoi(s.appConfig.ActlabsServerUPWaitTimeSeconds)
	if err != nil {
		slog.Error("error converting ACTLABS_SERVER_UP_WAIT_TIME_SECONDS to int",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("ACTLABS_SERVER_UP_WAIT_TIME_SECONDS", s.appConfig.ActlabsServerUPWaitTimeSeconds),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	// Ensure server is up and running. check every 5 seconds for 3 minutes.
	for i := 0; i < waitTimeSeconds/5; i++ {
		if err := s.serverRepository.EnsureServerUp(server); err == nil {
			slog.Info("Server is up and running",
				slog.String("userPrincipalName", server.UserPrincipalName),
				slog.String("subscriptionId", server.SubscriptionId),
				slog.String("status", string(server.Status)),
			)

			server.Status = entity.ServerStatusRunning
			server.LastUserActivityTime = time.Now().Format(time.RFC3339)
			server.DeployedAtTime = time.Now().Format(time.RFC3339)

			// Update server in database.
			if err := s.UpsertServerInDatabase(server); err != nil {
				return server, err
			}

			return server, err
		}
		time.Sleep(5 * time.Second)
	}

	slog.Error("server deployed, but not able to verify server is up and running",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
	)

	server.Status = entity.ServerStatusUnknown

	// Update server in database.
	if err := s.UpsertServerInDatabase(server); err != nil {
		return server, err
	}

	return server, errors.New("server deployed, but not able to verify server is up and running")
}

func (s *serverService) DestroyServer(userPrincipalName string) error {
	slog.Debug("destroying server",
		slog.String("userPrincipalName", userPrincipalName),
	)

	// get server from db.
	server, err := s.GetServerFromDatabase(userPrincipalName)
	if err != nil {
		return err
	}

	// Validate object.
	if err := s.Validate(server); err != nil {
		return err
	}

	s.ServerDefaults(&server)

	if err := s.serverRepository.DestroyAzureContainerGroup(server); err != nil {
		slog.Error("error destroying server:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("status", string(server.Status)),
			slog.String("error", err.Error()),
		)
		return err
	}

	server.Status = entity.ServerStatusDestroyed
	server.DestroyedAtTime = time.Now().Format(time.RFC3339)

	if err := s.UpsertServerInDatabase(server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) GetServer(userPrincipalName string) (entity.Server, error) {
	slog.Debug("getting server",
		slog.String("userPrincipalName", userPrincipalName),
	)

	// get server from db.
	server, err := s.GetServerFromDatabase(userPrincipalName)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			server.Status = entity.ServerStatusUnregistered
			return server, nil
		}

		return server, errors.New("not able to get server status from database")
	}
	return server, nil
}

func (s *serverService) UpdateActivityStatus(userPrincipalName string) error {
	slog.Debug("updating server activity status",
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
	servers, err := s.serverRepository.GetServerFromDatabase("actlabs", userPrincipalName)
	if err != nil {
		slog.Error("error upserting server in db",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)
		return servers, err
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
		return errors.New("failed to verify if user is the owner of subscription")
	}
	if !ok {
		slog.Error("user is not the owner of subscription:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
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

		return errors.New("managed identity not found. please register your subscription")
	}

	return nil
}
