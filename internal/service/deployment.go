package service

import (
	"context"
	"fmt"
	"time"

	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"

	"golang.org/x/exp/slog"
)

type DeploymentService struct {
	deploymentRepository entity.DeploymentRepository
	serverService        entity.ServerService
	eventService         entity.EventService
	appConfig            *config.Config
}

func NewDeploymentService(
	deploymentRepo entity.DeploymentRepository,
	serverService entity.ServerService,
	eventService entity.EventService,
	appConfig *config.Config,
) entity.DeploymentService {
	return &DeploymentService{
		deploymentRepository: deploymentRepo,
		serverService:        serverService,
		eventService:         eventService,
		appConfig:            appConfig,
	}
}

func (d *DeploymentService) GetAllDeployments(ctx context.Context) ([]entity.Deployment, error) {
	deployments, err := d.deploymentRepository.GetAllDeployments(ctx)
	if err != nil {
		slog.Error("not able to get all deployments",
			slog.String("error", err.Error()),
		)

		return nil, err
	}

	return deployments, err
}

func (d *DeploymentService) GetUserDeployments(ctx context.Context, usePrincipalName string) ([]entity.Deployment, error) {
	slog.Info("getting user deployments",
		slog.String("userPrincipalName", usePrincipalName),
	)

	deployments, err := d.deploymentRepository.GetUserDeployments(ctx, usePrincipalName)
	if err != nil {
		slog.Error("not able to get user deployments",
			slog.String("userPrincipalName", usePrincipalName),
			slog.String("error", err.Error()),
		)

		return nil, err
	}

	return deployments, err
}

func (d *DeploymentService) GetDeployment(ctx context.Context, usePrincipalName string, workspace string, subscriptionId string) (entity.Deployment, error) {
	slog.Info("getting deployment",
		slog.String("userPrincipalName", usePrincipalName),
		slog.String("workspace", workspace),
		slog.String("subscriptionId", subscriptionId),
	)

	deployment, err := d.deploymentRepository.GetDeployment(ctx, usePrincipalName, workspace, subscriptionId)
	if err != nil {
		slog.Error("not able to get deployment",
			slog.String("userPrincipalName", usePrincipalName),
			slog.String("workspace", workspace),
			slog.String("subscriptionId", subscriptionId),
			slog.String("error", err.Error()),
		)

		return entity.Deployment{}, err
	}

	return deployment, err
}

