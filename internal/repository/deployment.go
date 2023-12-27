package repository

import (
	"context"
	"encoding/json"
	"errors"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type deploymentRepository struct {
	auth *auth.Auth
	rdb  *redis.Client
}

func NewDeploymentRepository(auth *auth.Auth, rdb *redis.Client) (entity.DeploymentRepository, error) {
	return &deploymentRepository{
		auth: auth,
		rdb:  rdb,
	}, nil
}

// don't implement redis caching unless all updates to deployments are done through the api
func (d *deploymentRepository) GetAllDeployments(ctx context.Context) ([]entity.Deployment, error) {
	slog.Debug("getting all deployments")

	var deployment entity.Deployment
	deployments := []entity.Deployment{}

	pager := d.auth.ActlabsDeploymentsTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			slog.Debug("error getting deployments ", slog.String("error", err.Error()))
			return nil, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			err := json.Unmarshal(entity, &myEntity)
			if err != nil {
				slog.Debug("error unmarshal deployment entity ", slog.String("error", err.Error()))
				return nil, err
			}

			deploymentString := myEntity.Properties["Deployment"].(string)
			if err := json.Unmarshal([]byte(deploymentString), &deployment); err != nil {
				slog.Debug("error unmarshal deployment ", slog.String("error", err.Error()))
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
		slog.Debug("error getting deployments from redis continue to get from table storage ",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)
	}
	if deploymentsString != "" {
		if err := json.Unmarshal([]byte(deploymentsString), &deployments); err == nil {
			slog.Debug("deployments found in redis ",
				slog.String("userPrincipalName", userPrincipalName),
			)

			return deployments, nil
		}
		slog.Debug("error unmarshal deployment found in redis continue to get from table storage ",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)
	}

	filter := "PartitionKey eq '" + userPrincipalName + "'"

	pager := d.auth.ActlabsDeploymentsTableClient.NewListEntitiesPager(&aztables.ListEntitiesOptions{Filter: &filter})
	pageCount := 1

	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			slog.Debug("error getting deployments ",
				slog.String("userPrincipalName", userPrincipalName),
				slog.String("error", err.Error()),
			)
			return nil, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			err := json.Unmarshal(entity, &myEntity)
			if err != nil {
				slog.Debug("error unmarshal deployment entity ",
					slog.String("userPrincipalName", userPrincipalName),
					slog.String("error", err.Error()),
				)
				return nil, err
			}

			deploymentString := myEntity.Properties["Deployment"].(string)
			if err := json.Unmarshal([]byte(deploymentString), &deployment); err != nil {
				slog.Debug("error unmarshal deployment ",
					slog.String("userPrincipalName", userPrincipalName),
					slog.String("error", err.Error()),
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
		slog.Debug("error occurred marshalling the deployments record.",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)

		return deployments, err
	}

	err = d.rdb.Set(ctx, userPrincipalName+"-deployments", marshalledDeployments, 0).Err()
	if err != nil {
		slog.Debug("error occurred saving the deployments record to redis.",
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)
	}

	return deployments, nil
}

func (d *deploymentRepository) GetDeployment(ctx context.Context, userId string, subscriptionId string, workspace string) (entity.Deployment, error) {
	deployment := entity.Deployment{}

	// check if deployment already exist in redis
	deploymentString, err := d.rdb.Get(ctx, userId+"-"+subscriptionId+"-"+workspace).Result()
	if err != nil {
		slog.Debug("error getting deployment from redis continue to get from table storage ",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)
	}
	if deploymentString != "" {
		if err := json.Unmarshal([]byte(deploymentString), &deployment); err == nil {
			slog.Debug("deployment found in redis ",
				slog.String("userId", userId),
				slog.String("subscriptionId", subscriptionId),
				slog.String("workspace", workspace),
			)

			return deployment, nil
		}
		slog.Debug("error unmarshal deployment found in redis continue to get from table storage ",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)
	}

	response, err := d.auth.ActlabsDeploymentsTableClient.GetEntity(ctx, userId, userId+"-"+subscriptionId+"-"+workspace, nil)
	if err != nil {
		slog.Error("error getting deployment ", err)
		return entity.Deployment{}, err
	}

	var myEntity aztables.EDMEntity
	err = json.Unmarshal(response.Value, &myEntity)
	if err != nil {
		slog.Error("error unmarshal deployment entity ", err)
		return entity.Deployment{}, err
	}

	deploymentString = myEntity.Properties["Deployment"].(string)
	if err := json.Unmarshal([]byte(deploymentString), &deployment); err != nil {
		slog.Error("error unmarshal deployment ", err)
		return entity.Deployment{}, err
	}

	// save deployment to redis
	marshalledDeployment, err := json.Marshal(deployment)
	if err != nil {
		slog.Debug("error occurred marshalling the deployment record.",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)

		return deployment, err
	}

	err = d.rdb.Set(ctx, userId+"-"+subscriptionId+"-"+workspace, marshalledDeployment, 0).Err()
	if err != nil {
		slog.Debug("error occurred saving the deployment record to redis.",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)
	}

	return deployment, nil
}

func (d *deploymentRepository) UpsertDeployment(ctx context.Context, deployment entity.Deployment) error {
	marshalledDeployment, err := json.Marshal(deployment)
	if err != nil {
		slog.Debug("error occurred marshalling the deployment record.",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
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
		slog.Debug("error occurred marshalling the deployment record",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = d.auth.ActlabsDeploymentsTableClient.UpsertEntity(ctx, marshalled, nil)
	if err != nil {
		slog.Debug("error adding deployment record ",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)
		return err
	}

	// save deployment to redis
	if err := d.rdb.Set(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace, marshalledDeployment, 0).Err(); err != nil {
		slog.Debug("error occurred saving the deployment record to redis.",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)

		// if not able to add deployment, delete existing deployment from redis if any
		if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace).Err(); err != nil {
			slog.Debug("error occurred deleting the deployment record from redis.",
				slog.String("userId", deployment.DeploymentUserId),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
				slog.String("workspace", deployment.DeploymentWorkspace),
				slog.String("error", err.Error()),
			)

			return err
		}
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-deployments").Err(); err != nil {
		slog.Debug("error occurred deleting the deployments record from redis.",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

func (d *deploymentRepository) DeploymentOperationEntry(ctx context.Context, deployment entity.Deployment) error {
	marshalledDeploymentLab, err := json.Marshal(deployment.DeploymentLab)
	if err != nil {
		slog.Debug("error occurred marshalling the deployment lab",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
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
		slog.Debug("error occurred marshalling the deployment entry record.",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = d.auth.ActlabSDeploymentOperationsTableClient.UpsertEntity(ctx, marshalled, nil)
	if err != nil {
		slog.Debug("error adding deployment entry record ",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (d *deploymentRepository) DeleteDeployment(ctx context.Context, userId string, workspace string, subscriptionId string) error {
	_, err := d.auth.ActlabsDeploymentsTableClient.DeleteEntity(ctx, userId, userId+"-"+workspace+"-"+subscriptionId, nil)
	if err != nil {
		slog.Debug("error deleting deployment record ",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)
		return err
	}

	// delete deployment from redis
	if err := d.rdb.Del(ctx, userId+"-"+subscriptionId+"-"+workspace).Err(); err != nil {
		slog.Debug("error occurred deleting the deployment record from redis.",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)

		return err
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, userId+"-deployments").Err(); err != nil {
		slog.Debug("error occurred deleting the deployments record from redis.",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

func (d *deploymentRepository) GetUserPrincipalNameByMSIPrincipalID(ctx context.Context, msiPrincipalID string) (string, error) {
	// check the cache first
	userPrincipalName, err := d.rdb.Get(ctx, msiPrincipalID+"-owner").Result()
	if err == nil && userPrincipalName != "" {
		return userPrincipalName, nil
	}

	slog.Debug("not able to find user principal name for msi principal id in redis, continue to get from table storage")

	pager := d.auth.ActlabsServersTableClient.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: to.Ptr("managedIdentityPrincipalId eq '" + msiPrincipalID + "'"),
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			slog.Error("not able to get next page", slog.String("error", err.Error()))
			return "", err
		}

		for _, entity := range resp.Entities {
			var myEntity aztables.EDMEntity
			err := json.Unmarshal(entity, &myEntity)
			if err != nil {
				slog.Error("not able to unmarshal entity", slog.String("error", err.Error()))
				return "", err
			}

			userPrincipalName := myEntity.Properties["userPrincipalName"].(string)
			return userPrincipalName, nil
		}
	}

	// save user principal name to redis
	if err := d.rdb.Set(ctx, msiPrincipalID+"-owner", userPrincipalName, 0).Err(); err != nil {
		slog.Debug("error occurred saving the user principal name record to redis.",
			slog.String("msiPrincipalID", msiPrincipalID),
			slog.String("userPrincipalName", userPrincipalName),
			slog.String("error", err.Error()),
		)
	}

	return userPrincipalName, errors.New("not able to find user principal name for msi principal id " + msiPrincipalID)
}
