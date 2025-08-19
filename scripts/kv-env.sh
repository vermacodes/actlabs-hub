#!/bin/bash

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

usage() {
  gap
  echo "Usage: $0 --vault-name <vault> --name <secret> --file <file> <upload|download>"
  echo "  -v, --vault-name   Azure Key Vault name"
  echo "  -n, --name         Secret name in Key Vault"
  echo "  -f, --file         .env file path"
  echo "  upload|download    Operation to perform"
  gap
  exit 1
}

check_az_cli() {
  if ! command -v az &>/dev/null; then
    err "Azure CLI (az) is not installed. Please install it first."
    exit 1
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -v|--vault-name)
        VAULT_NAME="$2"
        shift 2
        ;;
      -n|--name)
        SECRET_NAME="$2"
        shift 2
        ;;
      -f|--file)
        FILE="$2"
        shift 2
        ;;
      upload|download)
        OPERATION="$1"
        shift
        ;;
      -*|*)
        err "Unknown option or argument: $1"
        usage
        ;;
    esac
  done

  if [[ -z "$VAULT_NAME" || -z "$SECRET_NAME" || -z "$FILE" || -z "$OPERATION" ]]; then
    err "Missing required arguments."
    usage
  fi

  if [[ "$OPERATION" != "upload" && "$OPERATION" != "download" ]]; then
    err "Operation must be 'upload' or 'download'."
    usage
  fi
}

upload_secret() {
  if [[ ! -f "$FILE" ]]; then
    err "File '$FILE' does not exist."
    exit 1
  fi

  log "Encoding $FILE to base64..."
  BASE64_CONTENT=$(base64 -w 0 "$FILE" 2>/dev/null)
  if [[ $? -ne 0 || -z "$BASE64_CONTENT" ]]; then
    err "Failed to base64 encode $FILE."
    exit 1
  fi

  log "Uploading secret '$SECRET_NAME' to Key Vault '$VAULT_NAME'..."
  az keyvault secret set --vault-name "$VAULT_NAME" --name "$SECRET_NAME" --value "$BASE64_CONTENT" >/dev/null 2>&1
  if [[ $? -ne 0 ]]; then
    err "Failed to upload secret to Azure Key Vault."
    exit 1
  fi

  ok "Secret '$SECRET_NAME' uploaded to Key Vault '$VAULT_NAME'."
}

download_secret() {
  log "Retrieving secret '$SECRET_NAME' from Key Vault '$VAULT_NAME'..."
  BASE64_CONTENT=$(az keyvault secret show --vault-name "$VAULT_NAME" --name "$SECRET_NAME" --query value -o tsv 2>/dev/null)
  if [[ $? -ne 0 || -z "$BASE64_CONTENT" ]]; then
    err "Failed to retrieve secret from Azure Key Vault."
    exit 1
  fi

  log "Decoding secret and writing to $FILE..."
  echo "$BASE64_CONTENT" | base64 -d > "$FILE"
  if [[ $? -ne 0 ]]; then
    err "Failed to decode and write to $FILE."
    exit 1
  fi

  ok "Secret '$SECRET_NAME' downloaded and written to $FILE."
}

main() {
  check_az_cli
  parse_args "$@"

  if [[ "$OPERATION" == "upload" ]]; then
    upload_secret
  else
    download_secret
  fi
}

main "$@"
