#!/bin/bash

ACTLABS_APP_ID="bee16ca1-a401-40ee-bb6a-34349ebd993e"
RESOURCE_GROUP="repro-project"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Function to log messages
err() {
  echo -e "${RED}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: ERROR - $* ${NC}" >&1
}

warn() {
  echo -e "${YELLOW}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: WARNING - $* ${NC}" >&1
}

log() {
  echo -e "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: INFO - $*" >&1
}

ok() {
  echo -e "${GREEN}[$(date +'%Y-%m-%dT%H:%M:%S%z')]: SUCCESS - $* ${NC}" >&1
}

gap() {
  echo >&1
}

# Function that sleeps for a specified number of seconds
function sleep_with_progress() {
  local TOTAL_SECONDS=$1
  local MESSAGE=$2 # optional; if provided, will be printed before the countdown

  # Check if TOTAL_SECONDS is a positive integer
  if ! [[ "$TOTAL_SECONDS" =~ ^[0-9]+$ ]]; then
    log "Error: Invalid number of seconds"
    exit 1
  fi

  if [[ -n "${MESSAGE}" ]]; then
    log "${MESSAGE}"
  fi

  local MINUTES=$((TOTAL_SECONDS / 60))
  local SECONDS_REMAINING=$((TOTAL_SECONDS % 60))
  log "Sleeping for ${MINUTES} minutes and ${SECONDS_REMAINING} seconds..."

  while [[ ${TOTAL_SECONDS} -gt 0 ]]; do
    MINUTES=$((TOTAL_SECONDS / 60))
    SECONDS_REMAINING=$((TOTAL_SECONDS % 60))
    log "${MINUTES} minutes and ${SECONDS_REMAINING} seconds remaining"
    sleep 10
    TOTAL_SECONDS=$((TOTAL_SECONDS - 10))
  done
}

# Function to handle errors
handle_error() {
  err "$1"
  exit 1
}

