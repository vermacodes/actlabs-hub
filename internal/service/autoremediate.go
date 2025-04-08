package service

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/helper"
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"golang.org/x/exp/slog"
)

type AutoRemediateService interface {
	IsNetworkAccessDisabled(ctx context.Context) (bool, error)
	EnablePublicNetworkAccess(ctx context.Context) error
	MonitorAndRemediate(ctx context.Context)
}

type autoRemediateService struct {
	auth      *auth.Auth
	appConfig *config.Config
}

func NewAutoRemediateService(appConfig *config.Config, auth *auth.Auth) AutoRemediateService {
	return &autoRemediateService{
		appConfig: appConfig,
		auth:      auth,
	}
}

func (a *autoRemediateService) MonitorAndRemediate(ctx context.Context) {
	helper.Recoverer(100, "MonitorAndRemediate", func() {
		ticker := time.NewTicker(time.Duration(a.appConfig.ActlabsHubAutoRemediatePollingIntervalSeconds) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Context was cancelled or the application finished, so stop the goroutine
				return
			case <-ticker.C:
				// Check for Remediation
				networkAccessDisabled, err := a.IsNetworkAccessDisabled(ctx)
				if err != nil {
					slog.Error("not able to check network access",
						slog.String("error", err.Error()),
					)
					continue
				}
				if networkAccessDisabled {
					slog.Info("Network access is disabled, enabling it now")
					if err := a.EnablePublicNetworkAccess(ctx); err != nil {
						slog.Error("not able to enable public network access",
							slog.String("error", err.Error()),
						)
						continue
					}
					slog.Info("Public network access enabled successfully")
				}
			}
		}
	})
}

// IsNetworkAccessDisabled checks if network access is disabled for a given storage account.
func (a *autoRemediateService) IsNetworkAccessDisabled(ctx context.Context) (bool, error) {

	// Create a storage account client
	client, err := armstorage.NewAccountsClient(a.appConfig.ActlabsHubSubscriptionID, a.auth.Cred, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create storage accounts client: %w", err)
	}

	// Get the storage account properties
	account, err := client.GetProperties(ctx, a.appConfig.ActlabsHubResourceGroup, a.appConfig.ActlabsHubStorageAccount, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get storage account: %w", err)
	}

	if account.Properties != nil && account.Properties.PublicNetworkAccess != nil {
		if string(*account.Properties.PublicNetworkAccess) == "Enabled" && string(*account.Properties.NetworkRuleSet.DefaultAction) == "Allow" {
			return false, nil
		}
	}

	return true, nil
}

// Enable public network access for a storage account.
func (a *autoRemediateService) EnablePublicNetworkAccess(ctx context.Context) error {
	// Create a storage account client
	client, err := armstorage.NewAccountsClient(a.appConfig.ActlabsHubSubscriptionID, a.auth.Cred, nil)
	if err != nil {
		return fmt.Errorf("failed to create storage accounts client: %w", err)
	}

	// Update the storage account properties
	_, err = client.Update(ctx, a.appConfig.ActlabsHubResourceGroup, a.appConfig.ActlabsHubStorageAccount, armstorage.AccountUpdateParameters{
		Properties: &armstorage.AccountPropertiesUpdateParameters{
			PublicNetworkAccess: to.Ptr(armstorage.PublicNetworkAccessEnabled),
			NetworkRuleSet: &armstorage.NetworkRuleSet{
				DefaultAction: to.Ptr(armstorage.DefaultActionAllow),
			},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to update storage account: %w", err)
	}

	return nil
}
