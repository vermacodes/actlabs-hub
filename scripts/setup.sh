#!/bin/bash

ACTLABS_SP_APP_ID="00399ddd-434c-4b8a-84be-d096cff4f494"
ACTLABS_FDPO_SP_APP_ID="50cc6d33-3224-477f-b2bd-5c1c6595fdf5"
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

		# # Ask the user for a location if one wasn't provided
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
	# Get a list of all storage accounts in the resource group
	SA_LIST=$(az storage account list --resource-group "${RESOURCE_GROUP}" --query "[].name" -o tsv)

	for SA in $SA_LIST; do
		if [[ "${SA}" == "${USER_ALIAS_FOR_SA}"* ]]; then
			log "storage account already exists with name ${SA}"
			STORAGE_ACCOUNT_NAME="$SA"
			break
		fi
	done

	if [[ -z "${STORAGE_ACCOUNT_NAME}" ]]; then
		# Generate a random name for the storage account
		RANDOM_NAME=$(openssl rand -hex 2)
		STORAGE_ACCOUNT_NAME="${USER_ALIAS_FOR_SA}sa${RANDOM_NAME}"

		log "creating storage account with name ${STORAGE_ACCOUNT_NAME} in resource group ${RESOURCE_GROUP}"

		# Register the storage provider
		az provider register --namespace Microsoft.Storage --consent-to-permission --wait
		# Create the storage account
		az storage account create --name "${STORAGE_ACCOUNT_NAME}" --resource-group "${RESOURCE_GROUP}" --sku Standard_LRS
		if [ $? -ne 0 ]; then
			err "failed to create storage account ${STORAGE_ACCOUNT_NAME}"
			exit 1
		else
			log "storage account ${STORAGE_ACCOUNT_NAME} created"
		fi
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

# Function to check if managed identity is 'Contributor' on the subscription
# If not, assign the managed identity the 'Contributor' role on the subscription
function assign_contributor_role() {
	# Check if the managed identity is 'Contributor' on the subscription
	MI_ROLE=$(az role assignment list --assignee "${ACTLABS_SP_APP_ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='Contributor'].roleDefinitionName" -o tsv)

	if [[ -n "${MI_ROLE}" ]]; then
		log "actlabs is already 'Contributor' on the subscription"
	else
		log "assigning actlabs 'Contributor' role on the subscription"
		# Assign the managed identity the 'Contributor' role on the subscription
		az role assignment create --assignee "${ACTLABS_SP_APP_ID}" --role Contributor --scope "/subscriptions/${SUBSCRIPTION_ID}"
		if [ $? -ne 0 ]; then
			err "failed to assign actlabs 'Contributor' role on the subscription"
			exit 1
		else
			log "actlabs assigned 'Contributor' role on the subscription"
		fi
	fi

	return 0
}

# Function to check if managed identity is 'User Access Administrator' on the subscription
# If not, assign the managed identity the 'User Access Administrator' role on the subscription
function assign_user_access_administrator_role() {
	# Check if the managed identity is 'User Access Administrator' on the subscription
	MI_ROLE=$(az role assignment list --assignee "${ACTLABS_SP_APP_ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='User Access Administrator'].roleDefinitionName" -o tsv)

	if [[ -n "${MI_ROLE}" ]]; then
		log "actlabs is already 'User Access Administrator' on the subscription"
	else
		log "assigning actlabs 'User Access Administrator' role on the subscription"
		# Assign the managed identity the 'User Access Administrator' role on the subscription
		az role assignment create --assignee "${ACTLABS_SP_APP_ID}" --role "User Access Administrator" --scope "/subscriptions/${SUBSCRIPTION_ID}"
		if [ $? -ne 0 ]; then
			err "failed to assign actlabs 'User Access Administrator' role on the subscription"
			exit 1
		else
			log "actlabs assigned 'User Access Administrator' role on the subscription"
		fi
	fi

	return 0
}

