package auth

import (
	"actlabs-hub/internal/config"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
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
		Cred:           cred,
		FdpoCredential: fdpoCredential,
		// StorageAccountKey:                      accountKey,
		ActlabsServersTableClient:              actlabsServersTableClient,
		ActlabsReadinessTableClient:            actlabsReadinessTableClient,
		ActlabsChallengesTableClient:           actlabsChallengesTableClient,
		ActlabsProfilesTableClient:             actlabsProfilesTableClient,
		ActlabsDeploymentsTableClient:          actlabsDeploymentsTableClient,
		ActlabsEventsTableClient:               actlabsEventsTableClient,
		ActlabSDeploymentOperationsTableClient: actlabsDeploymentOperationsTableClient,
	}, nil
}

func GetTableClient(cred azcore.TokenCredential, storageAccountName string, tableName string) (*aztables.Client, error) {
	tableUrl := "https://" + storageAccountName + ".table.core.windows.net/" + tableName

	// Use this for local emulator
	if storageAccountName == "devstoreaccount1" {
		tableUrl = "https://127.0.0.1:10002/" + storageAccountName + "/" + tableName
	}

	return aztables.NewClient(tableUrl, cred, nil)
}