func (d *DeploymentService) UpsertDeployment(ctx context.Context, deployment entity.Deployment) error {
	slog.Info("upserting deployment",
		slog.String("userPrincipalName", deployment.DeploymentUserId),
		slog.String("deploymentWorkspace", deployment.DeploymentWorkspace),
		slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
	)

	if deployment.DeploymentWorkspace == "" || deployment.DeploymentSubscriptionId == "" || deployment.DeploymentUserId == "" {
		slog.Error("workspace or subscription id cant be empty",
			slog.String("userPrincipalName", deployment.DeploymentUserId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
		)

		return fmt.Errorf("userId, workspace or subscription id cant be empty")
	}

	if err := d.deploymentRepository.UpsertDeployment(ctx, deployment); err != nil {
		slog.Error("not able to upsert deployment",
			slog.String("userPrincipalName", deployment.DeploymentUserId),
			slog.String("deploymentWorkspace", deployment.DeploymentWorkspace),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("error", err.Error()),
		)

		// Create Event
		if err := d.eventService.CreateEvent(ctx, entity.Event{
			TimeStamp: time.Now().Format(time.RFC3339),
			Type:      "Warning",
			Reason:    "DeploymentUpdateFailed",
			Message:   fmt.Sprintf("Update of deployment of user %s for subscription %s with workspace %s failed.", deployment.DeploymentUserId, deployment.DeploymentSubscriptionId, deployment.DeploymentWorkspace),
			Reporter:  "actlabs-hub",
			Object:    deployment.DeploymentUserId,
		}); err != nil {
			slog.Error("not able to create event",
				slog.String("userPrincipalName", deployment.DeploymentUserId),
				slog.String("workspace", deployment.DeploymentWorkspace),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
				slog.String("error", err.Error()),
			)
		}

		return err
	}

	// Create Event
	if err := d.eventService.CreateEvent(ctx, entity.Event{
		TimeStamp: time.Now().Format(time.RFC3339),
		Type:      "Normal",
		Reason:    "DeploymentUpdated",
		Message:   fmt.Sprintf("Deployment of user %s for subscription %s with workspace %s is updated/created.", deployment.DeploymentUserId, deployment.DeploymentSubscriptionId, deployment.DeploymentWorkspace),
		Reporter:  "actlabs-hub",
		Object:    deployment.DeploymentUserId,
	}); err != nil {
		slog.Error("not able to create event",
			slog.String("userPrincipalName", deployment.DeploymentUserId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("error", err.Error()),
		)
	}

	// Add deployment operation entry
	// sending a different context here cause http context will end early while this operation is still running.
	go d.deploymentRepository.DeploymentOperationEntry(context.Background(), deployment)

	return nil
}

func (d *DeploymentService) DeleteDeployment(ctx context.Context, userPrincipalName string, subscriptionId string, workspace string) error {
	slog.Info("deleting deployment",
		slog.String("userPrincipalName", userPrincipalName),
		slog.String("workspace", workspace),
		slog.String("subscriptionId", subscriptionId),
	)

	// default deployment cant be deleted.
	if workspace == "default" {
		slog.Error("default workspace cant be deleted",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("workspace", workspace),
			slog.String("subscriptionId", subscriptionId),
		)
		return nil
	}

	if err := d.deploymentRepository.DeleteDeployment(ctx, userPrincipalName, subscriptionId, workspace); err != nil {
		slog.Error("not able to delete deployment",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("workspace", workspace),
			slog.String("subscriptionId", subscriptionId),
			slog.String("error", err.Error()),
		)

		// Create Event
		if err := d.eventService.CreateEvent(ctx, entity.Event{
			TimeStamp: time.Now().Format(time.RFC3339),
			Type:      "Warning",
			Reason:    "DeploymentDeleteFailed",
			Message:   fmt.Sprintf("Delete operation of deployment of user %s for subscription %s with workspace %s failed.", userPrincipalName, subscriptionId, workspace),
			Reporter:  "actlabs-hub",
			Object:    userPrincipalName,
		}); err != nil {
			slog.Error("not able to create event",
				slog.String("userPrincipalName", userPrincipalName),
				slog.String("workspace", workspace),
				slog.String("subscriptionId", subscriptionId),
				slog.String("error", err.Error()),
			)
		}

		return err
	}

	// Create Event
	if err := d.eventService.CreateEvent(ctx, entity.Event{
		TimeStamp: time.Now().Format(time.RFC3339),
		Type:      "Normal",
		Reason:    "DeploymentDeleted",
		Message:   fmt.Sprintf("Deployment of user %s for subscription %s with workspace %s is deleted successfully.", userPrincipalName, subscriptionId, workspace),
		Reporter:  "actlabs-hub",
		Object:    userPrincipalName,
	}); err != nil {
		slog.Error("not able to create event",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("workspace", workspace),
			slog.String("subscriptionId", subscriptionId),
			slog.String("error", err.Error()),
		)
	}

	return nil
}

func (d *DeploymentService) MonitorAndDeployAutoDestroyedServersToDestroyPendingDeployments(ctx context.Context) {
	helper.Recoverer(100, "MonitorAndDeployAutoDestroyedServersToDestroyPendingDeployments", func() {
		ticker := time.NewTicker(time.Duration(d.appConfig.ActlabsHubDeploymentsPollingIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled or the application finished, so stop the goroutine
				return
			case <-ticker.C:
				// Every minute, check for servers to destroy
				if err := d.PollDeploymentsToBeAutoDestroyed(ctx); err != nil {
					slog.Error("not able to deploy auto destroyed servers to destroy pending deployments",
						slog.String("error", err.Error()),
					)
				}
			}
		}
	})
}

func (d *DeploymentService) PollDeploymentsToBeAutoDestroyed(ctx context.Context) error {
	slog.Info("polling for deployments to be destroyed")
	allDeployments, err := d.deploymentRepository.GetAllDeployments(ctx)
	if err != nil {
		slog.Error("not able to get all deployments",
			slog.String("error", err.Error()),
		)
		return err
	}

	for _, deployment := range allDeployments {
		if !deployment.DeploymentAutoDelete {
			continue
		}

		currentEpochTime := time.Now().Unix()

		if deployment.DeploymentAutoDeleteUnixTime < currentEpochTime &&
			deployment.DeploymentAutoDeleteUnixTime != 0 &&
			(deployment.DeploymentStatus == entity.DeploymentCompleted ||
				deployment.DeploymentStatus == entity.DeploymentFailed) {

			slog.Info("redeploying server to destroy pending deployment",
				slog.String("userPrincipalName", deployment.DeploymentUserId),
				slog.String("workspace", deployment.DeploymentWorkspace),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			)

			go d.RedeployServer(ctx, deployment)
		}

	}

	return nil
}

func (d *DeploymentService) RedeployServer(ctx context.Context, deployment entity.Deployment) {

	server, err := d.serverService.GetServer(deployment.DeploymentUserId)
	if err != nil {
		slog.Error("not able to get server",
			slog.String("userPrincipalName", deployment.DeploymentUserId),
		)
		return
	}

	if server.Status == entity.ServerStatusAutoDestroyed &&
		server.SubscriptionId == deployment.DeploymentSubscriptionId {
		// deploy server again.
		server, err := d.serverService.DeployServer(server)
		if err != nil {
			slog.Error("not able to deploy server",
				slog.String("userPrincipalName", deployment.DeploymentUserId),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
				slog.String("error", err.Error()),
			)

			return
		}

		// ensure server status is running.
		if server.Status != entity.ServerStatusRunning {
			slog.Error("not able to deploy server",
				slog.String("userPrincipalName", deployment.DeploymentUserId),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
				slog.String("error", err.Error()),
			)

			return
		}

		// update activity so that server stays alive for at least 15 minutes.
		if err := d.serverService.UpdateActivityStatus(server.UserPrincipalName); err != nil {
			slog.Error("not able to update activity status",
				slog.String("userPrincipalName", deployment.DeploymentUserId),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
				slog.String("error", err.Error()),
			)
		}
	}
}

func (d *DeploymentService) GetUserPrincipalNameByMSIPrincipalID(ctx context.Context, msiPrincipalID string) (string, error) {
	slog.Info("getting user principal name by msi principal id",
		slog.String("msiPrincipalID", msiPrincipalID),
	)

	userPrincipalName, err := d.deploymentRepository.GetUserPrincipalNameByMSIPrincipalID(ctx, msiPrincipalID)
	if err != nil {
		slog.Error("not able to get user principal name by msi principal id",
			slog.String("msiPrincipalID", msiPrincipalID),
			slog.String("error", err.Error()),
		)

		return "", err
	}

	return userPrincipalName, nil
}
