package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/redis/go-redis/v9"
)

type serverRepository struct {
	// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential
	auth      *auth.Auth
	appConfig *config.Config
	rdb       *redis.Client
}

func NewServerRepository(
	appConfig *config.Config,
	auth *auth.Auth,
	rdb *redis.Client,
) (entity.ServerRepository, error) {
	return &serverRepository{
		appConfig: appConfig,
		auth:      auth,
		rdb:       rdb,
	}, nil
}

// verify that user is the owner/contributor of the subscription
func (s *serverRepository) IsUserAuthorized(ctx context.Context, server entity.Server) (bool, error) {
	if server.UserAlias == "" {
		return false, errors.New("userId is required")
	}

	if server.SubscriptionId == "" {
		return false, errors.New("subscriptionId is required")
	}

	userPrincipalId := server.UserPrincipalId
	cred := s.auth.Cred
	// Updates for FDPO Environments.
	// The userPrincipalId is different for FDPO environments. This is the object ID in new tenant.
	// The FDPO Credentials are different from the normal credentials.
	if server.Version == "V3" {
		userPrincipalId = server.FdpoUserPrincipalId
		if !s.appConfig.ActlabsHubUseUserAuth && !s.appConfig.ActlabsServerUseMsi {
			cred = s.auth.FdpoCredential
		}
	}

	clientFactory, err := armauthorization.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		return false, err
	}

	filter := "assignedTo('" + userPrincipalId + "')"

	pager := clientFactory.NewRoleAssignmentsClient().NewListForSubscriptionPager(&armauthorization.RoleAssignmentsClientListForSubscriptionOptions{
		Filter:   &filter,
		TenantID: nil,
	})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}
		for _, roleAssignment := range page.Value {

			ownerRoleDefinitionID := "/subscriptions/" + server.SubscriptionId + "/providers" + entity.OwnerRoleDefinitionId
			contributorRoleDefinitionID := "/subscriptions/" + server.SubscriptionId + "/providers" + entity.ContributorRoleDefinitionId

			if *roleAssignment.Properties.PrincipalID == userPrincipalId &&
				*roleAssignment.Properties.Scope == "/subscriptions/"+server.SubscriptionId {
				if *roleAssignment.Properties.RoleDefinitionID == ownerRoleDefinitionID {
					return true, nil
				}
				if *roleAssignment.Properties.RoleDefinitionID == contributorRoleDefinitionID && server.Version == "V2" {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (s *serverRepository) UpsertServerInDatabase(ctx context.Context, server entity.Server) error {
	server.PartitionKey = "actlabs"
	server.RowKey = server.UserPrincipalName

	val, err := json.Marshal(server)
	if err != nil {
		logger.LogError(ctx, "failed to marshal server for database storage",
			"subscription_id", server.SubscriptionId,
			"error", err,
		)
		return err
	}

	_, err = s.auth.ActlabsServersTableClient.UpsertEntity(ctx, val, nil)
	if err != nil {
		logger.LogError(ctx, "failed to upsert server in database",
			"subscription_id", server.SubscriptionId,
			"error", err,
		)
		return err
	}

	return nil
}

func (s *serverRepository) GetServerFromDatabase(ctx context.Context, partitionKey string, rowKey string) (entity.Server, error) {
	response, err := s.auth.ActlabsServersTableClient.GetEntity(ctx, partitionKey, rowKey, nil)
	if err != nil {
		logger.LogError(ctx, "failed to get server from database",
			"error", err,
		)
		return entity.Server{}, err
	}

	server := entity.Server{}
	err = json.Unmarshal(response.Value, &server)
	if err != nil {
		logger.LogError(ctx, "failed to unmarshal server from database",
			"error", err,
		)
		return entity.Server{}, err
	}

	return server, nil

}

func (s *serverRepository) GetAllServersFromDatabase(ctx context.Context) ([]entity.Server, error) {
	servers := []entity.Server{}
	//server := entity.Server{}
	pager := s.auth.ActlabsServersTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get servers from database",
				"error", err,
			)
			return servers, err
		}

		for _, e := range response.Entities {
			var myEntity aztables.EDMEntity
			var server entity.Server
			if err := json.Unmarshal(e, &myEntity); err != nil {
				logger.LogError(ctx, "failed to unmarshal server entity from database",
					"error", err,
				)
				return servers, err
			}
			propertiesBytes, err := json.Marshal(myEntity.Properties)
			if err != nil {
				logger.LogError(ctx, "failed to marshal server properties from database",
					"error", err,
				)
				return servers, err
			}
			if err := json.Unmarshal(propertiesBytes, &server); err != nil {
				logger.LogError(ctx, "failed to unmarshal server properties from database",
					"error", err,
				)
				return servers, err
			}
			servers = append(servers, server)
		}
	}

	return servers, nil
}

func (s *serverRepository) DeleteServerFromDatabase(ctx context.Context, server entity.Server) error {
	_, err := s.auth.ActlabsServersTableClient.DeleteEntity(ctx, server.PartitionKey, server.RowKey, nil)
	if err != nil {
		logger.LogError(ctx, "failed to delete server from database",
			"error", err,
		)
		return err
	}

	return nil
}
