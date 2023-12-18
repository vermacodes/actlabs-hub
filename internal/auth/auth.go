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
	Cred                      azcore.TokenCredential
	ActlabsServersTableClient *aztables.Client
	StorageAccountKey         string
}

func NewAuth(appConfig *config.Config) (*Auth, error) {
	var cred azcore.TokenCredential
	var err error

	if appConfig.UseMsi {
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

	tableClient, err := GetTableClient(
		accountKey,
		appConfig.ActlabsHubStorageAccount,
		appConfig.ActlabsHubManagedServersTableName,
	)
	if err != nil {
		return nil, fmt.Errorf("not able to create table client %w", err)
	}

	return &Auth{
		Cred:                      cred,
		StorageAccountKey:         accountKey,
		ActlabsServersTableClient: tableClient,
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
