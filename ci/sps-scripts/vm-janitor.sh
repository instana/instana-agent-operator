#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail

echo "===== vm-janitor.sh - start ====="

PROJECT_ID="instana-agent-qa"
MAX_AGE_SECONDS=$((90 * 60)) # 90 minutes in seconds

# Authenticate to GCP
echo "Authenticating to GCP..."
echo -n "$(get_env GOOGLE_APPLICATION_CREDENTIALS_BASE64)" | base64 -d > gcp-key.json
gcloud auth activate-service-account --key-file gcp-key.json
gcloud config set project ${PROJECT_ID}

echo "Running VM janitor cleanup..."

# Get current timestamp
CURRENT_TIME=$(date +%s)

# List VMs with e2e-testing purpose label
VM_LIST=$(gcloud compute instances list \
  --project=${PROJECT_ID} \
  --filter="labels.purpose=e2e-testing" \
  --format="csv[no-heading](name,zone,labels.creation-time)")

# Check each VM
while IFS=, read -r VM_NAME VM_ZONE CREATION_TIME; do
  if [[ -z "${VM_NAME}" ]]; then
    continue
  fi
  
  # Calculate age
  VM_AGE=$((CURRENT_TIME - CREATION_TIME))
  
  # If older than MAX_AGE_SECONDS, delete it
  if [[ ${VM_AGE} -gt ${MAX_AGE_SECONDS} ]]; then
    echo "VM ${VM_NAME} in zone ${VM_ZONE} is ${VM_AGE} seconds old, exceeding limit of ${MAX_AGE_SECONDS}. Deleting..."
    gcloud compute instances delete ${VM_NAME} \
      --project=${PROJECT_ID} \
      --zone=${VM_ZONE} \
      --quiet
  else
    echo "VM ${VM_NAME} in zone ${VM_ZONE} is ${VM_AGE} seconds old, within limit of ${MAX_AGE_SECONDS}. Keeping."
  fi
done <<< "${VM_LIST}"

echo "VM janitor cleanup completed."
echo "===== vm-janitor.sh - end ====="

# Made with Bob
