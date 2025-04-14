#!/bin/bash

ACTLABS_SP_APP_ID="00399ddd-434c-4b8a-84be-d096cff4f494"

# If user is in fdpo tenant, then script will replace ACTLABS_SP_APP_ID with ACTLABS_FDPO_SP_APP_ID
ACTLABS_FDPO_SP_APP_ID="50cc6d33-3224-477f-b2bd-5c1c6595fdf5"
ACTLABS_MSI_APP_ID="9735b762-ef8d-477b-af26-13c9b8d6f35c"
RESOURCE_GROUP="repro-project"

# Add some color
RED='\033[0;91m'
GREEN='\033[0;92m'
YELLOW='\033[0;93m'
PURPLE='\033[0;95m'
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
    ok "${MESSAGE}"
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
  USER_PRINCIPAL_ID=$(az ad signed-in-user show --query id -o tsv)
  TENANT_ID=$(az account show --query tenantId -o tsv)
  log "UPN: $UPN"

  # drop the domain name from the upn
  if [[ "${UPN}" == *"fdpo.onmicrosoft.com"* ]]; then
    log "FDPO Tenant"
    ACTLABS_SP_APP_ID=${ACTLABS_FDPO_SP_APP_ID}
    USER_ALIAS=${UPN%%_*}
    is_fdpo=true
    ENV="fdpo"
    # handle_error "We currently do not support Microsoft Non-Prod Tenant. Please reach out to the team for support."
  else
    USER_ALIAS=${UPN%%@*}
    is_fdpo=false
    ENV="prod"
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
    is_owner=true
  else
    warn "user ${UPN} is not the owner of the subscription. Ensure that 'Owner' has already registered the subcription with the actlabs. If not, please ask the Owner the subscription with the actlabs"
    is_owner=false
  fi

  return 0
}

# Function to ensure that user is the contributor of the subscription
ensure_user_is_contributor() {
  # Check if the user is the contributor of the subscription
  USER_ROLE=$(az role assignment list --assignee "${UPN}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='Contributor'].roleDefinitionName" -o tsv)

  if [[ -n "${USER_ROLE}" ]]; then
    log "user ${UPN} is the contributor of the subscription"
    is_contributor=true
  else
    err "user ${UPN} is not the contributor of the subscription. Either 'Owner' or 'Contributor' role is required to proceed"
    exit 1
  fi

  return 0
}

# Function to ensure that identity has contributor and user access administrator role on the subscription
function ensure_required_roles() {

  IDENTITY=$1
  if [[ -z "${IDENTITY}" ]]; then
    err "identity is not provided"
    exit 1
  fi
  # Check if the identity is 'Contributor' on the subscription
  SP_ROLE=$(az role assignment list --assignee "${IDENTITY}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='Contributor'].roleDefinitionName" -o tsv)
  if [[ -n "${SP_ROLE}" ]]; then
    log "actlabs ID ${IDENTITY} is already 'Contributor' on the subscription"
  else
    err "actlabs ID ${IDENTITY} is not 'Contributor' on the subscription. Registration requires 'Owner' access to the subscription. Please ask the Owner to register the subscription with the actlabs"
    exit 1
  fi

  # Check if the identity is 'User Access Administrator' on the subscription
  SP_ROLE=$(az role assignment list --assignee "${IDENTITY}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='User Access Administrator'].roleDefinitionName" -o tsv)
  if [[ -n "${SP_ROLE}" ]]; then
    log "actlabs ID ${IDENTITY} is already 'User Access Administrator' on the subscription"
  else
    err "actlabs ID ${IDENTITY} is not 'User Access Administrator' on the subscription. Registration requires 'Owner' access to the subscription. Please ask the Owner to register the subscription with the actlabs"
    exit 1
  fi
}

