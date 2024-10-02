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
