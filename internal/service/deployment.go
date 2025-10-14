package service

import (
	"context"
	"fmt"
	"time"

	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
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
		logger.LogError(ctx, "failed to get all deployments",
			"error", err,
		)
		return nil, err
	}

	return deployments, err
}

func (d *DeploymentService) GetUserDeployments(ctx context.Context, usePrincipalName string) ([]entity.Deployment, error) {
	deployments, err := d.deploymentRepository.GetUserDeployments(ctx, usePrincipalName)
	if err != nil {
		logger.LogError(ctx, "failed to get user deployments",
			"requested_user_id", usePrincipalName,
			"error", err,
		)
		return nil, err
	}

	return deployments, err
}

func (d *DeploymentService) GetDeployment(ctx context.Context, usePrincipalName string, workspace string, subscriptionId string) (entity.Deployment, error) {
	deployment, err := d.deploymentRepository.GetDeployment(ctx, usePrincipalName, workspace, subscriptionId)
	if err != nil {
		logger.LogError(ctx, "failed to get deployment",
			"requested_user_id", usePrincipalName,
			"workspace", workspace,
			"subscription_id", subscriptionId,
			"error", err,
		)
		return entity.Deployment{}, err
	}

	return deployment, err
}

func (d *DeploymentService) UpsertDeployment(ctx context.Context, deployment entity.Deployment) error {
	if deployment.DeploymentWorkspace == "" || deployment.DeploymentSubscriptionId == "" || deployment.DeploymentUserId == "" {
		logger.LogError(ctx, "workspace or subscription id cannot be empty",
			"workspace", deployment.DeploymentWorkspace,
			"subscription_id", deployment.DeploymentSubscriptionId,
		)
		return fmt.Errorf("userId, workspace or subscription id cant be empty")
	}

	if err := d.deploymentRepository.UpsertDeployment(ctx, deployment); err != nil {
		logger.LogError(ctx, "failed to upsert deployment",
			"workspace", deployment.DeploymentWorkspace,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"error", err,
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
			logger.LogError(ctx, "failed to create warning event",
				"workspace", deployment.DeploymentWorkspace,
				"subscription_id", deployment.DeploymentSubscriptionId,
				"error", err,
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
		logger.LogError(ctx, "failed to create success event",
			"workspace", deployment.DeploymentWorkspace,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"error", err,
		)
	}

	// Add deployment operation entry
	// sending a different context here cause http context will end early while this operation is still running.
	go d.deploymentRepository.DeploymentOperationEntry(context.Background(), deployment)

	return nil
}

func (d *DeploymentService) DeleteDeployment(ctx context.Context, userPrincipalName string, subscriptionId string, workspace string) error {
	// default deployment cant be deleted.
	if workspace == "default" {
		logger.LogError(ctx, "default workspace cannot be deleted",
			"requested_user_id", userPrincipalName,
			"workspace", workspace,
			"subscription_id", subscriptionId,
		)
		return nil
	}

	if err := d.deploymentRepository.DeleteDeployment(ctx, userPrincipalName, subscriptionId, workspace); err != nil {
		logger.LogError(ctx, "failed to delete deployment",
			"requested_user_id", userPrincipalName,
			"workspace", workspace,
			"subscription_id", subscriptionId,
			"error", err,
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
			logger.LogError(ctx, "failed to create warning event",
				"requested_user_id", userPrincipalName,
				"workspace", workspace,
				"subscription_id", subscriptionId,
				"error", err,
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
		logger.LogError(ctx, "failed to create success event",
			"requested_user_id", userPrincipalName,
			"workspace", workspace,
			"subscription_id", subscriptionId,
			"error", err,
		)
	}

	return nil
}

func (d *DeploymentService) MonitorAndAutoDestroyDeployments(ctx context.Context) {
	helper.Recoverer(100, "MonitorAndAutoDestroyDeployments", func() {
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
					logger.LogError(ctx, "failed to poll deployments for auto destruction",
						"error", err,
					)
				}
			}
		}
	})
}

func (d *DeploymentService) PollDeploymentsToBeAutoDestroyed(ctx context.Context) error {
	allDeployments, err := d.deploymentRepository.GetAllDeployments(ctx)
	if err != nil {
		logger.LogError(ctx, "failed to get all deployments for auto destruction check",
			"error", err,
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

			d.deploymentRepository.AutoDestroyDeployment(ctx, deployment.DeploymentUserId, deployment)
		}

	}

	return nil
}
