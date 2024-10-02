#!/bin/bash

# gather input parameters
# -t tag

while getopts ":t:" opt; do
  case $opt in
  t)
    TAG="$OPTARG"
    ;;
  \?)
    echo "Invalid option -$OPTARG" >&2
    ;;
  esac
done

source .env

if [ -z "${TAG}" ]; then
  TAG="latest"
fi

echo "TAG = ${TAG}"

required_env_vars=("AUTH_TOKEN_ISS" "AUTH_TOKEN_AUD")

for var in "${required_env_vars[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo "Required environment variable $var is missing"
    exit 1
  fi
done

go build -o actlabs-hub ./cmd/actlabs-hub
if [ $? -ne 0 ]; then
  echo "Failed to build actlabs-hub"
  exit 1
fi

docker build -t actlabs.azurecr.io/actlabs-hub:${TAG} .
if [ $? -ne 0 ]; then
  echo "Failed to build docker image"
  exit 1
fi

rm actlabs-hub

az acr login --name actlabs --subscription ACT-CSS-Readiness-NPRD
docker push actlabs.azurecr.io/actlabs-hub:${TAG}

docker tag actlabs.azurecr.io/actlabs-hub:${TAG} ashishvermapu/actlabs-hub:${TAG}
docker push ashishvermapu/actlabs-hub:${TAG}
