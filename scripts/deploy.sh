#!/bin/bash

# Load environment variables from .env file
export $(egrep -v '^#' .prod.containerapp.env | xargs)

# Create the environment
az containerapp env create --name actlabs-hub-env-eastus \
    --resource-group actlabs-app \
    --subscription ACT-CSS-Readiness \
    --location eastus \
    --logs-destination none

if [ $? -ne 0 ]; then
    echo "Failed to create environment"
    exit 1
fi

# Check if 'beta' argument is passed
if [ "$1" == "beta" ]; then
    APP_NAME="actlabs-hub-capp-beta"
else
    APP_NAME="actlabs-hub-capp"
fi

# Deploy the Container App
az containerapp create \
    --name actlabs-hub-capp-beta \
    --resource-group actlabs-app \
    --subscription ACT-CSS-Readiness \
    --environment actlabs-hub-env-eastus \
    --allow-insecure false \
    --image ashishvermapu/actlabs-hub:latest \
    --ingress 'external' \
    --min-replicas 1 \
    --max-replicas 1 \
    --target-port $ACTLABS_HUB_PORT \
    --user-assigned $ACTLABS_HUB_MSI \
    --env-vars \
        "ACTLABS_HUB_URL=$ACTLABS_HUB_URL" \
        "ACTLABS_HUB_LOG_LEVEL=$ACTLABS_HUB_LOG_LEVEL" \
        "ACTLABS_HUB_SUBSCRIPTION_ID=$ACTLABS_HUB_SUBSCRIPTION_ID" \
        "ACTLABS_HUB_RESOURCE_GROUP=$ACTLABS_HUB_RESOURCE_GROUP" \
        "ACTLABS_HUB_STORAGE_ACCOUNT=$ACTLABS_HUB_STORAGE_ACCOUNT" \
        "ACTLABS_HUB_MANAGED_SERVERS_TABLE_NAME=$ACTLABS_HUB_MANAGED_SERVERS_TABLE_NAME" \
        "ACTLABS_HUB_READINESS_ASSIGNMENTS_TABLE_NAME=$ACTLABS_HUB_READINESS_ASSIGNMENTS_TABLE_NAME" \
        "ACTLABS_HUB_CHALLENGES_TABLE_NAME=$ACTLABS_HUB_CHALLENGES_TABLE_NAME" \
        "ACTLABS_HUB_PROFILES_TABLE_NAME=$ACTLABS_HUB_PROFILES_TABLE_NAME" \
        "ACTLABS_HUB_DEPLOYMENTS_TABLE_NAME=$ACTLABS_HUB_DEPLOYMENTS_TABLE_NAME" \
        "ACTLABS_HUB_DEPLOYMENT_OPERATIONS_TABLE_NAME=$ACTLABS_HUB_DEPLOYMENT_OPERATIONS_TABLE_NAME" \
        "ACTLABS_HUB_CLIENT_ID=$ACTLABS_HUB_CLIENT_ID" \
        "ACTLABS_HUB_USE_MSI=$ACTLABS_HUB_USE_MSI" \
        "PORT=$ACTLABS_HUB_PORT" \
        "ACTLABS_HUB_AUTO_DESTROY_POLLING_INTERVAL_SECONDS=$ACTLABS_HUB_AUTO_DESTROY_POLLING_INTERVAL_SECONDS" \
        "ACTLABS_HUB_AUTO_DESTROY_IDLE_TIME_SECONDS=$ACTLABS_HUB_AUTO_DESTROY_IDLE_TIME_SECONDS" \
        "ACTLABS_HUB_DEPLOYMENTS_POLLING_INTERVAL_SECONDS=$ACTLABS_HUB_DEPLOYMENTS_POLLING_INTERVAL_SECONDS" \
        "ACTLABS_SERVER_PORT=$ACTLABS_SERVER_PORT" \
        "ACTLABS_SERVER_READINESS_PROBE_PATH=$ACTLABS_SERVER_READINESS_PROBE_PATH" \
        "ACTLABS_SERVER_ROOT_DIR=$ACTLABS_SERVER_ROOT_DIR" \
        "ACTLABS_SERVER_UP_WAIT_TIME_SECONDS=$ACTLABS_SERVER_UP_WAIT_TIME_SECONDS" \
        "ACTLABS_SERVER_USE_MSI=$ACTLABS_SERVER_USE_MSI" \
        "ACTLABS_SERVER_CPU=$ACTLABS_SERVER_CPU" \
        "ACTLABS_SERVER_MEMORY=$ACTLABS_SERVER_MEMORY" \
        "AUTH_TOKEN_AUD=$AUTH_TOKEN_AUD" \
        "AUTH_TOKEN_ISS=$AUTH_TOKEN_ISS" \
        "HTTPS_PORT=$HTTPS_PORT" \
        "HTTP_PORT=$HTTP_PORT" \
        "PROTECTED_LAB_SECRET=$PROTECTED_LAB_SECRET" \
        "TENANT_ID=$TENANT_ID"

if [ $? -ne 0 ]; then
    echo "Failed to create container app"
    exit 1
fi