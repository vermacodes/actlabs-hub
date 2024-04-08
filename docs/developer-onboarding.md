# Onboarding for new developers

## Prerequisites

Environment setup is out of scope for this document, but the individuals and teams named in the [OWNERS](../OWNERS) file are available to help with any questions you might have. Before diving in to work on the applications and automations in this repository, there are a few things you'll need to have set up on your machine.

* Go 1.22 or newet
* Docker (or equivalent runtime)
* Git
* Redis (or the ability to connect to a Redis instance)

The following components can make the development processe easier, but are not strictly required:

* Local copy of the [actlabs-ui](https://github.com/vermacodes/one-click-aks-ui) repository with a working configuration.
* Local copy of the [actlabs-hub](https://github.com/vermacodes/one-click-aks-hub) repository with a working hub setup

We also strongly encourage Windows users to take advantage of the Windows Subsystem for Linux (WSL) to run the applications and automations in this repository. The applications and automations are developed and tested on Linux and managing the required tooling and dependencies for this project is much easier on a Linux environment.

## Development Workflow

When developing for this repository, the recommended path is working off of a locally-cloned copy of the repository. If you're confident in your abilities to write code without the assistance of an IDE or extensible editor such as VSCode, you're welcome to make changes from the browser. That said, the following workflow is recommended for the best development experience:

1. Fork this repository into your own GitHub account.
2. Clone the repository to your local machine (Git CLI via WSL).
3. Create a new branch for your work, naming appropriately for the work you're doing and the work items you're focusing on.
4. Iterate through code changes, committing and pushing at regular intervals to sync your work onto your remote branch.
5. When you're ready to submit your work, open a pull request in the Azure DevOps repository and assign it to the appropriate team members for review.
6. Iterate through the review process, making changes as necessary until the pull request is approved and merged.
7. Delete your remote branch and pull the latest changes from the `main` branch.

### Running actlabs-server

The actlabs-server application is a Go application with some required dependencies.

#### Setting up Redis

Prior to starting the actlabs-server application, Redis needs to be running. A local install of Redis is acceptable, however it can also be run as a Docker container. The following command will start a Redis container on the default port:

```bash
docker run -d -p 6379:6379 redis:latest
```

> Note: [Microsoft Garnet](https://github.com/Microsoft/garnet) is a high-performance key-value store that can be used as a drop-in replacement for Redis. It can be used in place of Redis for local development if desired.

#### Configuring the runtime environment

To run, the Go application needs a `.env` file _or_ a set of environment variables. The `.env` file is the easiest approach to get running and can be generated using the following script in the root of your actlabs-server repository:

> Note: this assumes a valid Azure CLI session is in place. If you haven't yet, please run `az login` and set your account to use the desired subscription using `az account set -s <subscription-id>`

```bash
#!/usr/bin/env bash
touch .env

echo "PORT=80" > .env
echo "ROOT_DIR=." >> .env
echo "USE_MSI=false" >> .env
echo "USE_SERVICE_PRINCIPAL=false" >> .env
echo "ARM_SUBSCRIPTION_ID=$(az account show --query id -o tsv)" >> .env
echo "ARM_TENANT_ID=$(az account show --query tenantId -o tsv)" >> .env
echo "ARM_USER_PRINCIPAL_NAME=$(az account show --query 'user.name' -o tsv)" >> .env
echo "AZURE_CLIENT_ID=not-used-for-self-hosting" >> .env
echo "AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)" >> .env
echo "AUTH_TOKEN_ISS=https://login.microsoftonline.com/72f988bf-86f1-41af-91ab-2d7cd011db47/v2.0" >> .env
echo "AUTH_TOKEN_AUD=00399ddd-434c-4b8a-84be-d096cff4f494" >> .env
echo "ACTLABS_HUB_URL=https://actlabs-hub-capp.redisland-ff4b63ab.azurecontainerapps.io" >> .env
echo "LOG_LEVEL=0" >> .env
```

#### Running the actlabs-server

Now that Redis is running and our .env file is present in the root of our repository, you can run it using the following command: `go run cmd/one-click-aks-server/main.go`.

### Finding and claiming work items

Work items are managed through a combination of GitHub issues and Azure Dev Ops.

For Azure Dev Ops work items, appropriate tagging and  with appropriate tagging and an Area Path are set. Work items are triaged on a regular basis (following a similar triage flow to the one used by the wiki) and are lumped into monthly buckets that align to the PG semester cadence. Uncommitted work or untriaged work items should slot in to a lower priority than the ones committed to the current month.
