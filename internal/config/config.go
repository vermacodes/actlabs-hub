package config

import (
	"actlabs-hub/internal/logger"
	"context"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ActlabsAppGatewayName                                    string
	ActlabsEnvironmentName                                   string
	ActlabsFQDN                                              string
	ActlabsHubClientID                                       string
	ActlabsHubManagedServersTableName                        string
	ActlabsHubReadinessAssignmentsTableName                  string
	ActlabsHubChallengesTableName                            string
	ActlabsHubProfilesTableName                              string
	ActlabsHubDeploymentsTableName                           string
	ActlabsHubEventsTableName                                string
	ActlabsHubDeploymentOperationsTableName                  string
	ActlabsHubManagedIdentityResourceId                      string
	ActlabsHubResourceGroup                                  string
	ActlabsHubStorageAccount                                 string
	ActlabsHubSubscriptionID                                 string
	ActlabsHubURL                                            string
	ActlabsHubAutoDestroyPollingIntervalSeconds              int32
	ActlabsHubAutoDestroyIdleTimeSeconds                     int32
	ActlabsHubDeploymentsPollingIntervalSeconds              int32
	ActlabsHubMonitorAndDestroyInactiveServers               bool
	ActlabsHubMonitorAndAutoDestroyDeployments               bool
	ActlabsServerCaddyCPU                                    float64
	ActlabsServerCaddyMemory                                 float64
	ActlabsServerCPU                                         float64
	ActlabsServerMemory                                      float64
	ActlabsServerImage                                       string
	ActlabsServerPort                                        int32
	ActlabsServerReadinessProbeFailureThreshold              int32
	ActlabsServerReadinessProbeInitialDelaySeconds           int32
	ActlabsServerReadinessProbePath                          string
	ActlabsServerReadinessProbePeriodSeconds                 int32
	ActlabsServerReadinessProbeSuccessThreshold              int32
	ActlabsServerReadinessProbeTimeoutSeconds                int32
	ActlabsServerRootDir                                     string
	ActlabsServerUPWaitTimeSeconds                           string
	ActlabsServerManagedEnvironmentId                        string
	ActlabsServerResourceGroup                               string
	AuthTokenAud                                             string
	AuthTokenIss                                             string
	HttpPort                                                 int32
	HttpsPort                                                int32
	TenantID                                                 string
	ActlabsServerApiKey                                      string
	ActlabsServerEndpointExternal                            string
	ActlabsServerEndpointInternal                            string
	ActlabsServerUseMsi                                      bool
	ActlabsServerUseServicePrincipal                         bool
	ActlabsServerServicePrincipalClientId                    string
	ActlabsServerServicePrincipalObjectId                    string
	ActlabsServerServicePrincipalClientSecretKeyvaultURL     string
	ActlabsServerFdpoServicePrincipalClientId                string
	ActlabsServerFdpoServicePrincipalObjectId                string
	ActlabsServerFdpoServicePrincipalSecret                  string
	FdpoTenantID                                             string
	ActlabsServerFdpoServicePrincipalClientSecretKeyvaultURL string
	ActlabsHubUseMsi                                         bool
	ActlabsHubUseUserAuth                                    bool
	MiseEndpoint                                             string
	MiseVerboseLogging                                       bool
	AuthVerifyMode                                           string
	CorsAllowOrigins                                         string
	CorsAllowMethods                                         string
	CorsAllowHeaders                                         string
	ActlabsServerAroRpFirstPartySpID                         string
	ActlabsServerAppSettingWebsiteSiteName                   string
	ActlabsServerArmMsiApiVersion                            string
	ActlabsServerArmMsiApiProxyPort                          string
	ActlabsServerAPIKey                                      string
	// Add other configuration fields as needed
}

