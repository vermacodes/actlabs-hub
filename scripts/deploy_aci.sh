#!/bin/bash

# gather input parameters
# -t "tag" for image tag
# -d for debug

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

source ./deploy/.prod.env

echo "TAG = ${TAG}"
echo "DEBUG = ${DEBUG}"

if [ -z "${TAG}" ]; then
    TAG="latest"
fi

if [ ${DEBUG} ]; then
    export ACTLABS_HUB_LOG_LEVEL="-4"
else
    export ACTLABS_HUB_LOG_LEVEL="0"
fi

required_env_vars=("AUTH_TOKEN_ISS" "AUTH_TOKEN_AUD" "ACTLABS_HUB_CLIENT_ID" "TENANT_ID")

for var in "${required_env_vars[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo "Required environment variable $var is missing"
    exit 1
  fi
done

# get storage account key
export STORAGE_ACCOUNT_KEY=$(az storage account keys list \
  --account-name ${ACTLABS_HUB_STORAGE_ACCOUNT} \
  --resource-group ${ACTLABS_HUB_RESOURCE_GROUP} \
  --subscription ${ACTLABS_HUB_SUBSCRIPTION_ID} \
  --query "[0].value" \
  --output tsv)

# generate deploy.yaml from deploy.template.yaml
envsubst <./deploy/deploy.template.yaml >./deploy/.deploy.yaml

# create container group
az container create \
  --resource-group actlabs-app \
  --subscription ACT-CSS-Readiness \
  --file ./deploy/.deploy.yaml

# remove deploy.yaml
rm ./deploy/.deploy.yaml