# Function to get upn of the logged in user
get_upn() {
  UPN=$(az ad signed-in-user show --query userPrincipalName -o tsv)
  log "UPN: $UPN"

  # drop the domain name from the upn
  if [[ "${UPN}" == *"fdpo.onmicrosoft.com"* ]]; then
    # USER_ALIAS=${UPN%%_*}
    handle_error "We currently do not support Microsoft Non-Prod Tenant. Please reach out to the team for support."
  else
    USER_ALIAS=${UPN%%@*}
  fi
  log "USER_ALIAS: $USER_ALIAS"

  USER_ALIAS_FOR_SA=${USER_ALIAS#v-}
  log "USER_ALIAS_FOR_SA: $USER_ALIAS_FOR_SA"
}

# Function to get the subscription id
get_subscription_id() {
  SUBSCRIPTION_ID=$(az account show --query id -o tsv)
  log "SUBSCRIPTION_ID: $SUBSCRIPTION_ID"
}

# Function to ensure that user is the owner of the subscription
ensure_user_is_owner() {
  # Check if the user is the owner of the subscription
  USER_ROLE=$(az role assignment list --assignee "${UPN}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='Owner'].roleDefinitionName" -o tsv)

  if [[ -n "${USER_ROLE}" ]]; then
    log "user ${UPN} is the owner of the subscription"
  else
    err "user ${UPN} is not the owner of the subscription"
    exit 1
  fi

  return 0
}

# Function to check if a resource group exists
# If the resource group doesn't exist, create one
function create_resource_group() {
  # Check if the resource group exists
  RG_EXISTS=$(az group exists --name "${RESOURCE_GROUP}" --output tsv)

  if [[ "${RG_EXISTS}" == "true" ]]; then
    log "resource group ${RESOURCE_GROUP} already exists"
    return 0
  else
    log "creating resource group with name ${RESOURCE_GROUP}"

    # Ask the user for a location if one wasn't provided
    # if [[ -z "${LOCATION}" ]]; then
    #   gap
    #   LOCATION=$(az account list-locations --query "[?metadata.regionType!='Logical' && metadata.physicalLocation!=null].name" -o tsv)
    #   echo "Please select a location (azure region) from the list below:"
    #   select LOCATION in ${LOCATION}; do
    #     if [[ -n "${LOCATION}" ]]; then
    #       break
    #     else
    #       echo "Invalid selection, please try again"
    #     fi
    #   done
    # fi

    export LOCATION="eastus"

    # Create the resource group
    az group create --name "${RESOURCE_GROUP}" --location "${LOCATION}"
    if [ $? -ne 0 ]; then
      err "failed to create resource group ${RESOURCE_GROUP}"
      exit 1
    else
      log "resource group ${RESOURCE_GROUP} created"
    fi
  fi

  return 0
}

# Function to check if a storage account exists in the resource group
# If the storage account doesn't exist, create one with a random name
function create_storage_account() {
  # Check if the storage account exists in the resource group
  SA_EXISTS=$(az storage account list --resource-group "${RESOURCE_GROUP}" --query "[].name" -o tsv)

  if [[ -n "${SA_EXISTS}" ]]; then
    log "storage account already exists with name ${SA_EXISTS}"
    STORAGE_ACCOUNT_NAME="$SA_EXISTS"
  else
    # Generate a random name for the storage account
    RANDOM_NAME=$(openssl rand -hex 4)
    STORAGE_ACCOUNT_NAME="${USER_ALIAS_FOR_SA}sa${RANDOM_NAME}"

    log "creating storage account with name ${STORAGE_ACCOUNT_NAME} in resource group ${RESOURCE_GROUP}"
    # Create the storage account
    az storage account create --name "${STORAGE_ACCOUNT_NAME}" --resource-group "${RESOURCE_GROUP}" --sku Standard_LRS
    if [ $? -ne 0 ]; then
      err "failed to create storage account ${STORAGE_ACCOUNT_NAME}"
      exit 1
    else
      log "storage account ${STORAGE_ACCOUNT_NAME} created"
    fi

    # Wait for a minute to let storage account sync
    sleep_with_progress 60 "Waiting for storage account to sync with azure"
  fi

  # get storage account key
  STORAGE_ACCOUNT_KEY=$(az storage account keys list --resource-group "${RESOURCE_GROUP}" --account-name "${STORAGE_ACCOUNT_NAME}" --query "[0].value" -o tsv)

  # check if a blob container named 'tfstate' exists in the storage account
  # if not create one
  log "checking if blob container tfstate exists in storage account ${STORAGE_ACCOUNT_NAME}"
  CONTAINER_EXISTS=$(az storage container exists --name "tfstate" --account-name "${STORAGE_ACCOUNT_NAME}" --account-key "${STORAGE_ACCOUNT_KEY}" --query "exists" -o tsv)
  if [[ "${CONTAINER_EXISTS}" == "true" ]]; then
    log "Blob container tfstate already exists in storage account ${STORAGE_ACCOUNT_NAME}"
  else
    log "Blob container tfstate does not exist in storage account ${STORAGE_ACCOUNT_NAME}, creating"
    az storage container create --name "tfstate" --account-name "${STORAGE_ACCOUNT_NAME}"
    if [ $? -ne 0 ]; then
      err "Failed to create blob container tfstate in storage account ${STORAGE_ACCOUNT_NAME}"
      exit 1
    else
      log "Blob container tfstate created in storage account ${STORAGE_ACCOUNT_NAME}"
    fi
  fi

  # check if a blob container named 'labs' exists in the storage account
  # if not create one
  log "checking if blob container labs exists in storage account ${STORAGE_ACCOUNT_NAME}"
  CONTAINER_EXISTS=$(az storage container exists --name "labs" --account-name "${STORAGE_ACCOUNT_NAME}" --account-key "${STORAGE_ACCOUNT_KEY}" --query "exists" -o tsv)
  if [[ "${CONTAINER_EXISTS}" == "true" ]]; then
    log "Blob container labs already exists in storage account ${STORAGE_ACCOUNT_NAME}"
  else
    log "Blob container labs does not exist in storage account ${STORAGE_ACCOUNT_NAME}, creating"
    az storage container create --name "labs" --account-name "${STORAGE_ACCOUNT_NAME}"
    if [ $? -ne 0 ]; then
      err "Failed to create blob container labs in storage account ${STORAGE_ACCOUNT_NAME}"
      exit 1
    else
      log "Blob container labs created in storage account ${STORAGE_ACCOUNT_NAME}"
    fi
  fi

  return 0
}

# Function to check if current user has 'Storage Blob Data Contributor' on the resource group repro-project
# If not, assign the role to the user
function assign_storage_blob_data_contributor_role() {
  # Check if the user has 'Storage Blob Data Contributor' role on the resource group
  USER_ROLE=$(az role assignment list --assignee "${UPN}" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" --query "[?roleDefinitionName=='Storage Blob Data Contributor'].roleDefinitionName" -o tsv)

  if [[ -n "${USER_ROLE}" ]]; then
    log "user ${UPN} already has 'Storage Blob Data Contributor' role on resource group ${RESOURCE_GROUP}"
  else
    log "assigning 'Storage Blob Data Contributor' role to user ${UPN} on resource group ${RESOURCE_GROUP}"
    az role assignment create --assignee "${UPN}" --role "Storage Blob Data Contributor" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}"
    if [ $? -ne 0 ]; then
      err "failed to assign 'Storage Blob Data Contributor' role to user ${UPN} on resource group ${RESOURCE_GROUP}"
      exit 1
    else
      log "assigned 'Storage Blob Data Contributor' role to user ${UPN} on resource group ${RESOURCE_GROUP}"
    fi
  fi

  return 0
}

function deploy() {
  # Environment Variables
  export PORT="80"
  export ROOT_DIR="/app"
  export USE_MSI="false"
  export USE_SERVICE_PRINCIPAL="false"
  export ARM_SUBSCRIPTION_ID=$(az account show --query "id" -o tsv)
  export ARM_TENANT_ID=$(az account show --query "tenantId" -o tsv)
  export ARM_USER_PRINCIPAL_NAME=$(az account show --query "user.name" -o tsv)
  export AZURE_CLIENT_ID="not-used-for-self-hosting"
  export AZURE_SUBSCRIPTION_ID=$(az account show --query "id" -o tsv)
  export AUTH_TOKEN_ISS="https://login.microsoftonline.com/72f988bf-86f1-41af-91ab-2d7cd011db47/v2.0"
  export AUTH_TOKEN_AUD="00399ddd-434c-4b8a-84be-d096cff4f494"
  export ACTLABS_HUB_URL="https://actlabs-hub-capp.redisland-ff4b63ab.eastus.azurecontainerapps.io/"

  # Start docker container and set environment variables
  log "starting docker container"
  docker run --pull=always -d --restart unless-stopped -it \
    -e PORT \
    -e ROOT_DIR \
    -e USE_MSI \
    -e USE_SERVICE_PRINCIPAL \
    -e LOG_LEVEL \
    -e ARM_SUBSCRIPTION_ID \
    -e ARM_TENANT_ID \
    -e ARM_USER_PRINCIPAL_NAME \
    -e AZURE_CLIENT_ID \
    -e AZURE_SUBSCRIPTION_ID \
    -e AUTH_TOKEN_AUD \
    -e AUTH_TOKEN_ISS \
    -e ACTLABS_HUB_URL \
    --name actlabs -p 8880:80 -v ${HOME}/.azure:/root/.azure ashishvermapu/repro:${TAG}
  if [ $? -ne 0 ]; then
    err "Failed to start docker container"
    exit 1
  fi

  return 0
}

function verify() {
  # Verify that the application is running
  log "verifying that the application is running"
  curl --silent --fail --show-error --max-time 10 --retry 30 --retry-delay 10 http://localhost:8880/status >/dev/null 2>&1
  if [ $? -ne 0 ]; then
    err "Failed to verify that the application is running"
    exit 1
  fi
  ok "All done! You can now access the application at https://actlabs.azureedge.net/. Your server's endpoint is http://localhost:8880/"

  return 0
}

# gather input parameters
# -t tag
# -d debug

while getopts ":t:d" opt; do
  case $opt in
  t)
    TAG=$OPTARG
    ;;
  d)
    DEBUG=true
    ;;
  \?)
    echo "Invalid option: -$OPTARG" >&2
    ;;
  esac
done

# delete existing docker container
log "deleting existing docker container"
docker rm -f actlabs

if [ -z "${TAG}" ]; then
  TAG="latest"
fi

if [ ${DEBUG} ]; then
  export LOG_LEVEL="-4"
else
  export LOG_LEVEL="0"
fi

log "TAG = ${TAG}"
log "DEBUG = ${DEBUG}"

# Call the functions
get_upn
get_subscription_id
ensure_user_is_owner
create_resource_group
create_storage_account
assign_storage_blob_data_contributor_role
deploy
sleep_with_progress 10 "Waiting for the application to start"
verify
