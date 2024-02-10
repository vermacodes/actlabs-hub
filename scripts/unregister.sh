#!/bin/bash

ACTLABS_SP_APP_ID="00399ddd-434c-4b8a-84be-d096cff4f494"
ACTLABS_MSI_APP_ID="bee16ca1-a401-40ee-bb6a-34349ebd993e"
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
    err "user ${UPN} is not the owner of the subscription."
    exit 1
  fi
  return 0
}

# Function to delete resource group
function delete_resource_group() {
  az group delete --name "${RESOURCE_GROUP}" --yes
  if [ $? -ne 0 ]; then
    err "failed to delete resource group ${RESOURCE_GROUP}"
    exit 1
  else
    log "resource group ${RESOURCE_GROUP} deleted"
  fi
}

# Function to delete 'Contributor' role from the subscription
function delete_contributor_role() {
  az role assignment delete --assignee "${ACTLABS_SP_APP_ID}" --role Contributor --scope "/subscriptions/${SUBSCRIPTION_ID}"
  if [ $? -ne 0 ]; then
    warn "failed to delete actlabs 'Contributor' role from the subscription. Please delete manually or contact support."
  else
    log "actlabs 'Contributor' role deleted from the subscription"
  fi
}

# Function to delete 'User Access Administrator' role from the subscription
function delete_user_access_administrator_role() {
  az role assignment delete --assignee "${ACTLABS_SP_APP_ID}" --role "User Access Administrator" --scope "/subscriptions/${SUBSCRIPTION_ID}"
  if [ $? -ne 0 ]; then
    warn "failed to delete actlabs 'User Access Administrator' role from the subscription. Please delete manually or contact support."
  else
    log "actlabs 'User Access Administrator' role deleted from the subscription"
  fi
}

# Function to delete 'Storage Blob Data Contributor' role from the resource group repro-project
function delete_storage_blob_data_contributor_role() {
  az role assignment delete --assignee "${ACTLABS_SP_APP_ID}" --role "Storage Blob Data Contributor" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}"
  if [ $? -ne 0 ]; then
    warn "failed to delete actlabs 'Storage Blob Data Contributor' role from the resource group. Please delete manually or contact support."
  else
    log "actlabs 'Storage Blob Data Contributor' role deleted from the resource group"
  fi
}

# Function to delete actlabs msi Contributor role from resource group
delete_actlabs_msi_contributor_role() {
  az role assignment delete --assignee "${ACTLABS_MSI_APP_ID}" --role Contributor --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}"
  if [ $? -ne 0 ]; then
    warn "failed to delete actlabs msi Contributor role from resource group. Please delete manually or contact support."
  else
    log "actlabs msi Contributor role deleted from resource group"
  fi
}

# Function to delete actlabs msi Reader role from subscription
delete_actlabs_msi_reader_role() {
  az role assignment delete --assignee "${ACTLABS_MSI_APP_ID}" --role Reader --scope "/subscriptions/${SUBSCRIPTION_ID}"
  if [ $? -ne 0 ]; then
    warn "failed to delete actlabs msi Reader role from subscription. Please delete manually or contact support."
  else
    log "actlabs msi Reader role deleted from subscription"
  fi
}

# Call the functions
get_upn
get_subscription_id
ensure_user_is_owner
delete_actlabs_msi_reader_role
delete_actlabs_msi_contributor_role
delete_storage_blob_data_contributor_role
delete_user_access_administrator_role
delete_contributor_role
delete_resource_group
ok "Unregistration complete. Thank you for using the service."
