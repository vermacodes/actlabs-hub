package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/redis/go-redis/v9"

	"github.com/google/uuid"
)

type deploymentRepository struct {
	auth   *auth.Auth
	rdb    *redis.Client
	config *config.Config
}

func NewDeploymentRepository(auth *auth.Auth, rdb *redis.Client, config *config.Config) (entity.DeploymentRepository, error) {
	return &deploymentRepository{
		auth:   auth,
		rdb:    rdb,
		config: config,
	}, nil
}

// don't implement redis caching unless all updates to deployments are done through the api
func (d *deploymentRepository) GetAllDeployments(ctx context.Context) ([]entity.Deployment, error) {
	var deployment entity.Deployment
	deployments := []entity.Deployment{}

	pager := d.auth.ActlabsDeploymentsTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get deployments from table storage",
				"error", err,
			)
			return nil, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			err := json.Unmarshal(entity, &myEntity)
			if err != nil {
				logger.LogError(ctx, "failed to unmarshal deployment entity",
					"error", err,
				)
				return nil, err
			}

			deploymentString := myEntity.Properties["Deployment"].(string)
			if err := json.Unmarshal([]byte(deploymentString), &deployment); err != nil {
				logger.LogError(ctx, "failed to unmarshal deployment",
					"error", err,
				)
				return nil, err
			}

			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

func (d *deploymentRepository) GetUserDeployments(ctx context.Context, userPrincipalName string) ([]entity.Deployment, error) {
	deployment := entity.Deployment{}
	deployments := []entity.Deployment{}

	// check if user deployments already exist in redis
	deploymentsString, err := d.rdb.Get(ctx, userPrincipalName+"-deployments").Result()
	if err != nil {
		// Redis miss is expected, continue to table storage silently
	}
	if deploymentsString != "" {
		if err := json.Unmarshal([]byte(deploymentsString), &deployments); err == nil {
			return deployments, nil
		}
		// If unmarshal fails, continue to table storage silently
	}

	filter := "PartitionKey eq '" + userPrincipalName + "'"

	pager := d.auth.ActlabsDeploymentsTableClient.NewListEntitiesPager(&aztables.ListEntitiesOptions{Filter: &filter})
	pageCount := 1

	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get user deployments from table storage",
				"requested_user_id", userPrincipalName,
				"error", err,
			)
			return nil, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			err := json.Unmarshal(entity, &myEntity)
			if err != nil {
				logger.LogError(ctx, "failed to unmarshal deployment entity from table storage",
					"requested_user_id", userPrincipalName,
					"error", err,
				)
				return nil, err
			}

			deploymentString := myEntity.Properties["Deployment"].(string)
			if err := json.Unmarshal([]byte(deploymentString), &deployment); err != nil {
				logger.LogError(ctx, "failed to unmarshal deployment data from table storage",
					"requested_user_id", userPrincipalName,
					"error", err,
				)
				return nil, err
			}

			deployments = append(deployments, deployment)
		}

		pageCount++
	}

	// save deployments to redis
	marshalledDeployments, err := json.Marshal(deployments)
	if err != nil {
		// Marshaling error is not critical, continue without caching
		return deployments, nil
	}

	err = d.rdb.Set(ctx, userPrincipalName+"-deployments", marshalledDeployments, 0).Err()
	if err != nil {
		// Redis error is not critical, continue without caching
	}

	return deployments, nil
}

func (d *deploymentRepository) GetDeployment(ctx context.Context, userId string, subscriptionId string, workspace string) (entity.Deployment, error) {
	deployment := entity.Deployment{}

	// check if deployment already exist in redis
	deploymentString, err := d.rdb.Get(ctx, userId+"-"+subscriptionId+"-"+workspace).Result()
	if err != nil {
		// Redis miss is expected, continue to table storage silently
	}
	if deploymentString != "" {
		if err := json.Unmarshal([]byte(deploymentString), &deployment); err == nil {
			return deployment, nil
		}
		// If unmarshal fails, continue to table storage silently
	}

	response, err := d.auth.ActlabsDeploymentsTableClient.GetEntity(ctx, userId, userId+"-"+subscriptionId+"-"+workspace, nil)
	if err != nil {
		logger.LogError(ctx, "failed to get deployment from table storage",
			"user_id", userId,
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)
		return entity.Deployment{}, err
	}

	var myEntity aztables.EDMEntity
	err = json.Unmarshal(response.Value, &myEntity)
	if err != nil {
		logger.LogError(ctx, "failed to unmarshal deployment entity from table storage",
			"user_id", userId,
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)
		return entity.Deployment{}, err
	}

	deploymentString = myEntity.Properties["Deployment"].(string)
	if err := json.Unmarshal([]byte(deploymentString), &deployment); err != nil {
		logger.LogError(ctx, "failed to unmarshal deployment data from table storage",
			"user_id", userId,
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)
		return entity.Deployment{}, err
	}

	// save deployment to redis
	marshalledDeployment, err := json.Marshal(deployment)
	if err != nil {
		// Marshaling error is not critical, continue without caching
		return deployment, nil
	}

	err = d.rdb.Set(ctx, userId+"-"+subscriptionId+"-"+workspace, marshalledDeployment, 0).Err()
	if err != nil {
		// Redis error is not critical, continue without caching
	}

	return deployment, nil
}

