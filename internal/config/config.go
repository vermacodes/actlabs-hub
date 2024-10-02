package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"golang.org/x/exp/slog"
)

type Config struct {
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
	ProtectedLabSecret                                       string
	TenantID                                                 string
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
	// Add other configuration fields as needed
}

func NewConfig() (*Config, error) {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file")
	}

	authTokenAud := getEnv("AUTH_TOKEN_AUD")
	if authTokenAud == "" {
		return nil, fmt.Errorf("AUTH_TOKEN_AUD not set")
	}

	authTokenIss := getEnv("AUTH_TOKEN_ISS")
	if authTokenIss == "" {
		return nil, fmt.Errorf("AUTH_TOKEN_ISS not set")
	}

	actlabsServerRootDir := getEnv("ACTLABS_SERVER_ROOT_DIR")
	if actlabsServerRootDir == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_ROOT_DIR not set")
	}

	protectedLabSecret := getEnv("PROTECTED_LAB_SECRET")
	if protectedLabSecret == "" {
		return nil, fmt.Errorf("PROTECTED_LAB_SECRET not set")
	}

	actlabsServerUseMsi, err := strconv.ParseBool(getEnvWithDefault("ACTLABS_SERVER_USE_MSI", "false"))
	if err != nil {
		return nil, err
	}

	actlabsServerUseServicePrincipal, err := strconv.ParseBool(getEnvWithDefault("ACTLABS_SERVER_USE_SERVICE_PRINCIPAL", "false"))
	if err != nil {
		return nil, err
	}

	actlabsServerServicePrincipalClientId := getEnv("ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_ID")
	if actlabsServerServicePrincipalClientId == "" && actlabsServerUseServicePrincipal {
		return nil, fmt.Errorf("ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_ID not set")
	}

	actlabsServerServicePrincipalObjectId := getEnv("ACTLABS_SERVER_SERVICE_PRINCIPAL_OBJECT_ID")
	if actlabsServerServicePrincipalObjectId == "" && actlabsServerUseServicePrincipal {
		return nil, fmt.Errorf("ACTLABS_SERVER_SERVICE_PRINCIPAL_OBJECT_ID not set")
	}

	actlabsServerServicePrincipalClientSecretKeyvaultURL := getEnv("ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL")
	if actlabsServerServicePrincipalClientSecretKeyvaultURL == "" && actlabsServerUseServicePrincipal {
		return nil, fmt.Errorf("ACTLABS_SERVER_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL not set")
	}

	actlabsServerFdpoServicePrincipalClientId := getEnv("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_ID")
	if actlabsServerFdpoServicePrincipalClientId == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_ID not set")
	}

	actlabsServerFdpoServicePrincipalObjectId := getEnv("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_OBJECT_ID")
	if actlabsServerFdpoServicePrincipalObjectId == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_OBJECT_ID not set")
	}

	actlabsServerFdpoServicePrincipalSecret := getEnv("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_SECRET")
	if actlabsServerFdpoServicePrincipalSecret == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_SECRET not set")
	}

	fdpoTenantID := getEnv("FDPO_TENANT_ID")
	if fdpoTenantID == "" {
		return nil, fmt.Errorf("FDPO_TENANT_ID not set")
	}

	actlabsServerFdpoServicePrincipalClientSecretKeyvaultURL := getEnv("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL")
	if actlabsServerFdpoServicePrincipalClientSecretKeyvaultURL == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_FDPO_SERVICE_PRINCIPAL_CLIENT_SECRET_KEYVAULT_URL not set")
	}

	actlabsHubUseMsi, err := strconv.ParseBool(getEnvWithDefault("ACTLABS_HUB_USE_MSI", "false"))
	if err != nil {
		return nil, err
	}

	actlabsHubUseUserAuth, err := strconv.ParseBool(getEnvWithDefault("ACTLABS_HUB_USE_USER_AUTH", "false"))
	if err != nil {
		return nil, err
	}

	actlabsServerUPWaitTimeSeconds := getEnvWithDefault("ACTLABS_SERVER_UP_WAIT_TIME_SECONDS", "180")
	if actlabsServerUPWaitTimeSeconds == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_UP_WAIT_TIME_SECONDS not set")
	}

	actlabsServerManagedEnvironmentId := getEnv("ACTLABS_SERVER_MANAGED_ENVIRONMENT_ID")
	if actlabsServerManagedEnvironmentId == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_MANAGED_ENVIRONMENT_ID not set")
	}

	actlabsServerResourceGroup := getEnv("ACTLABS_SERVER_RESOURCE_GROUP")
	if actlabsServerResourceGroup == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_RESOURCE_GROUP not set")
	}

	actlabsServerPort, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_SERVER_PORT", "8881"), 10, 32)
	if err != nil {
		return nil, fmt.Errorf("ACTLABS_SERVER_PORT not set")
	}

	actlabsServerImage := getEnvWithDefault("ACTLABS_SERVER_IMAGE", "ashishvermapu/repro:latest")
	if actlabsServerImage == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_IMAGE not set")
	}

	actlabsHubURL := getEnv("ACTLABS_HUB_URL")
	if actlabsHubURL == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_URL not set")
	}

	httpPort, err := strconv.Atoi(getEnvWithDefault("HTTP_PORT", "80"))
	if err != nil {
		return nil, fmt.Errorf("HTTP_PORT not set")
	}

	httpsPort, err := strconv.Atoi(getEnvWithDefault("HTTPS_PORT", "443"))
	if err != nil {
		return nil, fmt.Errorf("HTTPS_PORT not set")
	}

	actlabsServerReadinessProbePath := getEnvWithDefault("ACTLABS_SERVER_READINESS_PROBE_PATH", "/status")
	if actlabsServerReadinessProbePath == "" {
		return nil, fmt.Errorf("ACTLABS_SERVER_READINESS_PROBE_PATH not set")
	}

	tenantID := getEnv("TENANT_ID")
	if tenantID == "" {
		return nil, fmt.Errorf("TENANT_ID not set")
	}

	actlabsServerCPUFloat, err := strconv.ParseFloat(getEnvWithDefault("ACTLABS_SERVER_CPU", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerMemoryFloat, err := strconv.ParseFloat(getEnvWithDefault("ACTLABS_SERVER_MEMORY", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerCaddyCPUFloat, err := strconv.ParseFloat(getEnvWithDefault("ACTLABS_SERVER_CADDY_CPU", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerCaddyMemoryFloat, err := strconv.ParseFloat(getEnvWithDefault("ACTLABS_SERVER_CADDY_MEMORY", "0.5"), 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeInitialDelaySecondsInt, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_SERVER_READINESS_PROBE_INITIAL_DELAY_SECONDS", "10"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeTimeoutSecondsInt, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_SERVER_READINESS_PROBE_TIMEOUT_SECONDS", "5"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbePeriodSecondsInt, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_SERVER_READINESS_PROBE_PERIOD_SECONDS", "10"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeSuccessThresholdInt, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_SERVER_READINESS_PROBE_SUCCESS_THRESHOLD", "1"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsServerReadinessProbeFailureThresholdInt, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_SERVER_READINESS_PROBE_FAILURE_THRESHOLD", "20"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsHubClientID := getEnv("ACTLABS_HUB_CLIENT_ID")
	if actlabsHubClientID == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_CLIENT_ID not set")
	}

	actlabsHubSubscriptionID := getEnv("ACTLABS_HUB_SUBSCRIPTION_ID")
	if actlabsHubSubscriptionID == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_SUBSCRIPTION_ID not set")
	}

	actlabsHubResourceGroup := getEnv("ACTLABS_HUB_RESOURCE_GROUP")
	if actlabsHubResourceGroup == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_RESOURCE_GROUP not set")
	}

	actlabsHubStorageAccount := getEnv("ACTLABS_HUB_STORAGE_ACCOUNT")
	if actlabsHubStorageAccount == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_STORAGE_ACCOUNT not set")
	}

	actlabsHubManagedServersTableName := getEnv("ACTLABS_HUB_MANAGED_SERVERS_TABLE_NAME")
	if actlabsHubManagedServersTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_MANAGED_SERVERS_TABLE_NAME not set")
	}

	actlabsHubReadinessAssignmentsTableName := getEnv("ACTLABS_HUB_READINESS_ASSIGNMENTS_TABLE_NAME")
	if actlabsHubReadinessAssignmentsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_READINESS_ASSIGNMENTS_TABLE_NAME not set")
	}

	actlabsHubChallengesTableName := getEnv("ACTLABS_HUB_CHALLENGES_TABLE_NAME")
	if actlabsHubChallengesTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_CHALLENGES_TABLE_NAME not set")
	}

	actlabsHubProfilesTableName := getEnv("ACTLABS_HUB_PROFILES_TABLE_NAME")
	if actlabsHubProfilesTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_PROFILES_TABLE_NAME not set")
	}

	actlabsHubDeploymentsTableName := getEnv("ACTLABS_HUB_DEPLOYMENTS_TABLE_NAME")
	if actlabsHubDeploymentsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_DEPLOYMENTS_TABLE_NAME not set")
	}

	actlabsHubEventsTableName := getEnv("ACTLABS_HUB_EVENTS_TABLE_NAME")
	if actlabsHubEventsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_EVENTS_TABLE_NAME not set")
	}

	actlabsHubDeploymentOperationsTableName := getEnv("ACTLABS_HUB_DEPLOYMENT_OPERATIONS_TABLE_NAME")
	if actlabsHubDeploymentOperationsTableName == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_DEPLOYMENT_OPERATIONS_TABLE_NAME not set")
	}

	actlabsHubManagedIdentityResourceId := getEnv("ACTLABS_HUB_MANAGED_IDENTITY_RESOURCE_ID")
	if actlabsHubManagedIdentityResourceId == "" {
		return nil, fmt.Errorf("ACTLABS_HUB_MANAGED_IDENTITY_RESOURCE_ID not set")
	}

	actlabsHubAutoDestroyPollingIntervalSeconds, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_HUB_AUTO_DESTROY_POLLING_INTERVAL_SECONDS", "600"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsHubAutoDestroyIdleTimeSeconds, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_HUB_AUTO_DESTROY_IDLE_TIME_SECONDS", "3600"), 10, 32)
	if err != nil {
		return nil, err
	}

	actlabsHubDeploymentsPollingIntervalSeconds, err := strconv.ParseInt(getEnvWithDefault("ACTLABS_HUB_DEPLOYMENTS_POLLING_INTERVAL_SECONDS", "300"), 10, 32)
	if err != nil {
		return nil, err
	}

	// Retrieve other environment variables and check them as needed

	return &Config{
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
		AuthTokenAud:                                             authTokenAud,
		AuthTokenIss:                                             authTokenIss,
		HttpPort:                                                 int32(httpPort),
		HttpsPort:                                                int32(httpsPort),
		ProtectedLabSecret:                                       protectedLabSecret,
		TenantID:                                                 tenantID,
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
		// Set other fields
	}, nil
}

// Helper function to retrieve the value and log it
func getEnv(env string) string {
	value := os.Getenv(env)
	slog.Info("environment variable", slog.String("name", env), slog.String("value", value))
	return value
}

// Helper function to retrieve the value, if none found, set default and log it
func getEnvWithDefault(env string, defaultValue string) string {
	value := os.Getenv(env)
	if value == "" {
		value = defaultValue
	}
	slog.Info("environment variable", slog.String("name", env), slog.String("value", value))
	return value
}