func NewConfig(ctx context.Context) (*Config, error) {

	actlabsAppGatewayName := getEnv(ctx, "ACTLABS_APP_GATEWAY_NAME")
	if actlabsAppGatewayName == "" {
		return nil, fmt.Errorf("ACTLABS_APP_GATEWAY_NAME not set")
	}

	actlabsEnvironmentName := getEnv(ctx, "ACTLABS_ENVIRONMENT_NAME")
	if actlabsEnvironmentName == "" {
		return nil, fmt.Errorf("ACTLABS_ENVIRONMENT_NAME not set")
	}

	actlabsFQDN := getEnv(ctx, "ACTLABS_FQDN")
	if actlabsFQDN == "" {
		return nil, fmt.Errorf("ACTLABS_FQDN not set")
	}

	authTokenAud := getEnv(ctx, "AUTH_TOKEN_AUD")
	if authTokenAud == "" {
		return nil, fmt.Errorf("AUTH_TOKEN_AUD not set")
	}

	authTokenIss := getEnv(ctx, "AUTH_TOKEN_ISS")
	if authTokenIss == "" {
		return nil, fmt.Errorf("AUTH_TOKEN_ISS not set")
	}

	actlabsServerRootDir := getEnv(ctx, "ACTLABS_SERVER_ROOT_DIR")
	if actlabsServerRootDir == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_ROOT_DIR not set")
	}

	actlabsServerApiKey := getEnv(ctx, "ACTLABS_SERVER_API_KEY")
	if actlabsServerApiKey == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_API_KEY not set")
	}

	actlabsServerEndpointExternal := getEnv(ctx, "ACTLABS_SERVER_ENDPOINT_EXTERNAL")
	if actlabsServerEndpointExternal == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_ENDPOINT_EXTERNAL not set")
	}

	actlabsServerEndpointInternal := getEnv(ctx, "ACTLABS_SERVER_ENDPOINT_INTERNAL")
	if actlabsServerEndpointInternal == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_ENDPOINT_INTERNAL not set")
	}

	actlabsServerUseMsi, err := strconv.ParseBool(getEnvWithDefault(ctx, "ACTLABS_SERVER_USE_MSI", "false"))
	if err != nil {
		return nil, err
	}

	actlabsServerUseServicePrincipal, err := strconv.ParseBool(getEnvWithDefault(ctx, "ACTLABS_SERVER_USE_SERVICE_PRINCIPAL", "false"))
	if err != nil {
		return nil, err
	}

	actlabsServerServicePrincipalClientId := getEnv(ctx, "ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_ID")
	if actlabsServerServicePrincipalClientId == "" && actlabsServerUseServicePrincipal {
		return nil, fmt.Errorf("ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_ID not set")
	}

	actlabsServerServicePrincipalObjectId := getEnv(ctx, "ACTLABS_SERVER_SERVICE_PRINCIPAL_OBJECT_ID")
	if actlabsServerServicePrincipalObjectId == "" && actlabsServerUseServicePrincipal {
		return nil, fmt.Errorf("ACTLABS_SERVER_SERVICE_PRINCIPAL_OBJECT_ID not set")
	}

	actlabsServerServicePrincipalClientSecretKeyvaultURL := getEnv(ctx, "ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL")
	if actlabsServerServicePrincipalClientSecretKeyvaultURL == "" && actlabsServerUseServicePrincipal {
		return nil, fmt.Errorf("ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL not set")
	}

	actlabsServerFdpoServicePrincipalClientId := getEnv(ctx, "ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_ID")
	if actlabsServerFdpoServicePrincipalClientId == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_ID not set")
	}

	actlabsServerFdpoServicePrincipalObjectId := getEnv(ctx, "ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_OBJECT_ID")
	if actlabsServerFdpoServicePrincipalObjectId == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_OBJECT_ID not set")
	}

	actlabsServerFdpoServicePrincipalSecret := getEnv(ctx, "ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_SECRET")
	if actlabsServerFdpoServicePrincipalSecret == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_SECRET not set")
	}

	fdpoTenantID := getEnv(ctx, "FDPO_TENANT_ID")
	if fdpoTenantID == "" {
		return nil, fmt.Errorf("FDPO_TENANT_ID not set")
	}

	actlabsServerFdpoServicePrincipalClientSecretKeyvaultURL := getEnv(ctx, "ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL")
	if actlabsServerFdpoServicePrincipalClientSecretKeyvaultURL == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL not set")
	}

	actlabsHubUseMsi, err := strconv.ParseBool(getEnvWithDefault(ctx, "ACTLABS_HUB_USE_MSI", "false"))
	if err != nil {
		return nil, err
	}

	actlabsHubUseUserAuth, err := strconv.ParseBool(getEnvWithDefault(ctx, "ACTLABS_HUB_USE_USER_AUTH", "false"))
	if err != nil {
		return nil, err
	}

	actlabsServerUPWaitTimeSeconds := getEnvWithDefault(ctx, "ACTLABS_SERVER_UP_WAIT_TIME_SECONDS", "180")
	if actlabsServerUPWaitTimeSeconds == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_UP_WAIT_TIME_SECONDS not set")
	}

	actlabsServerManagedEnvironmentId := getEnv(ctx, "ACTLABS_SERVER_MANAGED_ENVIRONMENT_ID")
	if actlabsServerManagedEnvironmentId == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_MANAGED_ENVIRONMENT_ID not set")
	}

	actlabsServerResourceGroup := getEnv(ctx, "ACTLABS_SERVER_RESOURCE_GROUP")
	if actlabsServerResourceGroup == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_RESOURCE_GROUP not set")
	}

	actlabsServerPort, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_SERVER_PORT", "8881"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("ACTLABS_SERVER_PORT not set")
	}

	actlabsServerImage := getEnvWithDefault(ctx, "ACTLABS_SERVER_IMAGE", "actlabs.azurecr.io/repro:latest")
	if actlabsServerImage == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_IMAGE not set")
	}

	actlabsHubURL := getEnv(ctx, "ACTLABS_HUB_URL")
	if actlabsHubURL == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_URL not set")
	}

	httpPort, err := strconv.Atoi(getEnvWithDefault(ctx, "HTTP_PORT", "80"))
	if err != nil {
		return nil, fmt.Errorf("HTTP_PORT not set")
	}

	httpsPort, err := strconv.Atoi(getEnvWithDefault(ctx, "HTTPS_PORT", "443"))
	if err != nil {
		return nil, fmt.Errorf("HTTPS_PORT not set")
	}

	actlabsServerReadinessProbePath := getEnvWithDefault(ctx, "ACTLABS_SERVER_READINESS_PROBE_PATH", "/status")

	tenantID := getEnv(ctx, "TENANT_ID")
	if tenantID == "" {
		return nil, fmt.Errorf("TENANT_ID not set")
	}

	actlabsServerCPUFloat, err := strconv.ParseFloat(getEnvWithDefault(ctx, "ACTLABS_SERVER_CPU", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerMemoryFloat, err := strconv.ParseFloat(getEnvWithDefault(ctx, "ACTLABS_SERVER_MEMORY", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerCaddyCPUFloat, err := strconv.ParseFloat(getEnvWithDefault(ctx, "ACTLABS_SERVER_CADDY_CPU", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerCaddyMemoryFloat, err := strconv.ParseFloat(getEnvWithDefault(ctx, "ACTLABS_SERVER_CADDY_MEMORY", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeInitialDelaySecondsInt, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_SERVER_READINESS_PROBE_INITIAL_DELAY_SECONDS", "10"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeTimeoutSecondsInt, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_SERVER_READINESS_PROBE_TIMEOUT_SECONDS", "5"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbePeriodSecondsInt, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_SERVER_READINESS_PROBE_PERIOD_SECONDS", "10"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeSuccessThresholdInt, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_SERVER_READINESS_PROBE_SUCCESS_THRESHOLD", "1"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeFailureThresholdInt, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_SERVER_READINESS_PROBE_FAILURE_THRESHOLD", "20"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsHubClientID := getEnv(ctx, "ACTLABS_HUB_CLIENT_ID")
	if actlabsHubClientID == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_CLIENT_ID not set")
	}

	actlabsHubSubscriptionID := getEnv(ctx, "ACTLABS_HUB_SUBSCRIPTION_ID")
	if actlabsHubSubscriptionID == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_SUBSCRIPTION_ID not set")
	}

	actlabsHubResourceGroup := getEnv(ctx, "ACTLABS_HUB_RESOURCE_GROUP_NAME")
	if actlabsHubResourceGroup == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_RESOURCE_GROUP_NAME not set")
	}

	actlabsHubStorageAccount := getEnv(ctx, "ACTLABS_HUB_STORAGE_ACCOUNT_NAME")
	if actlabsHubStorageAccount == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_STORAGE_ACCOUNT_NAME not set")
	}

	actlabsHubManagedServersTableName := getEnv(ctx, "ACTLABS_HUB_MANAGED_SERVERS_TABLE_NAME")
	if actlabsHubManagedServersTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_MANAGED_SERVERS_TABLE_NAME not set")
	}

	actlabsHubReadinessAssignmentsTableName := getEnv(ctx, "ACTLABS_HUB_READINESS_ASSIGNMENTS_TABLE_NAME")
	if actlabsHubReadinessAssignmentsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_READINESS_ASSIGNMENTS_TABLE_NAME not set")
	}

	actlabsHubChallengesTableName := getEnv(ctx, "ACTLABS_HUB_CHALLENGES_TABLE_NAME")
	if actlabsHubChallengesTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_CHALLENGES_TABLE_NAME not set")
	}

	actlabsHubProfilesTableName := getEnv(ctx, "ACTLABS_HUB_PROFILES_TABLE_NAME")
	if actlabsHubProfilesTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_PROFILES_TABLE_NAME not set")
	}

	actlabsHubDeploymentsTableName := getEnv(ctx, "ACTLABS_HUB_DEPLOYMENTS_TABLE_NAME")
	if actlabsHubDeploymentsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_DEPLOYMENTS_TABLE_NAME not set")
	}

	actlabsHubEventsTableName := getEnv(ctx, "ACTLABS_HUB_EVENTS_TABLE_NAME")
	if actlabsHubEventsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_EVENTS_TABLE_NAME not set")
	}

	actlabsHubDeploymentOperationsTableName := getEnv(ctx, "ACTLABS_HUB_DEPLOYMENT_OPERATIONS_TABLE_NAME")
	if actlabsHubDeploymentOperationsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_DEPLOYMENT_OPERATIONS_TABLE_NAME not set")
	}

	actlabsHubManagedIdentityResourceId := getEnv(ctx, "ACTLABS_HUB_MANAGED_IDENTITY_RESOURCE_ID")
	if actlabsHubManagedIdentityResourceId == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_MANAGED_IDENTITY_RESOURCE_ID not set")
	}

	actlabsHubAutoDestroyPollingIntervalSeconds, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_HUB_AUTO_DESTROY_POLLING_INTERVAL_SECONDS", "600"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsHubAutoDestroyIdleTimeSeconds, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_HUB_AUTO_DESTROY_IDLE_TIME_SECONDS", "3600"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsHubDeploymentsPollingIntervalSeconds, err := strconv.ParseInt(getEnvWithDefault(ctx, "ACTLABS_HUB_DEPLOYMENTS_POLLING_INTERVAL_SECONDS", "30"), 10, 32)
	if err != nil {
		return nil, err
	}

	miseEndpoint := getEnv(ctx, "MISE_ENDPOINT")
	if miseEndpoint == "" {
		return nil, fmt.Errorf("MISE_ENDPOINT not set")
	}

	miseVerboseLogging, err := strconv.ParseBool(getEnvWithDefault(ctx, "MISE_VERBOSE_LOGGING", "false"))
	if err != nil {
		return nil, fmt.Errorf("MISE_VERBOSE_LOGGING not set or invalid: %w", err)
	}

	authVerifyMode := getEnvWithDefault(ctx, "AUTH_VERIFY_MODE", "Custom")

	corsAllowOrigins := getEnv(ctx, "CORS_ALLOW_ORIGINS")
	if corsAllowOrigins == "" {
		return nil, fmt.Errorf("CORS_ALLOW_ORIGINS not set")
	}

	corsAllowMethods := getEnv(ctx, "CORS_ALLOW_METHODS")
	if corsAllowMethods == "" {
		return nil, fmt.Errorf("CORS_ALLOW_METHODS not set")
	}

	corsAllowHeaders := getEnv(ctx, "CORS_ALLOW_HEADERS")
	if corsAllowHeaders == "" {
		return nil, fmt.Errorf("CORS_ALLOW_HEADERS not set")
	}

	actlabsServerAroRpFirstPartySpID := getEnv(ctx, "ACTLABS_SERVER_AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID")
	if actlabsServerAroRpFirstPartySpID == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID not set")
	}

	actlabsServerAppSettingWebsiteSiteName := getEnvWithDefault(ctx, "ACTLABS_SERVER_APPSETTING_WEBSITE_SITE_NAME", "dummy")
	if actlabsServerAppSettingWebsiteSiteName == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_APPSETTING_WEBSITE_SITE_NAME not set")
	}

	actlabsServerArmMsiApiVersion := getEnvWithDefault(ctx, "ACTLABS_SERVER_ARM_MSI_API_VERSION", "2019-08-01")
	if actlabsServerArmMsiApiVersion == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_ARM_MSI_API_VERSION not set")
	}

	actlabsServerArmMsiApiProxyPort := getEnvWithDefault(ctx, "ACTLABS_SERVER_ARM_MSI_API_PROXY_PORT", "42300")
	if actlabsServerArmMsiApiProxyPort == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_ARM_MSI_API_PROXY_PORT not set")
	}

	actlabsHubMonitorAndDestroyInactiveServers, err := strconv.ParseBool(getEnvWithDefault(ctx, "ACTLABS_HUB_MONITOR_AND_DESTROY_INACTIVE_SERVERS", "true"))
	if err != nil {
		return nil, err
	}

	actlabsHubMonitorAndAutoDestroyDeployments, err := strconv.ParseBool(getEnvWithDefault(ctx, "ACTLABS_HUB_MONITOR_AUTO_DESTROY_DEPLOYMENTS", "true"))
	if err != nil {
		return nil, err
	}

	// Retrieve other environment variables and check them as needed

	return &Config{
		ActlabsAppGatewayName:                                    actlabsAppGatewayName,
		ActlabsEnvironmentName:                                   actlabsEnvironmentName,
		ActlabsFQDN:                                              actlabsFQDN,
		ActlabsHubClientID:                                       actlabsHubClientID,
		ActlabsHubManagedServersTableName:                        actlabsHubManagedServersTableName,
		ActlabsHubReadinessAssignmentsTableName:                  actlabsHubReadinessAssignmentsTableName,
		ActlabsHubChallengesTableName:                            actlabsHubChallengesTableName,
		ActlabsHubProfilesTableName:                              actlabsHubProfilesTableName,
		ActlabsHubDeploymentsTableName:                           actlabsHubDeploymentsTableName,
		ActlabsHubEventsTableName:                                actlabsHubEventsTableName,
		ActlabsHubDeploymentOperationsTableName:                  actlabsHubDeploymentOperationsTableName,
		ActlabsHubManagedIdentityResourceId:                      actlabsHubManagedIdentityResourceId,
		ActlabsHubResourceGroup:                                  actlabsHubResourceGroup,
		ActlabsHubStorageAccount:                                 actlabsHubStorageAccount,
		ActlabsHubSubscriptionID:                                 actlabsHubSubscriptionID,
		ActlabsHubURL:                                            actlabsHubURL,
		ActlabsHubAutoDestroyPollingIntervalSeconds:              int32(actlabsHubAutoDestroyPollingIntervalSeconds),
		ActlabsHubAutoDestroyIdleTimeSeconds:                     int32(actlabsHubAutoDestroyIdleTimeSeconds),
		ActlabsHubDeploymentsPollingIntervalSeconds:              int32(actlabsHubDeploymentsPollingIntervalSeconds),
		ActlabsServerCaddyCPU:                                    actlabsServerCaddyCPUFloat,
		ActlabsServerCaddyMemory:                                 actlabsServerCaddyMemoryFloat,
		ActlabsServerCPU:                                         actlabsServerCPUFloat,
		ActlabsServerMemory:                                      actlabsServerMemoryFloat,
		ActlabsServerPort:                                        int32(actlabsServerPort),
		ActlabsServerImage:                                       actlabsServerImage,
		ActlabsServerReadinessProbeFailureThreshold:              int32(actlabsServerReadinessProbeFailureThresholdInt),
		ActlabsServerReadinessProbeInitialDelaySeconds:           int32(actlabsServerReadinessProbeInitialDelaySecondsInt),
		ActlabsServerReadinessProbePath:                          actlabsServerReadinessProbePath,
		ActlabsServerReadinessProbePeriodSeconds:                 int32(actlabsServerReadinessProbePeriodSecondsInt),
		ActlabsServerReadinessProbeSuccessThreshold:              int32(actlabsServerReadinessProbeSuccessThresholdInt),
		ActlabsServerReadinessProbeTimeoutSeconds:                int32(actlabsServerReadinessProbeTimeoutSecondsInt),
		ActlabsServerRootDir:                                     actlabsServerRootDir,
		ActlabsServerUPWaitTimeSeconds:                           actlabsServerUPWaitTimeSeconds,
		ActlabsServerManagedEnvironmentId:                        actlabsServerManagedEnvironmentId,
		ActlabsServerResourceGroup:                               actlabsServerResourceGroup,
		ActlabsHubMonitorAndDestroyInactiveServers:               actlabsHubMonitorAndDestroyInactiveServers,
		ActlabsHubMonitorAndAutoDestroyDeployments:               actlabsHubMonitorAndAutoDestroyDeployments,
		AuthTokenAud:                                             authTokenAud,
		AuthTokenIss:                                             authTokenIss,
		HttpPort:                                                 int32(httpPort),
		HttpsPort:                                                int32(httpsPort),
		TenantID:                                                 tenantID,
		ActlabsServerApiKey:                                      actlabsServerApiKey,
		ActlabsServerEndpointExternal:                            actlabsServerEndpointExternal,
		ActlabsServerEndpointInternal:                            actlabsServerEndpointInternal,
		ActlabsServerUseMsi:                                      actlabsServerUseMsi,
		ActlabsServerUseServicePrincipal:                         actlabsServerUseServicePrincipal,
		ActlabsServerServicePrincipalClientId:                    actlabsServerServicePrincipalClientId,
		ActlabsServerServicePrincipalObjectId:                    actlabsServerServicePrincipalObjectId,
		ActlabsServerServicePrincipalClientSecretKeyvaultURL:     actlabsServerServicePrincipalClientSecretKeyvaultURL,
		ActlabsServerFdpoServicePrincipalClientId:                actlabsServerFdpoServicePrincipalClientId,
		ActlabsServerFdpoServicePrincipalObjectId:                actlabsServerFdpoServicePrincipalObjectId,
		ActlabsServerFdpoServicePrincipalSecret:                  actlabsServerFdpoServicePrincipalSecret,
		FdpoTenantID:                                             fdpoTenantID,
		ActlabsServerFdpoServicePrincipalClientSecretKeyvaultURL: actlabsServerFdpoServicePrincipalClientSecretKeyvaultURL,
		ActlabsHubUseMsi:                                         actlabsHubUseMsi,
		ActlabsHubUseUserAuth:                                    actlabsHubUseUserAuth,
		MiseEndpoint:                                             miseEndpoint,
		MiseVerboseLogging:                                       miseVerboseLogging,
		AuthVerifyMode:                                           authVerifyMode,
		CorsAllowOrigins:                                         corsAllowOrigins,
		CorsAllowMethods:                                         corsAllowMethods,
		CorsAllowHeaders:                                         corsAllowHeaders,
		ActlabsServerAroRpFirstPartySpID:                         actlabsServerAroRpFirstPartySpID,
		ActlabsServerAppSettingWebsiteSiteName:                   actlabsServerAppSettingWebsiteSiteName,
		ActlabsServerArmMsiApiVersion:                            actlabsServerArmMsiApiVersion,
		ActlabsServerArmMsiApiProxyPort:                          actlabsServerArmMsiApiProxyPort,
		// Set other fields
	}, nil
}

// Helper function to retrieve the value and log it
func getEnv(ctx context.Context, env string) string {
	value := os.Getenv(env)
	logger.LogDebug(ctx, "environment variable", "name", env, "value", value)
	return value
}

// Helper function to retrieve the value, if none found, set default and log it
func getEnvWithDefault(ctx context.Context, env string, defaultValue string) string {
	value := os.Getenv(env)
	if value == "" {
		value = defaultValue
	}
	logger.LogInfo(ctx, "environment variable", "name", env, "value", value)
	return value
}