func (d *deploymentRepository) UpsertDeployment(ctx context.Context, deployment entity.Deployment) error {
	marshalledDeployment, err := json.Marshal(deployment)
	if err != nil {
		logger.LogError(ctx, "failed to marshal deployment for storage",
			"user_id", deployment.DeploymentUserId,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)
		return err
	}

	deploymentEntry := entity.DeploymentEntry{
		Entity: aztables.Entity{
			PartitionKey: deployment.DeploymentUserId,
			RowKey:       deployment.DeploymentId,
		},
		Deployment: string(marshalledDeployment),
	}

	marshalled, err := json.Marshal(deploymentEntry)
	if err != nil {
		logger.LogError(ctx, "failed to marshal deployment entry for table storage",
			"user_id", deployment.DeploymentUserId,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)
		return err
	}

	_, err = d.auth.ActlabsDeploymentsTableClient.UpsertEntity(ctx, marshalled, nil)
	if err != nil {
		logger.LogError(ctx, "failed to upsert deployment in table storage",
			"user_id", deployment.DeploymentUserId,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)
		return err
	}

	// save deployment to redis
	if err := d.rdb.Set(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace, marshalledDeployment, 0).Err(); err != nil {
		// Redis error is not critical, continue without caching

		// if not able to add deployment, delete existing deployment from redis if any
		if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace).Err(); err != nil {
			// Redis deletion error is not critical
		}
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-deployments").Err(); err != nil {
		// Redis deletion error is not critical
	}

	return nil
}

func (d *deploymentRepository) DeploymentOperationEntry(ctx context.Context, deployment entity.Deployment) error {
	marshalledDeploymentLab, err := json.Marshal(deployment.DeploymentLab)
	if err != nil {
		logger.LogError(ctx, "failed to marshal deployment lab for operation entry",
			"user_id", deployment.DeploymentUserId,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)
		return err
	}

	operationEntry := entity.OperationEntry{
		PartitionKey:                 deployment.DeploymentUserId,
		RowKey:                       helper.Generate(64),
		DeploymentUserId:             deployment.DeploymentUserId,
		DeploymentSubscriptionId:     deployment.DeploymentSubscriptionId,
		DeploymentWorkspace:          deployment.DeploymentWorkspace,
		DeploymentStatus:             deployment.DeploymentStatus,
		DeploymentAutoDelete:         deployment.DeploymentAutoDelete,
		DeploymentLifespan:           deployment.DeploymentLifespan,
		DeploymentAutoDeleteUnixTime: deployment.DeploymentAutoDeleteUnixTime,
		DeploymentLab:                string(marshalledDeploymentLab),
	}

	marshalled, err := json.Marshal(operationEntry)
	if err != nil {
		logger.LogError(ctx, "failed to marshal deployment operation entry for table storage",
			"user_id", deployment.DeploymentUserId,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)
		return err
	}

	_, err = d.auth.ActlabSDeploymentOperationsTableClient.UpsertEntity(ctx, marshalled, nil)
	if err != nil {
		logger.LogError(ctx, "failed to upsert deployment operation entry in table storage",
			"user_id", deployment.DeploymentUserId,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)
		return err
	}

	return nil
}

func (d *deploymentRepository) DeleteDeployment(ctx context.Context, userId string, workspace string, subscriptionId string) error {
	_, err := d.auth.ActlabsDeploymentsTableClient.DeleteEntity(ctx, userId, userId+"-"+workspace+"-"+subscriptionId, nil)
	if err != nil {
		logger.LogError(ctx, "failed to delete deployment from table storage",
			"user_id", userId,
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)
		return err
	}

	// delete deployment from redis
	if err := d.rdb.Del(ctx, userId+"-"+subscriptionId+"-"+workspace).Err(); err != nil {
		// Redis deletion error is not critical
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, userId+"-deployments").Err(); err != nil {
		// Redis deletion error is not critical
	}

	return nil
}

func (d *deploymentRepository) AutoDestroyDeployment(ctx context.Context, userPrincipalName string, deployment entity.Deployment) error {

	// http://actlabsserver.com/api/terraform/destroy/operationId
	autoDestroyServiceEndpoint := d.config.ActlabsServerEndpoint + "/api/terraform/destroy/" + uuid.New().String()

	// Marshal deployment to JSON for request body
	deploymentJSON, err := json.Marshal(deployment)
	if err != nil {
		logger.LogError(ctx, "failed to marshal deployment for auto-destroy request",
			"user_id", userPrincipalName,
			"workspace", deployment.DeploymentWorkspace,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"error", err,
		)
		return err
	}

	req, err := http.NewRequest("POST", autoDestroyServiceEndpoint, bytes.NewBuffer(deploymentJSON))
	if err != nil {
		logger.LogError(ctx, "failed to create HTTP request for auto-destroy",
			"user_id", userPrincipalName,
			"workspace", deployment.DeploymentWorkspace,
			"subscription_id", deployment.DeploymentSubscriptionId,
			"error", err,
		)
		return err
	}

	req.Header.Set("x-api-key", d.config.ActlabsServerApiKey)
	req.Header.Set("x-user-id", userPrincipalName)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(ctx, "failed to make http request for auto destroy deployment",
			"error", err,
		)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		logger.LogError(ctx, "http request for auto destroy deployment failed",
			"status_code", resp.StatusCode,
		)
		return err
	}

	return nil
}