# Function to check if managed identity is 'Storage Blob Data Contributor' on the resource group repro-project
# If not, assign the managed identity the 'Storage Blob Data Contributor' role on the resource group repro-project
function assign_storage_blob_data_contributor_role() {
	# Check if the managed identity is 'Storage Blob Data Contributor' on the resource group repro-project
	MI_ROLE=$(az role assignment list --assignee "${ACTLABS_SP_APP_ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" --query "[?roleDefinitionName=='Storage Blob Data Contributor'].roleDefinitionName" -o tsv)

	if [[ -n "${MI_ROLE}" ]]; then
		log "actlabs is already 'Storage Blob Data Contributor' on the resource group"
	else
		log "assigning actlabs 'Storage Blob Data Contributor' role on the resource group"
		# Assign the managed identity the 'Storage Blob Data Contributor' role on the resource group
		az role assignment create --assignee "${ACTLABS_SP_APP_ID}" --role "Storage Blob Data Contributor" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}"
		if [ $? -ne 0 ]; then
			err "failed to assign actlabs 'Storage Blob Data Contributor' role on the resource group"
			exit 1
		else
			log "actlabs assigned 'Storage Blob Data Contributor' role on the resource group"
		fi
	fi

	return 0
}

# Function to assign actlabs msi as a contributor on the resource group
function assign_actlabs_msi_contributor_role() {
	# Check if actlabs msi is a contributor on the resource group
	ACTLABS_ROLE=$(az role assignment list --assignee "${ACTLABS_MSI_APP_ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}" --query "[?roleDefinitionName=='Contributor'].roleDefinitionName" -o tsv)

	if [[ -n "${ACTLABS_ROLE}" ]]; then
		log "actlabs msi is already 'Contributor' on the resource group"
	else
		log "assigning actlabs msi 'Contributor' role on the resource group"
		# Assign actlabs msi the 'Contributor' role on the resource group
		az role assignment create --assignee "${ACTLABS_MSI_APP_ID}" --role Contributor --scope "/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}"
		if [ $? -ne 0 ]; then
			err "failed to assign actlabs msi 'Contributor' role on the resource group"
			exit 1
		else
			log "actlabs msi assigned 'Contributor' role on the resource group"
		fi
	fi

	return 0
}

# Function to assign actlabs msi as a reader on the subscription
function assign_actlabs_msi_reader_role() {
	# Check if actlabs msi is a reader on the subscription
	ACTLABS_ROLE=$(az role assignment list --assignee "${ACTLABS_MSI_APP_ID}" --scope "/subscriptions/${SUBSCRIPTION_ID}" --query "[?roleDefinitionName=='Reader'].roleDefinitionName" -o tsv)

	if [[ -n "${ACTLABS_ROLE}" ]]; then
		log "actlabs msi is already 'Reader' on the subscription"
	else
		log "assigning actlabs msi 'Reader' role on the subscription"
		# Assign actlabs msi the 'Reader' role on the subscription
		az role assignment create --assignee "${ACTLABS_MSI_APP_ID}" --role Reader --scope "/subscriptions/${SUBSCRIPTION_ID}"
		if [ $? -ne 0 ]; then
			err "failed to assign actlabs msi 'Reader' role on the subscription"
			exit 1
		else
			log "actlabs msi assigned 'Reader' role on the subscription"
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
		https://actlabs-hub-capp.redisland-ff4b63ab.eastus.azurecontainerapps.io/arm/server/register \
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
		ok "Next steps: Go back to UI and register your subscription with the lab"
		ok "Your subscription id is: ${SUBSCRIPTION_ID}"
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
fi
create_resource_group
create_storage_account
if [[ "${is_owner}" == true ]]; then
	assign_contributor_role
	assign_user_access_administrator_role
	assign_storage_blob_data_contributor_role
	if [[ "${is_fdpo}" == false ]]; then
		assign_actlabs_msi_contributor_role
		assign_actlabs_msi_reader_role
	fi
fi
register_subscription
