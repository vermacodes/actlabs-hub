package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appcontainers/armappcontainers/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
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

func (s *serverRepository) GetAzureContainerGroup(server entity.Server) (entity.Server, error) {
	ctx := context.Background()
	clientFactory, err := armcontainerinstance.NewContainerGroupsClient(server.SubscriptionId, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	res, err := clientFactory.Get(ctx, server.ResourceGroup, server.UserAlias+"-aci", nil)
	if err != nil {
		slog.Debug("failed to finish the request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	server.Endpoint = *res.Properties.IPAddress.Fqdn
	server.Status = s.ParseServerStatus(*res.Properties.ProvisioningState)

	return server, nil
}

func (s *serverRepository) GetUserAssignedManagedIdentity(server entity.Server) (entity.Server, error) {

	// Not needed in V3 servers.
	if server.Version == "V3" {
		return server, nil
	}

	ctx := context.Background()
	clientFactory, err := armmsi.NewClientFactory(server.SubscriptionId, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	res, err := clientFactory.NewUserAssignedIdentitiesClient().Get(ctx, server.ResourceGroup, server.UserAlias+"-msi", nil)
	if err != nil {
		slog.Debug("failed to finish the request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	server.ManagedIdentityClientId = *res.Properties.ClientID
	server.ManagedIdentityPrincipalId = *res.Properties.PrincipalID
	server.ManagedIdentityResourceId = *res.ID

	return server, nil
}

// https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/list-by-resource-group?view=rest-storagerp-2023-01-01&tabs=Go
func (s *serverRepository) GetClientStorageAccount(server entity.Server) (armstorage.Account, error) {

	// Check if the storage account is already cached in Redis
	storageAccountStr, err := s.rdb.Get(context.Background(), server.UserAlias+"-storageAccount").Result()
	if err == nil {
		var storageAccount armstorage.Account
		err = json.Unmarshal([]byte(storageAccountStr), &storageAccount)
		if err != nil {
			return armstorage.Account{}, err
		}
		return storageAccount, nil
	}

	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth && !s.appConfig.ActlabsServerUseMsi {
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armstorage.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		slog.Debug("not able to create client factory to get storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return armstorage.Account{}, err
	}

	pager := clientFactory.NewAccountsClient().NewListByResourceGroupPager("repro-project", nil)
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("not able to get next page for storage account",
				slog.String("userPrincipalName", server.UserPrincipalName),
				slog.String("subscriptionId", server.SubscriptionId),
				slog.String("error", err.Error()),
			)
			return armstorage.Account{}, err
		}
		for _, account := range page.Value {
			// Cache storage account in Redis
			storageAccountStr, err := json.Marshal(account)
			if err != nil {
				slog.Debug("not able to marshal storage account",
					slog.String("userPrincipalName", server.UserPrincipalName),
					slog.String("subscriptionId", server.SubscriptionId),
					slog.String("error", err.Error()),
				)
			}
			err = s.rdb.Set(context.Background(), server.UserAlias+"-storageAccount", storageAccountStr, 0).Err()
			if err != nil {
				slog.Debug("not able to set storage account in redis",
					slog.String("userPrincipalName", server.UserPrincipalName),
					slog.String("subscriptionId", server.SubscriptionId),
					slog.String("error", err.Error()),
				)
			}

			return *account, nil // return the first storage account found.
		}
	}

	return armstorage.Account{}, errors.New("storage account not found in resource group repro-project")
}

func (s *serverRepository) GetClientStorageAccountKey(server entity.Server) (string, error) {
	// Check if the storage account key is already cached in Redis
	storageAccountKey, err := s.rdb.Get(context.Background(), server.UserAlias+"-storageAccountKey").Result()
	if err == nil {
		return storageAccountKey, nil
	}

	storageAccount, err := s.GetClientStorageAccount(server)
	if err != nil {
		return "", err
	}

	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth && !s.appConfig.ActlabsServerUseMsi {
		cred = s.auth.FdpoCredential
	}

	client, err := armstorage.NewAccountsClient(server.SubscriptionId, cred, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.ListKeys(context.Background(), server.ResourceGroup, *storageAccount.Name, nil)
	if err != nil {
		return "", err
	}

	if len(resp.Keys) == 0 {
		return "", errors.New("no storage account key found")
	}

	// Cache storage account key in Redis
	err = s.rdb.Set(context.Background(), server.UserAlias+"-storageAccountKey", *resp.Keys[0].Value, 0).Err()
	if err != nil {
		slog.Debug("not able to set storage account key in redis",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
	}

	return *resp.Keys[0].Value, nil
}

func (s *serverRepository) DeployServer(server entity.Server) (entity.Server, error) {

	ctx := context.Background()

	managedEnvironmentId, err := s.GetNextManagedEnvironmentId(server)
	if err != nil {
		return server, err
	}

	clientFactory, err := armappcontainers.NewClientFactory(s.appConfig.ActlabsHubSubscriptionID, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	tenantID := s.appConfig.TenantID

	if server.Version == "V3" {
		tenantID = s.appConfig.FdpoTenantID
	}

	// Defaults to service principal, if not using MSI
	clientId := s.appConfig.ActlabsServerFdpoServicePrincipalClientId
	if s.appConfig.ActlabsServerUseMsi {
		clientId = s.appConfig.ActlabsHubClientID
	}

	poller, err := clientFactory.NewContainerAppsClient().BeginCreateOrUpdate(ctx, s.appConfig.ActlabsServerResourceGroup, server.UserAlias+"-app", armappcontainers.ContainerApp{
		Location: to.Ptr("eastus"),
		Identity: &armappcontainers.ManagedServiceIdentity{
			Type: to.Ptr(armappcontainers.ManagedServiceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armappcontainers.UserAssignedIdentity{
				s.appConfig.ActlabsHubManagedIdentityResourceId: {},
			},
		},
		Properties: &armappcontainers.ContainerAppProperties{
			ManagedEnvironmentID: to.Ptr(managedEnvironmentId),
			Configuration: &armappcontainers.Configuration{
				Ingress: &armappcontainers.Ingress{
					External:   to.Ptr(true),
					TargetPort: to.Ptr(int32(s.appConfig.ActlabsServerPort)),
				},
			},
			Template: &armappcontainers.Template{
				Scale: &armappcontainers.Scale{
					MaxReplicas: to.Ptr(int32(1)),
					MinReplicas: to.Ptr(int32(1)),
				},
				Containers: []*armappcontainers.Container{
					{
						Name:  to.Ptr("actlabs"),
						Image: to.Ptr(s.appConfig.ActlabsServerImage),

						Env: []*armappcontainers.EnvironmentVar{
							{
								Name:  to.Ptr("USE_SERVICE_PRINCIPAL"),
								Value: to.Ptr(strconv.FormatBool(s.appConfig.ActlabsServerUseServicePrincipal)),
							},
							{
								Name:  to.Ptr("ARM_USE_MSI"),
								Value: to.Ptr(strconv.FormatBool(s.appConfig.ActlabsServerUseMsi)),
							},
							{
								Name:  to.Ptr("ARM_CLIENT_ID"),
								Value: to.Ptr(clientId),
							},
							{
								Name:  to.Ptr("USE_MSI"),
								Value: to.Ptr(strconv.FormatBool(s.appConfig.ActlabsServerUseMsi)),
							},
							{
								Name:  to.Ptr("ACTLABS_HUB_SUBSCRIPTION_ID"),
								Value: to.Ptr(s.appConfig.ActlabsHubSubscriptionID),
							},
							{
								Name:  to.Ptr("ACTLABS_HUB_RESOURCE_GROUP_NAME"),
								Value: to.Ptr(s.appConfig.ActlabsHubResourceGroup),
							},
							{
								Name:  to.Ptr("ACTLABS_HUB_STORAGE_ACCOUNT_NAME"),
								Value: to.Ptr(s.appConfig.ActlabsHubStorageAccount),
							},
							{
								Name:  to.Ptr("ACTLABS_HUB_URL"),
								Value: to.Ptr(s.appConfig.ActlabsHubURL),
							},
							{
								Name:  to.Ptr("PORT"),
								Value: to.Ptr(strconv.Itoa(int(s.appConfig.ActlabsServerPort))),
							},
							{
								Name:  to.Ptr("ROOT_DIR"),
								Value: to.Ptr(s.appConfig.ActlabsServerRootDir),
							},
							{
								Name:  to.Ptr("AZURE_CLIENT_ID"),
								Value: to.Ptr(s.appConfig.ActlabsServerFdpoServicePrincipalClientId),
							},
							{
								Name:  to.Ptr("AZURE_CLIENT_SECRET"),
								Value: to.Ptr(s.appConfig.ActlabsServerFdpoServicePrincipalSecret),
							},
							{
								Name:  to.Ptr("AZURE_TENANT_ID"),
								Value: to.Ptr(tenantID),
							},
							{
								Name:  to.Ptr("ARM_SUBSCRIPTION_ID"),
								Value: &server.SubscriptionId,
							},
							{
								Name:  to.Ptr("AZURE_SUBSCRIPTION_ID"),
								Value: &server.SubscriptionId,
							},
							{
								Name:  to.Ptr("ARM_TENANT_ID"),
								Value: to.Ptr(tenantID),
							},
							{
								Name:  to.Ptr("ARM_USER_PRINCIPAL_NAME"),
								Value: to.Ptr(server.UserPrincipalName),
							},
							{
								Name:  to.Ptr("LOG_LEVEL"),
								Value: to.Ptr(server.LogLevel),
							},
							{
								Name:  to.Ptr("AUTH_TOKEN_ISS"),
								Value: to.Ptr(s.appConfig.AuthTokenIss),
							},
							{
								Name:  to.Ptr("AUTH_TOKEN_AUD"),
								Value: to.Ptr(s.appConfig.AuthTokenAud),
							},
							{
								Name:  to.Ptr("USER_ALIAS"),
								Value: to.Ptr(server.UserAlias),
							},
							{
								Name:  to.Ptr("MISE_ENDPOINT"),
								Value: to.Ptr(s.appConfig.MiseEndpoint),
							},
							{
								Name:  to.Ptr("MISE_VERBOSE_LOGGING"),
								Value: to.Ptr(strconv.FormatBool(s.appConfig.MiseVerboseLogging)),
							},
							{
								Name:  to.Ptr("CORS_ALLOW_ORIGINS"),
								Value: to.Ptr(s.appConfig.CorsAllowOrigins),
							},
							{
								Name:  to.Ptr("CORS_ALLOW_METHODS"),
								Value: to.Ptr(s.appConfig.CorsAllowMethods),
							},
							{
								Name:  to.Ptr("CORS_ALLOW_HEADERS"),
								Value: to.Ptr(s.appConfig.CorsAllowHeaders),
							},
							{
								Name:  to.Ptr("AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID"),
								Value: to.Ptr(s.appConfig.ActlabsServerAroRpFirstPartySpID),
							},
							{
								Name:  to.Ptr("APPSETTING_WEBSITE_SITE_NAME"),
								Value: to.Ptr(s.appConfig.ActlabsServerAppSettingWebsiteSiteName),
							},
							{
								Name:  to.Ptr("ARM_MSI_API_VERSION"),
								Value: to.Ptr(s.appConfig.ActlabsServerArmMsiApiVersion),
							},
							{
								Name:  to.Ptr("ARM_MSI_API_PROXY_PORT"),
								Value: to.Ptr(s.appConfig.ActlabsServerArmMsiApiProxyPort),
							},
						},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		slog.Debug("failed to finish the request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		slog.Debug("failed to pull the result:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	server.ContainerAppFqdn = *resp.Properties.Configuration.Ingress.Fqdn
	server.Status = s.ParseServerStatus(string(*resp.Properties.ProvisioningState))

	return server, nil
}

func (s *serverRepository) AddApplicationGatewayConfigForUser(ctx context.Context, server entity.Server) (entity.Server, error) {

	clientFactory, err := armnetwork.NewClientFactory(s.appConfig.ActlabsHubSubscriptionID, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	agClient := clientFactory.NewApplicationGatewaysClient()
	existing, err := agClient.Get(ctx, s.appConfig.ActlabsHubResourceGroup, s.appConfig.ActlabsAppGatewayName, nil)
	if err != nil {
		slog.Debug("failed to get existing application gateway:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	if existing.Properties == nil {
		existing.Properties = &armnetwork.ApplicationGatewayPropertiesFormat{}
	}

	// backend pool to add
	pool := &armnetwork.ApplicationGatewayBackendAddressPool{
		Name: to.Ptr("appgw-backend-pool-" + server.UserAlias),
		Properties: &armnetwork.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: []*armnetwork.ApplicationGatewayBackendAddress{
				{Fqdn: to.Ptr(server.ContainerAppFqdn)},
			},
		},
	}
	if !existsByName(existing.Properties.BackendAddressPools, *pool.Name, func(p *armnetwork.ApplicationGatewayBackendAddressPool) *string { return p.Name }) {
		existing.Properties.BackendAddressPools = append(existing.Properties.BackendAddressPools, pool)
	}

	var poolRef *armnetwork.SubResource
	if pool.ID != nil {
		poolRef = &armnetwork.SubResource{ID: pool.ID}
	} else if existing.ID != nil && pool.Name != nil {
		poolRef = &armnetwork.SubResource{ID: to.Ptr(*existing.ID + "/backendAddressPools/" + *pool.Name)}
	}

	// probe
	probe := &armnetwork.ApplicationGatewayProbe{
		Name: to.Ptr("appgw-probe-" + server.UserAlias),
		Properties: &armnetwork.ApplicationGatewayProbePropertiesFormat{
			Protocol:                            to.Ptr(armnetwork.ApplicationGatewayProtocolHTTPS),
			Path:                                to.Ptr("/status"),
			Interval:                            to.Ptr[int32](30),
			Timeout:                             to.Ptr(int32(20)),
			Match:                               &armnetwork.ApplicationGatewayProbeHealthResponseMatch{StatusCodes: []*string{to.Ptr("200")}},
			PickHostNameFromBackendHTTPSettings: to.Ptr(true),
		},
	}
	if !existsByName(existing.Properties.Probes, *probe.Name, func(p *armnetwork.ApplicationGatewayProbe) *string { return p.Name }) {
		existing.Properties.Probes = append(existing.Properties.Probes, probe)
	}

	var probeRef *armnetwork.SubResource
	if probe.ID != nil {
		probeRef = &armnetwork.SubResource{ID: probe.ID}
	} else if existing.ID != nil && probe.Name != nil {
		probeRef = &armnetwork.SubResource{ID: to.Ptr(*existing.ID + "/probes/" + *probe.Name)}
	}

	// http settings
	httpSetting := &armnetwork.ApplicationGatewayBackendHTTPSettings{
		Name: to.Ptr("appgw-backend-http-settings-" + server.UserAlias),
		Properties: &armnetwork.ApplicationGatewayBackendHTTPSettingsPropertiesFormat{
			Port:                           to.Ptr[int32](443),
			Protocol:                       to.Ptr(armnetwork.ApplicationGatewayProtocolHTTPS),
			RequestTimeout:                 to.Ptr[int32](60),
			Probe:                          probeRef,
			CookieBasedAffinity:            to.Ptr(armnetwork.ApplicationGatewayCookieBasedAffinityDisabled),
			PickHostNameFromBackendAddress: to.Ptr(true),
		},
	}
	if !existsByName(existing.Properties.BackendHTTPSettingsCollection, *httpSetting.Name, func(h *armnetwork.ApplicationGatewayBackendHTTPSettings) *string { return h.Name }) {
		existing.Properties.BackendHTTPSettingsCollection = append(existing.Properties.BackendHTTPSettingsCollection, httpSetting)
	}

	var httpSettingRef *armnetwork.SubResource
	if httpSetting.ID != nil {
		httpSettingRef = &armnetwork.SubResource{ID: httpSetting.ID}
	} else if existing.ID != nil && httpSetting.Name != nil {
		httpSettingRef = &armnetwork.SubResource{ID: to.Ptr(*existing.ID + "/backendHttpSettingsCollection/" + *httpSetting.Name)}
	}

	// rewrite ruleset
	rewrite := &armnetwork.ApplicationGatewayRewriteRuleSet{
		Name: to.Ptr("appgw-rewrite-ruleset-" + server.UserAlias),
		Properties: &armnetwork.ApplicationGatewayRewriteRuleSetPropertiesFormat{
			RewriteRules: []*armnetwork.ApplicationGatewayRewriteRule{
				{
					Name:         to.Ptr("strip-path-rule-" + server.UserAlias),
					RuleSequence: to.Ptr[int32](110),
					Conditions: []*armnetwork.ApplicationGatewayRewriteRuleCondition{
						{
							Variable:   to.Ptr("var_uri_path"),
							Pattern:    to.Ptr("/" + server.UserAlias + "(.*)"),
							IgnoreCase: to.Ptr(true),
						},
					},
					ActionSet: &armnetwork.ApplicationGatewayRewriteRuleActionSet{
						URLConfiguration: &armnetwork.ApplicationGatewayURLConfiguration{ModifiedPath: to.Ptr("/{var_uri_path_1}")},
					},
				},
			},
		},
	}
	if !existsByName(existing.Properties.RewriteRuleSets, *rewrite.Name, func(r *armnetwork.ApplicationGatewayRewriteRuleSet) *string { return r.Name }) {
		existing.Properties.RewriteRuleSets = append(existing.Properties.RewriteRuleSets, rewrite)
	}

	var rewriteRef *armnetwork.SubResource
	if rewrite.ID != nil {
		rewriteRef = &armnetwork.SubResource{ID: rewrite.ID}
	} else if existing.ID != nil && rewrite.Name != nil {
		rewriteRef = &armnetwork.SubResource{ID: to.Ptr(*existing.ID + "/rewriteRuleSets/" + *rewrite.Name)}
	}

	// path rule
	pathRule := &armnetwork.ApplicationGatewayPathRule{
		Name: to.Ptr("appgw-path-rule-" + server.UserAlias),
		Properties: &armnetwork.ApplicationGatewayPathRulePropertiesFormat{
			BackendAddressPool:  poolRef,
			BackendHTTPSettings: httpSettingRef,
			Paths:               []*string{to.Ptr("/" + server.UserAlias + "/*")},
			RewriteRuleSet:      rewriteRef,
		},
	}

	if len(existing.Properties.URLPathMaps) > 0 {
		// ensure Properties is non-nil before using PathRules
		if existing.Properties.URLPathMaps[0].Properties == nil {
			existing.Properties.URLPathMaps[0].Properties = &armnetwork.ApplicationGatewayURLPathMapPropertiesFormat{}
		}
		if !existsByName(existing.Properties.URLPathMaps[0].Properties.PathRules, *pathRule.Name, func(r *armnetwork.ApplicationGatewayPathRule) *string { return r.Name }) {
			existing.Properties.URLPathMaps[0].Properties.PathRules = append(existing.Properties.URLPathMaps[0].Properties.PathRules, pathRule)
		}
	} else {
		// If no path map exists, create one
		existing.Properties.URLPathMaps = []*armnetwork.ApplicationGatewayURLPathMap{
			{
				Name: to.Ptr("appgw-url-path-map"),
				Properties: &armnetwork.ApplicationGatewayURLPathMapPropertiesFormat{
					PathRules: []*armnetwork.ApplicationGatewayPathRule{
						pathRule,
					},
					// default backend pool and http settings to avoid validation errors
					// find the default pool with name appgw-backend-pool-not-found-api
					DefaultBackendAddressPool: &armnetwork.SubResource{
						ID: to.Ptr(*existing.ID + "/backendAddressPools/appgw-backend-pool-not-found-api"),
					},
					DefaultBackendHTTPSettings: &armnetwork.SubResource{
						ID: to.Ptr(*existing.ID + "/backendHttpSettingsCollection/appgw-backend-http-setting-not-found-api"),
					},
				},
			},
		}
	}

	// update (reuse the full existing resource)
	poller, err := agClient.BeginCreateOrUpdate(ctx, s.appConfig.ActlabsHubResourceGroup, s.appConfig.ActlabsAppGatewayName, armnetwork.ApplicationGateway{
		Location:   existing.Location,
		Tags:       existing.Tags,
		Identity:   existing.Identity,
		Properties: existing.Properties,
	}, nil)

	if err != nil {
		slog.Error("failed to create application gateway:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		slog.Error("failed to pull the result:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	server.Endpoint = s.appConfig.ActlabsFQDN + "/" + server.UserAlias

	return server, nil
}

func (s *serverRepository) DeleteApplicationGatewayConfigForUser(ctx context.Context, server entity.Server) error {
	clientFactory, err := armnetwork.NewClientFactory(s.appConfig.ActlabsHubSubscriptionID, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	agClient := clientFactory.NewApplicationGatewaysClient()
	existing, err := agClient.Get(ctx, s.appConfig.ActlabsHubResourceGroup, s.appConfig.ActlabsAppGatewayName, nil)
	if err != nil {
		slog.Debug("failed to get existing application gateway:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	if existing.Properties == nil {
		return nil
	}

	// Remove backend pool
	existing.Properties.BackendAddressPools = removeByName(existing.Properties.BackendAddressPools, "appgw-backend-pool-"+server.UserAlias, func(p *armnetwork.ApplicationGatewayBackendAddressPool) *string { return p.Name })

	// Remove probe
	existing.Properties.Probes = removeByName(existing.Properties.Probes, "appgw-probe-"+server.UserAlias, func(p *armnetwork.ApplicationGatewayProbe) *string { return p.Name })

	// Remove http settings
	existing.Properties.BackendHTTPSettingsCollection = removeByName(existing.Properties.BackendHTTPSettingsCollection, "appgw-backend-http-settings-"+server.UserAlias, func(h *armnetwork.ApplicationGatewayBackendHTTPSettings) *string { return h.Name })

	// Remove rewrite ruleset
	existing.Properties.RewriteRuleSets = removeByName(existing.Properties.RewriteRuleSets, "appgw-rewrite-ruleset-"+server.UserAlias, func(r *armnetwork.ApplicationGatewayRewriteRuleSet) *string { return r.Name })

	// Remove path rule
	if len(existing.Properties.URLPathMaps) > 0 {
		existing.Properties.URLPathMaps[0].Properties.PathRules = removeByName(existing.Properties.URLPathMaps[0].Properties.PathRules, "appgw-path-rule-"+server.UserAlias, func(r *armnetwork.ApplicationGatewayPathRule) *string { return r.Name })
	}

	// update (reuse the full existing resource)
	poller, err := agClient.BeginCreateOrUpdate(ctx, s.appConfig.ActlabsHubResourceGroup, s.appConfig.ActlabsAppGatewayName, armnetwork.ApplicationGateway{
		Location:   existing.Location,
		Tags:       existing.Tags,
		Identity:   existing.Identity,
		Properties: existing.Properties,
	}, nil)

	if err != nil {
		slog.Error("failed to update application gateway:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		slog.Error("failed to pull the result:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (s *serverRepository) EnsureServerUp(server entity.Server) error {
	// Call the server endpoint to check if it is up
	serverEndpoint := "https://" + server.Endpoint + s.appConfig.ActlabsServerReadinessProbePath
	slog.Debug("Checking if server is up: " + serverEndpoint)

	resp, err := http.Get(serverEndpoint)
	if err != nil {
		slog.Debug("Failed to make HTTP request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("Server is not up. Status code:", resp.StatusCode)
		return errors.New("server is not up")
	}

	return nil
}

func (s *serverRepository) EnsureServerIdle(server entity.Server) (bool, error) {
	// Call the server endpoint to check if it is up
	serverEndpoint := "https://" + server.Endpoint + "/actionstatus"
	slog.Debug("Checking if server is busy: " + serverEndpoint)

	resp, err := http.Get(serverEndpoint)
	if err != nil {
		slog.Debug("Failed to make HTTP request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("not able to check for action status:", resp.StatusCode)
		return false, errors.New("not able to check for action status")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Debug("not able to read response body:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return false, err
	}

	managedServerActionStatus := entity.ManagedServerActionStatus{}
	if err = json.Unmarshal(body, &managedServerActionStatus); err != nil {
		slog.Debug("not able to unmarshal response body:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return false, err
	}

	if managedServerActionStatus.InProgress {
		return false, nil
	}
	return true, nil
}

func (s *serverRepository) DestroyServer(server entity.Server) error {

	// remove storage account and key from redis
	if err := s.rdb.Del(context.Background(), server.UserAlias+"-storageAccount", server.UserAlias+"-storageAccountKey").Err(); err != nil {
		slog.Debug("failed to delete storage account and key from redis",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("resourceGroup", server.ResourceGroup),
			slog.String("error", err.Error()),
		)
		return err
	}

	if server.Version == "V2" || server.Version == "V3" {
		return s.DestroyAzureContainerApp(server)
	}
	return s.DestroyAzureContainerGroup(server)
}

func (s *serverRepository) DestroyAzureContainerApp(server entity.Server) error {

	ctx := context.Background()

	clientFactory, err := armappcontainers.NewClientFactory(s.appConfig.ActlabsHubSubscriptionID, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	poller, err := clientFactory.NewContainerAppsClient().BeginDelete(ctx, s.appConfig.ActlabsServerResourceGroup, server.UserAlias+"-app", nil)
	if err != nil {
		slog.Debug("failed to finish the request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		slog.Debug("failed to pull the result:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (s *serverRepository) DestroyAzureContainerGroup(server entity.Server) error {

	ctx := context.Background()

	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth && !s.appConfig.ActlabsServerUseMsi {
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armcontainerinstance.NewContainerGroupsClient(server.SubscriptionId, cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	poller, err := clientFactory.BeginDelete(ctx, server.ResourceGroup, server.UserAlias+"-aci", nil)
	if err != nil {
		slog.Debug("failed to finish the request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		slog.Debug("failed to pull the result:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

// https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/create-or-update?view=rest-managedidentity-2023-01-31&tabs=Go
func (s *serverRepository) CreateUserAssignedManagedIdentity(server entity.Server) (entity.Server, error) {
	ctx := context.Background()
	clientFactory, err := armmsi.NewClientFactory(server.SubscriptionId, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	res, err := clientFactory.NewUserAssignedIdentitiesClient().CreateOrUpdate(ctx, server.ResourceGroup, server.UserAlias+"-msi", armmsi.Identity{
		Location: to.Ptr(server.Region),
	}, nil)
	if err != nil {
		slog.Debug("failed to finish the request:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	slog.Debug("Managed Identity: " + *res.ID)
	slog.Debug("Managed Identity Client ID: " + *res.Properties.ClientID)
	slog.Debug("Managed Identity Principal ID: " + *res.Properties.PrincipalID)

	server.ManagedIdentityClientId = *res.Properties.ClientID
	server.ManagedIdentityPrincipalId = *res.Properties.PrincipalID
	server.ManagedIdentityResourceId = *res.ID

	return server, nil
}

// verify that user is the owner/contributor of the subscription
func (s *serverRepository) IsUserAuthorized(server entity.Server) (bool, error) {
	slog.Debug("is user owner/contributor of subscription",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
	)

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
		page, err := pager.NextPage(context.Background())
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

// verify that actlabs is the contributor of the subscription
func (s *serverRepository) IsActlabsAuthorized(server entity.Server) (bool, error) {
	slog.Debug("is actlabs contributor of subscription",
		slog.String("userAlias", server.UserAlias),
		slog.String("subscriptionId", server.SubscriptionId),
	)

	if server.UserAlias == "" {
		return false, errors.New("userId is required")
	}

	if server.SubscriptionId == "" {
		return false, errors.New("subscriptionId is required")
	}

	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth && !s.appConfig.ActlabsServerUseMsi {
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armauthorization.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		return false, err
	}

	filter := "assignedTo('" + s.appConfig.ActlabsServerServicePrincipalObjectId + "')"

	pager := clientFactory.NewRoleAssignmentsClient().NewListForSubscriptionPager(&armauthorization.RoleAssignmentsClientListForSubscriptionOptions{
		Filter:   &filter,
		TenantID: nil,
	})
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return false, err
		}
		for _, roleAssignment := range page.Value {
			contributorRoleDefinitionID := "/subscriptions/" + server.SubscriptionId + "/providers" + entity.ContributorRoleDefinitionId

			if *roleAssignment.Properties.PrincipalID == s.appConfig.ActlabsServerServicePrincipalObjectId &&
				*roleAssignment.Properties.Scope == "/subscriptions/"+server.SubscriptionId &&
				*roleAssignment.Properties.RoleDefinitionID == contributorRoleDefinitionID {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *serverRepository) UpsertServerInDatabase(server entity.Server) error {
	server.PartitionKey = "actlabs"
	server.RowKey = server.UserPrincipalName

	val, err := json.Marshal(server)
	if err != nil {
		slog.Debug("error marshalling server:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = s.auth.ActlabsServersTableClient.UpsertEntity(context.Background(), val, nil)
	if err != nil {
		slog.Debug("error upserting server:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	slog.Debug("Server upserted in database")

	return nil
}
func (s *serverRepository) GetServerFromDatabase(partitionKey string, rowKey string) (entity.Server, error) {
	response, err := s.auth.ActlabsServersTableClient.GetEntity(context.Background(), partitionKey, rowKey, nil)
	if err != nil {
		slog.Debug("error getting server from database:",
			slog.String("userPrincipalName", rowKey),
			slog.String("error", err.Error()),
		)
		return entity.Server{}, err
	}

	server := entity.Server{}
	err = json.Unmarshal(response.Value, &server)
	if err != nil {
		slog.Debug("error unmarshalling server:",
			slog.String("userPrincipalName", rowKey),
			slog.String("error", err.Error()),
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
			slog.Debug("error getting servers from database:", slog.String("error", err.Error()))
			return servers, err
		}

		for _, e := range response.Entities {
			var myEntity aztables.EDMEntity
			var server entity.Server
			if err := json.Unmarshal(e, &myEntity); err != nil {
				slog.Debug("error unmarshalling server:", slog.String("error", err.Error()))
				return servers, err
			}
			propertiesBytes, err := json.Marshal(myEntity.Properties)
			if err != nil {
				slog.Debug("error marshalling server:", slog.String("error", err.Error()))
				return servers, err
			}
			if err := json.Unmarshal(propertiesBytes, &server); err != nil {
				slog.Debug("error unmarshalling server:", slog.String("error", err.Error()))
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
		slog.Debug("error deleting server from database:",
			slog.String("userPrincipalName", server.RowKey),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

// Get the list of container app environments in the resource group.
// Count the number of container apps in each environment.
// Deploy the container app in the environment with the least number of container apps.
func (s *serverRepository) GetNextManagedEnvironmentId(server entity.Server) (string, error) {

	slog.Debug("get next managed environment id",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
		slog.String("resourceGroup", s.appConfig.ActlabsServerResourceGroup),
	)

	ctx := context.Background()

	managedEnvironmentIds := map[string]int{}

	clientFactory, err := armappcontainers.NewClientFactory(s.appConfig.ActlabsHubSubscriptionID, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return "", err
	}

	pager := clientFactory.NewManagedEnvironmentsClient().NewListByResourceGroupPager(s.appConfig.ActlabsServerResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			slog.Debug("failed to get the list of managed environments in the resource group:",
				slog.String("userPrincipalName", server.UserPrincipalName),
				slog.String("subscriptionId", server.SubscriptionId),
				slog.String("resourceGroup", s.appConfig.ActlabsServerResourceGroup),
				slog.String("error", err.Error()),
			)
			return "", err
		}

		for _, managedEnvironment := range page.Value {
			managedEnvironmentIds[*managedEnvironment.ID] = 0
		}

		// Get the list of container apps in the resource group.
		// Count the number of container apps in each environment.
		appPager := clientFactory.NewContainerAppsClient().NewListByResourceGroupPager(s.appConfig.ActlabsServerResourceGroup, nil)
		for appPager.More() {
			appPage, err := appPager.NextPage(ctx)
			if err != nil {
				slog.Debug("failed to get the list of container apps in the resource group:",
					slog.String("userPrincipalName", server.UserPrincipalName),
					slog.String("subscriptionId", server.SubscriptionId),
					slog.String("resourceGroup", s.appConfig.ActlabsServerResourceGroup),
					slog.String("error", err.Error()),
				)
				return "", err
			}

			for _, app := range appPage.Value {
				if app.Properties.EnvironmentID != nil {
					managedEnvironmentIds[*app.Properties.EnvironmentID]++
				}
			}
		}

	}
	// Deploy the container app in the environment with the least number of container apps.
	min := math.MaxInt32
	environmentId := ""
	for id, count := range managedEnvironmentIds {
		if count < min {
			min = count
			environmentId = id
		}
	}

	if environmentId == "" {
		slog.Debug("no managed environment found",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
		)
		return "", errors.New("no managed environment found")
	}

	return environmentId, nil
}

func (s *serverRepository) ParseServerStatus(status string) entity.ServerStatus {
	switch status {
	case "Succeeded", "Running":
		return entity.ServerStatusRunning
	case "Creating", "Pending":
		return entity.ServerStatusStarting
	case "Stopped":
		return entity.ServerStatusStopped
	case "Deploying":
		return entity.ServerStatusDeploying
	case "Deployed":
		return entity.ServerStatusDeployed
	case "Destroyed":
		return entity.ServerStatusDestroyed
	case "Failed":
		return entity.ServerStatusFailed
	case "Registered":
		return entity.ServerStatusRegistered
	case "Stopping":
		return entity.ServerStatusStopping
	case "Unregistered":
		return entity.ServerStatusUnregistered
	case "Updating":
		return entity.ServerStatusUpdating
	// Add other cases as needed...
	default:
		return entity.ServerStatusUnknown
	}
}

func existsByName[T any](list []*T, name string, getName func(*T) *string) bool {
	// handle nil list safely
	if len(list) == 0 {
		return false
	}
	for _, v := range list {
		if v == nil || getName(v) == nil {
			continue
		}
		if *getName(v) == name {
			return true
		}
	}
	return false
}

func removeByName[T any](list []*T, name string, getName func(*T) *string) []*T {
	// handle nil list safely
	if len(list) == 0 {
		return list
	}
	newList := []*T{}
	for _, v := range list {
		if v == nil || getName(v) == nil {
			newList = append(newList, v)
			continue
		}
		if *getName(v) != name {
			newList = append(newList, v)
		}
	}
	return newList
}
