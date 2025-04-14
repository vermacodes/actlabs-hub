#!/bin/bash

# This script creates the Azure Container Apps environments for ACT Labs servers.
# Use this script to create the environments in the specified resource group and subscription.

az containerapp env create --name actlabs-servers-env-01 \
  --resource-group actlabs-servers \
  --subscription ACT-CSS-Readiness-NPRD \
  --location eastus \
  --logs-destination none

az containerapp env create --name actlabs-servers-env-02 \
  --resource-group actlabs-servers \
  --subscription ACT-CSS-Readiness-NPRD \
  --location eastus \
  --logs-destination none

az containerapp env create --name actlabs-servers-env-03 \
  --resource-group actlabs-servers \
  --subscription ACT-CSS-Readiness-NPRD \
  --location eastus \
  --logs-destination none
