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
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubManagedServersTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsReadinessTableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubReadinessAssignmentsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsChallengesTableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubChallengesTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsProfilesTableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubProfilesTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsDeploymentsTableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubDeploymentsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsEventsTableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubEventsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	actlabsDeploymentOperationsTableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubDeploymentOperationsTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	return &Auth{
		Cred:                                   cred,
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

func GetTableClient(accountKey string, storageAccountName string, tableName string) (*aztables.Client, error) {
	sharedKeyCred, err := aztables.NewSharedKeyCredential(storageAccountName, accountKey)
	if err != nil {
		return &aztables.Client{}, fmt.Errorf("error creating shared key credential %w", err)
	}

	tableUrl := "https://" + storageAccountName + ".table.core.windows.net/" + tableName

	return aztables.NewClientWithSharedKey(tableUrl, sharedKeyCred, nil)
}
