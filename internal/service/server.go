package service

import (
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slog"
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
	// server := entity.Server{
	// 	PartitionKey:      "actlabs",
	// 	RowKey:            userPrincipalName,
	// 	SubscriptionId:    subscriptionId,
	// 	UserPrincipalId:   userPrincipalId,
	// 	UserPrincipalName: userPrincipalName,
	// 	Version:           "V2",
	// 	Region:            "eastus",
	// }
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

	// All deployments are being done in East US.
	// get resource group region.
	// region, err := s.serverRepository.GetResourceGroupRegion(context.TODO(), server)
	// if err != nil {
	// 	slog.Error("error getting resource group region",
	// 		slog.String("userPrincipalName", server.UserPrincipalName),
	// 		slog.String("subscriptionId", server.SubscriptionId),
	// 		slog.String("error", err.Error()),
	// 	)
	// 	return fmt.Errorf("error getting resource group region. is it deployed?")
	// }
	// server.Region = region

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

	if err := s.DestroyServer(userPrincipalName); err != nil {
		return err
	}

	// delete resource group
	if err := s.serverRepository.DeleteResourceGroup(ctx, server); err != nil {
		slog.Error("error deleting resource group",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)

		if !strings.Contains(err.Error(), "ERROR CODE: ResourceGroupNotFound") {
			return fmt.Errorf("error deleting resource group")
		}
	}

	// if server.Version == "V2" {
	// 	// delete storage account
	// 	if err := s.serverRepository.DeleteStorageAccount(ctx, server); err != nil {
	// 		slog.Error("error deleting storage account",
	// 			slog.String("userPrincipalName", server.UserPrincipalName),
	// 			slog.String("subscriptionId", server.SubscriptionId),
	// 			slog.String("error", err.Error()),
	// 		)

	// 		return fmt.Errorf("error deleting storage account")
	// 	}
	// } else {
	// 	// delete resource group
	// 	if err := s.serverRepository.DeleteResourceGroup(ctx, server); err != nil {
	// 		slog.Error("error deleting resource group",
	// 			slog.String("userPrincipalName", server.UserPrincipalName),
	// 			slog.String("subscriptionId", server.SubscriptionId),
	// 			slog.String("error", err.Error()),
	// 		)

	// 		return fmt.Errorf("error deleting resource group")
	// 	}
	// }

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

func (s *serverService) UpdateServer(server entity.Server) error {
	slog.Info("updating server",
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
	slog.Info("deploying server",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
	)

	// get server from db.
	serverFromDB, err := s.GetServerFromDatabase(server.UserPrincipalName)
	if err != nil {
		// if not able to get server from db, that means server is not registered.
		// return error.
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

	// here onwards, just use the server which is already in DB and take action.
	// this will ensure that we are not overriding any properties which are not
	// passed in the request.
	server = serverFromDB

	// Validate input.
	if err := s.Validate(server); err != nil {
		return server, err
	}

	s.ServerDefaults(&server) // Set defaults.

	// Managed Identity
	if err := s.UserAssignedIdentity(&server); err != nil {
		return server, err
	}

	// Verify Actlabs Access
	if err := s.VerifyActlabsAccess(&server); err != nil {
		return server, err
	}

	// Before deploying, update the status in db.
	server.Status = entity.ServerStatusDeploying

	// Update server in database.
	if err := s.UpsertServerInDatabase(server); err != nil {
		return server, err
	}

	// Deploy server. Retry 5 times.
	for i := 0; i < 5; i++ {
		server, err = s.serverRepository.DeployServer(server)
		if err == nil {
			break
		}

		slog.Error("deploying server failed",
			slog.String("backoff", strconv.FormatFloat(math.Min(math.Pow(2, float64(i))*10, 120.0), 'f', -1, 64)+"s"),
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("attempt", strconv.Itoa(i+1)),
			slog.String("error", err.Error()),
		)

		if i < 4 { // Don't sleep after the last attempt
			sleepDuration := math.Min(math.Pow(2, float64(i))*10, 120.0)
			time.Sleep(time.Duration(sleepDuration) * time.Second) // Exponential backoff: wait for twice the time from previous wait and a maximum of 120 seconds
		}
	}

	if err != nil {
		// Server Deployment Failed, Reset Status and Update database.
		// Makes no sense to handle error here.
		server.Status = entity.ServerStatusFailed
		s.UpsertServerInDatabase(server)

		s.eventService.CreateEvent(context.TODO(), entity.Event{
			Type:      "Warning",
			Reason:    "ServerDeploymentFailed",
			Message:   "server deployment for user " + server.UserPrincipalName + " in subscription " + server.SubscriptionId + " with version " + server.Version + " failed" + " with error " + err.Error(),
			Reporter:  "actlabs-hub",
			Object:    server.UserPrincipalName,
			TimeStamp: time.Now().Format(time.RFC3339),
		})

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
		return server, fmt.Errorf("server deployment started, but not able to verify server is up and running")
	}

	// Ensure server is up and running. check every 5 seconds for 3 minutes.
	for i := 0; i < waitTimeSeconds/5; i++ {
		if err := s.serverRepository.EnsureServerUp(server); err == nil {
			slog.Info("server is up and running",
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

			// Create Event
			s.eventService.CreateEvent(context.TODO(), entity.Event{
				Type:      "Normal",
				Reason:    "ServerDeployed",
				Message:   "server deployed for user " + server.UserPrincipalName + " in subscription " + server.SubscriptionId + " with version " + server.Version,
				Reporter:  "actlabs-hub",
				Object:    server.UserPrincipalName,
				TimeStamp: time.Now().Format(time.RFC3339),
			})

			// return server. this is the success case.
			return server, nil
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

	// Create Event
	s.eventService.CreateEvent(context.TODO(), entity.Event{
		Type:      "Warning",
		Reason:    "ServerUnknown",
		Message:   "server deployed for user " + server.UserPrincipalName + " in subscription " + server.SubscriptionId + " with version " + server.Version + " but not able to verify server is up and running",
		Reporter:  "actlabs-hub",
		Object:    server.UserPrincipalName,
		TimeStamp: time.Now().Format(time.RFC3339),
	})

	return server, errors.New("server deployed, but not able to verify server is up and running")
}

func (s *serverService) DestroyServer(userPrincipalName string) error {
	slog.Info("destroying server",
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

	// Retry 5 times.
	for i := 0; i < 5; i++ {
		err = s.serverRepository.DestroyServer(server)
		if err == nil {
			break
		}

		slog.Error("destroying server failed",
			slog.String("backoff", strconv.FormatFloat(math.Min(math.Pow(2, float64(i))*10, 120.0), 'f', -1, 64)+"s"),
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("attempt", strconv.Itoa(i+1)),
			slog.String("error", err.Error()),
		)

		if i < 4 { // Don't sleep after the last attempt
			sleepDuration := math.Min(math.Pow(2, float64(i))*10, 120.0)
			time.Sleep(time.Duration(sleepDuration) * time.Second) // Exponential backoff: wait for twice the time from previous wait and a maximum of 120 waitTimeSeconds
		}
	}

	if err != nil {
		// Create Event
		s.eventService.CreateEvent(context.TODO(), entity.Event{
			Type:      "Normal",
			Reason:    "ServerDestroyFailed",
			Message:   "Failed destroying server for user " + server.UserPrincipalName + " in subscription " + server.SubscriptionId + " with version " + server.Version,
			Reporter:  "actlabs-hub",
			Object:    server.UserPrincipalName,
			TimeStamp: time.Now().Format(time.RFC3339),
		})

		slog.Error("destroying server failed, all attempts exhausted",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)

		return err
	}

	server.Status = entity.ServerStatusDestroyed
	server.DestroyedAtTime = time.Now().Format(time.RFC3339)

	if err := s.UpsertServerInDatabase(server); err != nil {
		return err
	}

	// Create Event
	s.eventService.CreateEvent(context.TODO(), entity.Event{
		Type:      "Normal",
		Reason:    "ServerDestroyed",
		Message:   "server destroyed for user " + server.UserPrincipalName + " in subscription " + server.SubscriptionId + " with version " + server.Version,
		Reporter:  "actlabs-hub",
		Object:    server.UserPrincipalName,
		TimeStamp: time.Now().Format(time.RFC3339),
	})

	return nil
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
		return server, fmt.Errorf("not able to find server for %s in database, is it registered?", userPrincipalName)
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

func (s *serverService) FailedServerDeployment(server entity.Server) error {
	server.Status = entity.ServerStatusFailed
	server.LastUserActivityTime = time.Now().Format(time.RFC3339)

	// Update server in database.
	if err := s.UpsertServerInDatabase(server); err != nil {
		return err
	}

	return nil
}

func (s *serverService) Validate(server entity.Server) error {
	if server.UserPrincipalName == "" || (server.UserPrincipalId == "" && server.FdpoUserPrincipalId == "") || server.SubscriptionId == "" {
		slog.Error("Server validation failed. userPrincipalName, userPrincipalId or FdpoUserPrincipalId, and subscriptionId are all required",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("userPrincipalId", server.UserPrincipalId),
			slog.String("FdpoUserPrincipalId", server.FdpoUserPrincipalId),
			slog.String("subscriptionId", server.SubscriptionId),
		)
		return errors.New("userPrincipalName, userPrincipalId or FdpoUserPrincipalId, and subscriptionId are all required")
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

func (s *serverService) UserAssignedIdentity(server *entity.Server) error {

	if server.Version == "V2" {
		return nil
	}

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

func (s *serverService) VerifyActlabsAccess(server *entity.Server) error {
	if server.Version != "V2" {
		return nil
	}

	ok, err := s.serverRepository.IsActlabsAuthorized(*server)
	if err != nil {
		slog.Error("failed to verify if actlabs is authorized to access subscription:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("serverVersion", server.Version),
			slog.String("error", err.Error()),
		)
		return errors.New("failed to verify if actlabs is authorized to access subscription")
	}

	if !ok {
		slog.Error("actlabs is not authorized to access subscription:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("serverVersion", server.Version),
		)
		return errors.New("actlabs does not have permissions to access subscription. Please ask Owner to register the subscription")
	}

	return nil
}