# Function to check if identity is 'Contributor' on the subscription
# If not, assign the identity the 'Contributor' role on the subscription
function assign_contributor_role() {

  IDENTITY=$1
  if [[ -z "${IDENTITY}" ]]; then
    err "identity is not provided"
    exit 1
  fi

  # Check if the identity is 'Contributor' on the subscription
  SP_ROLE=$(az role assignment list --assignee "${IDENTITY}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='Contributor'].roleDefinitionName" -o tsv)

  if [[ -n "${SP_ROLE}" ]]; then
    log "actlabs ID ${IDENTITY} is already 'Contributor' on the subscription"
  else
    log "assigning actlabs ID ${IDENTITY} 'Contributor' role on the subscription"
    # Assign the identity the 'Contributor' role on the subscription
    az role assignment create --assignee "${IDENTITY}" --role Contributor --scope "/subscriptions/${SUBSCRIPTION_ID}"
    if [ $? -ne 0 ]; then
      err "failed to assign actlabs ID ${IDENTITY} 'Contributor' role on the subscription"
      exit 1
    else
      log "actlabs ID ${IDENTITY} assigned 'Contributor' role on the subscription"
    fi
  fi

  return 0
}

# Function to check if given identity is 'User Access Administrator' on the subscription
# If not, assign the given identity the 'User Access Administrator' role on the subscription
function assign_user_access_administrator_role() {

  IDENTITY=$1
  if [[ -z "${IDENTITY}" ]]; then
    err "identity is not provided"
    exit 1
  fi

  # Check if the identity is 'User Access Administrator' on the subscription
  SP_ROLE=$(az role assignment list --assignee "${IDENTITY}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='User Access Administrator'].roleDefinitionName" -o tsv)

  if [[ -n "${SP_ROLE}" ]]; then
    log "actlabs ID ${IDENTITY} is already 'User Access Administrator' on the subscription"
  else
    log "assigning actlabs ID ${IDENTITY} 'User Access Administrator' role on the subscription"
    # Assign the identity the 'User Access Administrator' role on the subscription
    az role assignment create --assignee "${IDENTITY}" --role "User Access Administrator" --scope "/subscriptions/${SUBSCRIPTION_ID}"
    if [ $? -ne 0 ]; then
      err "failed to assign actlabs ID ${IDENTITY} 'User Access Administrator' role on the subscription"
      exit 1
    else
      log "actlabs ID ${IDENTITY} assigned 'User Access Administrator' role on the subscription"
    fi
  fi

  return 0
}

# Function to call register api to register the subscription with the lab
function register_subscription() {
  # get access token from cli
  log "getting access token from cli"
  ACCESS_TOKEN=$(az account get-access-token --query accessToken -o tsv)
  log "registering subscription with the lab"

  OUTPUT=$(curl -X PUT \
    https://actlabs-hub-capp.purplegrass-7409b036.eastus.azurecontainerapps.io/arm/server/register \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -d '{
        "userAlias": "'"${USER_ALIAS}"'",
        "userPrincipalId": "'"${USER_PRINCIPAL_ID}"'",
        "subscriptionId": "'"${SUBSCRIPTION_ID}"'",
        "tenantId": "'"${TENANT_ID}"'"
      }' \
    -w "\n%{http_code}")

  log "Output: $OUTPUT"

  HTTP_STATUS="${OUTPUT:(-3)}"

  if [ "${HTTP_STATUS}" -ne 200 ]; then
    err "deployment completed, but, failed to automatically register subscription with the lab"
    log "Next steps: Try re-running the script again. If error persists, please contact support using Help & Feedback option on portal and share your subscription id ${SUBSCRIPTION_ID}"
    exit 0
  else
    log "Output: $OUTPUT"
    ok "subscription ${SUBSCRIPTION_ID} registered for user ${UPN}"
  fi
}

# Call the functions
get_upn
get_subscription_id
ensure_user_is_owner
if [[ "${is_owner}" == false ]]; then
  ensure_user_is_contributor
  ensure_required_roles "${ACTLABS_MSI_APP_ID}"
  ensure_required_roles "${ACTLABS_SP_APP_ID}"
fi
if [[ "${is_owner}" == true ]]; then
  assign_contributor_role "${ACTLABS_SP_APP_ID}"
  assign_contributor_role "${ACTLABS_MSI_APP_ID}"
  assign_user_access_administrator_role "${ACTLABS_SP_APP_ID}"
  assign_user_access_administrator_role "${ACTLABS_MSI_APP_ID}"

fi
register_subscription
