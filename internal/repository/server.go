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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
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
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
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
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
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
	if server.Version == "V2" || server.Version == "V3" {
		return s.DeployAzureContainerApp(server)
	}
	return s.DeployAzureContainerGroup(server)
}

func (s *serverRepository) DeployAzureContainerApp(server entity.Server) (entity.Server, error) {

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

	server.Endpoint = *resp.Properties.Configuration.Ingress.Fqdn
	server.Status = s.ParseServerStatus(string(*resp.Properties.ProvisioningState))

	return server, nil
}

func (s *serverRepository) DeployAzureContainerGroup(server entity.Server) (entity.Server, error) {

	ctx := context.Background()

	storageAccount, err := s.GetClientStorageAccount(server)
	if err != nil {
		return server, err
	}

	storageAccountKey, err := s.GetClientStorageAccountKey(server)
	if err != nil {
		return server, err
	}

	clientFactory, err := armcontainerinstance.NewContainerGroupsClient(server.SubscriptionId, s.auth.Cred, nil)
	if err != nil {
		slog.Debug("failed to create client:",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return server, err
	}

	poller, err := clientFactory.BeginCreateOrUpdate(ctx,
		server.ResourceGroup,
		server.UserAlias+"-aci", armcontainerinstance.ContainerGroup{
			Location: to.Ptr(server.Region),
			Identity: &armcontainerinstance.ContainerGroupIdentity{
				Type: to.Ptr(armcontainerinstance.ResourceIdentityTypeUserAssigned),
				UserAssignedIdentities: map[string]*armcontainerinstance.Components10Wh5UdSchemasContainergroupidentityPropertiesUserassignedidentitiesAdditionalproperties{
					server.ManagedIdentityResourceId: {},
				},
			},
			Properties: &armcontainerinstance.ContainerGroupProperties{
				// https://learn.microsoft.com/en-us/azure/container-instances/container-instances-init-container
				InitContainers: []*armcontainerinstance.InitContainerDefinition{
					{
						Name: to.Ptr("init"),
						Properties: &armcontainerinstance.InitContainerPropertiesDefinition{
							Image: to.Ptr("busybox"),
							EnvironmentVariables: []*armcontainerinstance.EnvironmentVariable{
								{
									Name:  to.Ptr("USER_ALIAS"),
									Value: to.Ptr(server.UserAlias),
								},
								{
									Name:  to.Ptr("ACTLABS_SERVER_PORT"),
									Value: to.Ptr(strconv.Itoa(int(s.appConfig.ActlabsServerPort))),
								},
								{
									Name:  to.Ptr("ACTLABS_SERVER_REGION"),
									Value: to.Ptr(server.Region),
								},
							},
							VolumeMounts: []*armcontainerinstance.VolumeMount{
								{
									Name:      to.Ptr("proxy-caddyfile"),
									MountPath: to.Ptr("/etc/caddy"),
								},
							},
							Command: []*string{
								to.Ptr("/bin/sh"),
								to.Ptr("-c"),
								to.Ptr("echo -e \"${USER_ALIAS}-actlabs-aci.${ACTLABS_SERVER_REGION}.azurecontainer.io {\n\treverse_proxy http://localhost:${ACTLABS_SERVER_PORT}\n}\" > /etc/caddy/Caddyfile"),
							},
						},
					},
				},
				Containers: []*armcontainerinstance.Container{
					{
						Name: to.Ptr("caddy"),
						Properties: &armcontainerinstance.ContainerProperties{
							Image: to.Ptr("ashishvermapu/caddy:latest"),
							Ports: []*armcontainerinstance.ContainerPort{
								{
									Port:     to.Ptr[int32](s.appConfig.HttpPort),
									Protocol: to.Ptr(armcontainerinstance.ContainerNetworkProtocolTCP),
								},
								{
									Port:     to.Ptr[int32](s.appConfig.HttpsPort),
									Protocol: to.Ptr(armcontainerinstance.ContainerNetworkProtocolTCP),
								},
							},
							Resources: &armcontainerinstance.ResourceRequirements{
								Requests: &armcontainerinstance.ResourceRequests{
									CPU:        to.Ptr[float64](s.appConfig.ActlabsServerCaddyCPU),
									MemoryInGB: to.Ptr[float64](s.appConfig.ActlabsServerCaddyMemory),
								},
							},
							VolumeMounts: []*armcontainerinstance.VolumeMount{
								{
									Name:      to.Ptr("emptydir"),
									MountPath: to.Ptr("/mnt/emptydir"),
								},
								{
									Name:      to.Ptr("proxy-caddyfile"),
									MountPath: to.Ptr("/etc/caddy"),
								},
								{
									Name:      to.Ptr("proxy-config"),
									MountPath: to.Ptr("/config"),
								},
								{
									Name:      to.Ptr("proxy-data"),
									MountPath: to.Ptr("/data"),
								},
							},
						},
					},
					{
						Name: to.Ptr("actlabs"),
						Properties: &armcontainerinstance.ContainerProperties{
							Image: to.Ptr("ashishvermapu/repro:latest"),
							Ports: []*armcontainerinstance.ContainerPort{
								{
									Port:     to.Ptr[int32](s.appConfig.ActlabsServerPort),
									Protocol: to.Ptr(armcontainerinstance.ContainerNetworkProtocolTCP),
								},
							},
							Resources: &armcontainerinstance.ResourceRequirements{
								Requests: &armcontainerinstance.ResourceRequests{
									CPU:        to.Ptr[float64](s.appConfig.ActlabsServerCPU),
									MemoryInGB: to.Ptr[float64](s.appConfig.ActlabsServerMemory),
								},
							},
							ReadinessProbe: &armcontainerinstance.ContainerProbe{
								InitialDelaySeconds: to.Ptr[int32](s.appConfig.ActlabsServerReadinessProbeInitialDelaySeconds),
								PeriodSeconds:       to.Ptr[int32](s.appConfig.ActlabsServerReadinessProbePeriodSeconds),
								FailureThreshold:    to.Ptr[int32](s.appConfig.ActlabsServerReadinessProbeFailureThreshold),
								SuccessThreshold:    to.Ptr[int32](s.appConfig.ActlabsServerReadinessProbeSuccessThreshold),
								TimeoutSeconds:      to.Ptr[int32](s.appConfig.ActlabsServerReadinessProbeTimeoutSeconds),
								HTTPGet: &armcontainerinstance.ContainerHTTPGet{
									Path:   to.Ptr(s.appConfig.ActlabsServerReadinessProbePath),
									Port:   to.Ptr[int32](s.appConfig.ActlabsServerPort),
									Scheme: to.Ptr(armcontainerinstance.SchemeHTTP),
								},
							},
							EnvironmentVariables: []*armcontainerinstance.EnvironmentVariable{
								{
									Name:  to.Ptr("ARM_USE_MSI"),
									Value: to.Ptr(strconv.FormatBool(s.appConfig.ActlabsServerUseMsi)),
								},
								{
									Name:  to.Ptr("USE_SERVICE_PRINCIPAL"),
									Value: to.Ptr("false"),
								},
								{
									Name:  to.Ptr("USE_MSI"),
									Value: to.Ptr(strconv.FormatBool(s.appConfig.ActlabsServerUseMsi)),
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
									Name:  to.Ptr("AZURE_CLIENT_ID"), // https://github.com/microsoft/azure-container-apps/issues/442
									Value: &server.ManagedIdentityClientId,
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
									Value: to.Ptr(s.appConfig.TenantID),
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
							},
							VolumeMounts: []*armcontainerinstance.VolumeMount{
								{
									Name:      to.Ptr("emptydir"),
									MountPath: to.Ptr("/mnt/emptydir"),
								},
								{
									Name:      to.Ptr("proxy-caddyfile"),
									MountPath: to.Ptr("/etc/caddy"),
								},
								{
									Name:      to.Ptr("proxy-config"),
									MountPath: to.Ptr("/config"),
								},
								{
									Name:      to.Ptr("proxy-data"),
									MountPath: to.Ptr("/data"),
								},
							},
						},
					},
				},
				OSType:        to.Ptr(armcontainerinstance.OperatingSystemTypesLinux),
				RestartPolicy: to.Ptr(armcontainerinstance.ContainerGroupRestartPolicyAlways),
				IPAddress: &armcontainerinstance.IPAddress{
					Ports: []*armcontainerinstance.Port{
						{
							Port:     to.Ptr[int32](s.appConfig.HttpPort),
							Protocol: to.Ptr(armcontainerinstance.ContainerGroupNetworkProtocolTCP),
						},
						{
							Port:     to.Ptr[int32](s.appConfig.HttpsPort),
							Protocol: to.Ptr(armcontainerinstance.ContainerGroupNetworkProtocolTCP),
						},
					},
					Type:         to.Ptr(armcontainerinstance.ContainerGroupIPAddressTypePublic),
					DNSNameLabel: to.Ptr(server.UserAlias + "-actlabs-aci"),
				},
				Volumes: []*armcontainerinstance.Volume{
					{
						Name:     to.Ptr("emptydir"),
						EmptyDir: &struct{}{},
					},
					{
						Name: to.Ptr("proxy-caddyfile"),
						AzureFile: &armcontainerinstance.AzureFileVolume{
							ShareName:          to.Ptr("proxy-caddyfile"),
							ReadOnly:           to.Ptr(false),
							StorageAccountName: storageAccount.Name,
							StorageAccountKey:  to.Ptr(storageAccountKey),
						},
					},
					{
						Name: to.Ptr("proxy-config"),
						AzureFile: &armcontainerinstance.AzureFileVolume{
							ShareName:          to.Ptr("proxy-config"),
							ReadOnly:           to.Ptr(false),
							StorageAccountName: storageAccount.Name,
							StorageAccountKey:  to.Ptr(storageAccountKey),
						},
					},
					{
						Name: to.Ptr("proxy-data"),
						AzureFile: &armcontainerinstance.AzureFileVolume{
							ShareName:          to.Ptr("proxy-data"),
							ReadOnly:           to.Ptr(false),
							StorageAccountName: storageAccount.Name,
							StorageAccountKey:  to.Ptr(storageAccountKey),
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

	server.Endpoint = *resp.Properties.IPAddress.Fqdn
	server.Status = s.ParseServerStatus(*resp.Properties.ProvisioningState)

	return server, nil
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
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
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
		if !s.appConfig.ActlabsHubUseUserAuth {
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
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
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

func (s *serverRepository) GetResourceGroupRegion(ctx context.Context, server entity.Server) (string, error) {
	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armresources.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		slog.Debug("failed to create client factory to get resource group region",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)

		return "", err
	}

	res, err := clientFactory.NewResourceGroupsClient().Get(ctx, server.ResourceGroup, nil)
	if err != nil {
		slog.Debug("failed to finish the request to get resource group region",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("resourceGroup", server.ResourceGroup),
			slog.String("error", err.Error()),
		)
		return "", err
	}

	return *res.Location, nil
}

func (s *serverRepository) EnableStorageAccountAccessKeys(ctx context.Context, server entity.Server) error {
	return s.UpdateStorageAccountAccessKeys(ctx, server, true)
}

func (s *serverRepository) DisableStorageAccountAccessKeys(ctx context.Context, server entity.Server) error {
	return s.UpdateStorageAccountAccessKeys(ctx, server, false)
}

func (s *serverRepository) UpdateStorageAccountAccessKeys(ctx context.Context, server entity.Server, status bool) error {

	cred := s.auth.Cred

	slog.Info("updating shared key access for storage account",
		slog.String("userPrincipalName", server.UserPrincipalName),
		slog.String("subscriptionId", server.SubscriptionId),
		slog.String("resourceGroup", server.ResourceGroup),
		slog.String("Access Key status", strconv.FormatBool(status)),
		slog.String("Server Version", server.Version),
		slog.String("ActlabsHubUseUserAuth", strconv.FormatBool(s.appConfig.ActlabsHubUseUserAuth)),
	)

	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
		slog.Info("using FDPO credentials for storage account access keys")
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armstorage.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		slog.Error("not able to create client factory to get storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	account, err := s.GetClientStorageAccount(server)
	if err != nil {
		slog.Error("not able to get storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = clientFactory.NewAccountsClient().Update(ctx, server.ResourceGroup, *account.Name, armstorage.AccountUpdateParameters{
		Properties: &armstorage.AccountPropertiesUpdateParameters{
			AllowSharedKeyAccess: to.Ptr(status),
		},
	}, nil)

	if err != nil {
		slog.Error("not able to update shared key access for storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("resourceGroup", server.ResourceGroup),
			slog.String("storageAccount", *account.Name),
			slog.String("Access Key status", strconv.FormatBool(status)),
		)
		return err
	}

	return nil
}

func (s *serverRepository) DeleteResourceGroup(ctx context.Context, server entity.Server) error {

	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armresources.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		slog.Debug("failed to create client factory to delete resource group",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	poller, err := clientFactory.NewResourceGroupsClient().BeginDelete(ctx, server.ResourceGroup, nil)
	if err != nil {
		slog.Debug("failed to finish the request to delete resource group",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("resourceGroup", server.ResourceGroup),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		slog.Debug("failed to pull the result to delete resource group",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("resourceGroup", server.ResourceGroup),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (s *serverRepository) DeleteStorageAccount(ctx context.Context, server entity.Server) error {

	cred := s.auth.Cred
	if server.Version == "V3" && !s.appConfig.ActlabsHubUseUserAuth {
		cred = s.auth.FdpoCredential
	}

	clientFactory, err := armstorage.NewClientFactory(server.SubscriptionId, cred, nil)
	if err != nil {
		slog.Debug("not able to create client factory to get storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	account, err := s.GetClientStorageAccount(server)
	if err != nil {
		slog.Debug("not able to get storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = clientFactory.NewAccountsClient().Delete(ctx, server.ResourceGroup, *account.Name, nil)
	if err != nil {
		slog.Debug("not able to delete storage account",
			slog.String("userPrincipalName", server.UserPrincipalName),
			slog.String("subscriptionId", server.SubscriptionId),
			slog.String("resourceGroup", server.ResourceGroup),
			slog.String("storageAccount", *account.Name),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
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
