#!/bin/bash

# This script is used by air to run the actlabs-hub application locally.
# Don't run the script directly, use air instead.
# refer .air.toml for more details.
rm ./tmp/main

# Source .env files from root directory
if [ -f .env ]; then
  source .env
  echo "Sourced .env"
else
  echo "Warning: .env file not found"
fi

if [ -f .env.local ]; then
  source .env.local
  echo "Sourced .env.local"
else
  echo "Warning: .env.local file not found"
fi

echo "Sourcing Variables Completed"

export ROOT_DIR=$(pwd)
export LOG_LEVEL="-4"

required_env_vars=("AUTH_TOKEN_ISS" "AUTH_TOKEN_AUD" "PROTECTED_LAB_SECRET")

for var in "${required_env_vars[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo "Required environment variable $var is missing"
    exit 1
  fi
done


go build -o ./tmp/main ./cmd/actlabs-hub

redis-cli flushall
