#!/bin/bash

source .env

export ROOT_DIR=$(pwd)
export LOG_LEVEL="-4"

required_env_vars=("AUTH_TOKEN_ISS" "AUTH_TOKEN_AUD")

for var in "${required_env_vars[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo "Required environment variable $var is missing"
    exit 1
  fi
done

rm ./tmp/main

go build -o ./tmp/main ./cmd/actlabs-hub

redis-cli flushall

export LOG_LEVEL="${LOG_LEVEL}" && export PORT="8883"
