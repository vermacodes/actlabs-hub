package auth

import (
	"actlabs-hub/internal/config"
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

type Auth struct {
	Cred                                   azcore.TokenCredential
	FdpoCredential                         azcore.TokenCredential
	ActlabsServersTableClient              *aztables.Client
	ActlabsReadinessTableClient            *aztables.Client
	ActlabsChallengesTableClient           *aztables.Client
	ActlabsProfilesTableClient             *aztables.Client
	ActlabsDeploymentsTableClient          *aztables.Client
	ActlabsEventsTableClient               *aztables.Client
	ActlabSDeploymentOperationsTableClient *aztables.Client
	StorageAccountKey                      string
}

func NewAuth(appConfig *config.Config) (*Auth, error) {
	var cred azcore.TokenCredential
	var err error

	if appConfig.ActlabsHubUseMsi {
		cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(appConfig.ActlabsHubClientID),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize managed identity auth: %v", err)
		}
	} else {
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize default auth: %v", err)
		}
	}

	fdpoCredential, err := azidentity.NewClientSecretCredential(appConfig.FdpoTenantID, appConfig.ActlabsServerFdpoServicePrincipalClientId, appConfig.ActlabsServerFdpoServicePrincipalSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fdpo auth: %v", err)
	}

	accountKey, err := GetStorageAccountKey(
		appConfig.ActlabsHubSubscriptionID,
		cred,
		appConfig.ActlabsHubResourceGroup,
		appConfig.ActlabsHubStorageAccount,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to get storage account key %w", err)
	}

	actlabsServersTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubManagedServersTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsReadinessTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubReadinessAssignmentsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsChallengesTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubChallengesTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsProfilesTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubProfilesTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsDeploymentsTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubDeploymentsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsEventsTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubEventsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsDeploymentOperationsTableClient, err := GetTableClient(
		cred,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubDeploymentOperationsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	return &Auth{
		Cred:                                   cred,
		FdpoCredential:                         fdpoCredential,
		StorageAccountKey:                      accountKey,
		ActlabsServersTableClient:              actlabsServersTableClient,
		ActlabsReadinessTableClient:            actlabsReadinessTableClient,
		ActlabsChallengesTableClient:           actlabsChallengesTableClient,
		ActlabsProfilesTableClient:             actlabsProfilesTableClient,
		ActlabsDeploymentsTableClient:          actlabsDeploymentsTableClient,
		ActlabsEventsTableClient:               actlabsEventsTableClient,
		ActlabSDeploymentOperationsTableClient: actlabsDeploymentOperationsTableClient,
	}, nil
}

func GetStorageAccountKey(subscriptionId string, cred azcore.TokenCredential, resourceGroup string, storageAccountName string) (string, error) {
	client, err := armstorage.NewAccountsClient(subscriptionId, cred, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.ListKeys(context.Background(), resourceGroup, storageAccountName, nil)
	if err != nil {
		return "", err
	}

	if len(resp.Keys) == 0 {
		return "", fmt.Errorf("no storage account key found")
	}

	return *resp.Keys[0].Value, nil
}

func GetTableClient(cred azcore.TokenCredential, storageAccountName string, tableName string) (*aztables.Client, error) {
	tableUrl := "https://" + storageAccountName + ".table.core.windows.net/" + tableName

	return aztables.NewClient(tableUrl, cred, nil)
}
