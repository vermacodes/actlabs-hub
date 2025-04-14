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
    warn "user ${UPN} is not the owner of the subscription. Ensure that 'Owner' has already registered the subscription with the actlabs. If not, please ask the Owner the subscription with the actlabs"
    is_owner=false
  fi

  return 0
}

# Function to delete a role assignment
# Usage: delete_role <role_name> <scope> <assignee>
# Example: delete_role "Storage Blob Data Contributor" "/subscriptions/<subscription_id>/resourceGroups/<resource_group>" "<user_principal_name>"
function delete_role() {
  local role_name=$1
  # Check if the role name is provided
  if [[ -z "${role_name}" ]]; then
    err "Role name is required."
    return 1
  fi

  local scope=$2
  # Check if the scope is provided
  if [[ -z "${scope}" ]]; then
    err "Scope is required."
    return 1
  fi

  local assignee=$3
  # Check if the assignee is provided
  if [[ -z "${assignee}" ]]; then
    err "Assignee is required."
    return 1
  fi

  # Check if the role assignment exists
  ROLE_ASSIGNMENT_EXISTS=$(az role assignment list --assignee "${assignee}" --role "${role_name}" --scope "${scope}" --query "[?roleDefinitionName=='${role_name}'].roleDefinitionName" -o tsv)
  if [[ -z "${ROLE_ASSIGNMENT_EXISTS}" ]]; then
    log "Role assignment '${role_name}' for ${assignee} within ${scope} does not exist. No need to delete."
    return 0
  fi

  # Delete the role assignment
  az role assignment delete --assignee "${assignee}" --role "${role_name}" --scope "${scope}"

  if [ $? -ne 0 ]; then
    warn "Failed to delete the '${role_name}' role for ${assignee} within ${scope}. Please remove it manually."
  else
    log "The '${role_name}' role for ${assignee} has been successfully deleted from ${scope}."
  fi
}

# Function to delete resource group
function delete_resource_group() {
  # Check if the resource group exists
  RESOURCE_GROUP_EXISTS=$(az group exists --name "${RESOURCE_GROUP}")
  if [ "${RESOURCE_GROUP_EXISTS}" == "false" ]; then
    log "Resource group ${RESOURCE_GROUP} does not exist. No need to delete."
    return 0
  fi
  # Delete the resource group
  log "Deleting resource group ${RESOURCE_GROUP}..."
  az group delete --name "${RESOURCE_GROUP}" --yes
  if [ $? -ne 0 ]; then
    err "failed to delete resource group ${RESOURCE_GROUP}"
    exit 1
  else
    log "resource group ${RESOURCE_GROUP} deleted"
  fi
}

# Call the functions
get_upn
get_subscription_id
ensure_user_is_owner
delete_role "Storage Blob Data Contributor" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_SP_APP_ID}"
delete_role "Storage Blob Data Contributor" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_SP_APP_ID}"
delete_role "Storage Blob Data Contributor" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_MSI_APP_ID}"
delete_role "Storage Blob Data Contributor" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_MSI_APP_ID}"
delete_role "User Access Administrator" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_SP_APP_ID}"
delete_role "User Access Administrator" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_SP_APP_ID}"
delete_role "User Access Administrator" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_MSI_APP_ID}"
delete_role "User Access Administrator" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_MSI_APP_ID}"
delete_role "Contributor" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_SP_APP_ID}"
delete_role "Contributor" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_SP_APP_ID}"
delete_role "Contributor" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_MSI_APP_ID}"
delete_role "Contributor" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_MSI_APP_ID}"
delete_role "Reader" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_SP_APP_ID}"
delete_role "Reader" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_SP_APP_ID}"
delete_role "Reader" "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" "${ACTLABS_MSI_APP_ID}"
delete_role "Reader" "/subscriptions/${SUBSCRIPTION_ID}" "${ACTLABS_MSI_APP_ID}"
delete_resource_group
ok "Unregistering complete. Thank you for using the service."
