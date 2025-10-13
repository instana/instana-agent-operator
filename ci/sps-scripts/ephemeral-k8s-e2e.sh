#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail

echo "===== ephemeral-k8s-e2e.sh - start ====="

# Generate unique VM name based on PR/branch
# Sanitize branch name: replace invalid characters with hyphens
SANITIZED_BRANCH_NAME=$(echo "${BRANCH_NAME}" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
VM_NAME="pr-test-${SANITIZED_BRANCH_NAME}-${BUILD_NUMBER}"

# Ensure VM name is not too long (GCP limit is 63 chars)
if [ ${#VM_NAME} -gt 63 ]; then
  # Truncate to 54 chars and append a short hash for uniqueness
  SHORT_HASH=$(echo "${BRANCH_NAME}-${BUILD_NUMBER}" | md5sum | cut -c1-8)
  VM_NAME="pr-test-${SANITIZED_BRANCH_NAME:0:46}-${SHORT_HASH}"
fi
ZONE="us-central1-a"
PROJECT_ID="instana-agent-qa"

# Cleanup function to ensure VM is always deleted
cleanup() {
  echo "Cleaning up resources..."
  
  # Set commit status based on test results
  set-commit-status \
    --repository "$(load_repo app-repo url)" \
    --commit-sha "$(load_repo app-repo commit)" \
    --state "${COMMIT_STATUS:-failure}" \
    --description "Kubernetes e2e test" \
    --context "tekton/e2e-ephemeral-k8s" \
    --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
  
  if [[ "$(get_env pipeline_namespace 2>/dev/null)" == *"pr"* ]]; then
    set-commit-status \
      --repository "$(load_repo app-repo url)" \
      --commit-sha "$(load_repo app-repo commit)" \
      --state "${COMMIT_STATUS:-failure}" \
      --description "Kubernetes e2e test" \
      --context "tekton/e2e-ephemeral-k8s" \
      --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
  else
    set-commit-status \
      --repository "$(load_repo app-repo url)" \
      --commit-sha "$(load_repo app-repo commit)" \
      --state "${COMMIT_STATUS:-failure}" \
      --description "Kubernetes e2e test" \
      --context "tekton/e2e-ephemeral-k8s" \
      --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/deploy\?}"
  fi
  
  # Delete the VM
  gcloud compute instances delete ${VM_NAME} --zone=${ZONE} --project=${PROJECT_ID} --quiet || true
  
  echo "===== ephemeral-k8s-e2e.sh - end ====="
  
  # Exit with the same code as the e2e tests
  if [[ ${E2E_EXIT_CODE:-0} -ne 0 ]]; then
    echo "Exiting with failure due to e2e test failure"
    exit ${E2E_EXIT_CODE}
  fi
}

# Ensure cleanup runs even if script fails
trap cleanup EXIT SIGINT SIGTERM

# Authenticate to GCP
echo "Authenticating to GCP..."
echo -n "$(get_env GOOGLE_APPLICATION_CREDENTIALS_BASE64)" | base64 -d > gcp-key.json
gcloud auth activate-service-account --key-file gcp-key.json
gcloud config set project ${PROJECT_ID}

# Check if persistent firewall rule exists, if not create it
FIREWALL_RULE_NAME="e2e-k3s-api"
if ! gcloud compute firewall-rules describe ${FIREWALL_RULE_NAME} --project=${PROJECT_ID} &>/dev/null; then
  echo "Creating persistent firewall rule for k3s API server..."
  gcloud compute firewall-rules create ${FIREWALL_RULE_NAME} \
    --project=${PROJECT_ID} \
    --direction=INGRESS \
    --priority=1000 \
    --network=default \
    --action=ALLOW \
    --rules=tcp:6443 \
    --source-ranges=0.0.0.0/0 \
    --target-tags=k3s-server
else
  echo "Using existing persistent firewall rule: ${FIREWALL_RULE_NAME}"
fi

# Create VM with k3s
echo "Creating VM ${VM_NAME}..."
gcloud compute instances create ${VM_NAME} \
  --project=${PROJECT_ID} \
  --zone=${ZONE} \
  --machine-type=e2-standard-4 \
  --image=projects/ubuntu-os-cloud/global/images/ubuntu-minimal-2404-noble-amd64-v20251002 \
  --labels="purpose=e2e-testing,creation-time=$(date +%s),max-lifetime=90m" \
  --metadata="startup-script=#!/bin/bash
    # Install k3s with tls-san to include the external IP in the certificate
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC=\"--disable=traefik --tls-san=\$(curl -s http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip -H 'Metadata-Flavor: Google')\" sh -
  " \
  --tags=k3s-server

# Wait for VM to be ready
echo "Waiting for VM to be ready..."
until gcloud compute ssh ${VM_NAME} --zone=${ZONE} --project=${PROJECT_ID} --command="echo VM is ready" --quiet; do
  echo "VM not ready yet, waiting..."
  sleep 10
done

# Wait for k3s to be ready
echo "Waiting for k3s to be ready..."
until gcloud compute ssh ${VM_NAME} --zone=${ZONE} --project=${PROJECT_ID} --command="sudo kubectl --kubeconfig /etc/rancher/k3s/k3s.yaml get nodes" --quiet; do
  echo "k3s not ready yet, waiting..."
  sleep 5
done

# Get kubeconfig from VM
echo "Retrieving kubeconfig..."
mkdir -p ~/.kube
gcloud compute ssh ${VM_NAME} --zone=${ZONE} --project=${PROJECT_ID} \
  --command="sudo cat /etc/rancher/k3s/k3s.yaml" > ~/.kube/config

# Update server address in kubeconfig to use the VM's external IP
VM_IP=$(gcloud compute instances describe ${VM_NAME} --zone=${ZONE} --project=${PROJECT_ID} --format='get(networkInterfaces[0].accessConfigs[0].natIP)')
echo "VM external IP: ${VM_IP}"
sed -i "s|server: https://127.0.0.1:6443|server: https://${VM_IP}:6443|g" ~/.kube/config

# Verify connection
echo "Verifying connection to k3s cluster..."
kubectl get nodes

# Setup container registry credentials
echo "Setting up container registry credentials..."
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
ARTIFACTORY_USERNAME=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".username")
ARTIFACTORY_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")

# Setup Instana backend details
echo "Setting up Instana backend details..."
INSTANA_E2E_BACKEND_DETAILS=$(get_env instana-e2e-backend-details)
INSTANA_ENDPOINT_HOST=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".endpoint_host")
INSTANA_ENDPOINT_PORT=443
INSTANA_API_KEY=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".agent_key")
INSTANA_API_URL=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_url")
INSTANA_API_TOKEN=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_token")

export INSTANA_ENDPOINT_HOST INSTANA_ENDPOINT_PORT INSTANA_API_KEY INSTANA_API_URL INSTANA_API_TOKEN
export ARTIFACTORY_USERNAME ARTIFACTORY_PASSWORD

# Get the Git commit for image tagging
GIT_COMMIT="$(load_repo app-repo commit)"
export GIT_COMMIT

# Initialize commit status variables
COMMIT_STATUS="failure"
E2E_EXIT_CODE=0

# Run e2e tests
echo "=== Running e2e tests ==="
cd "${WORKSPACE}/${APP_REPO_FOLDER}"

if ! make e2e; then
  echo "E2E tests failed"
  E2E_EXIT_CODE=1
else
  COMMIT_STATUS="success"
  echo "Tests completed successfully!"
fi

# Made with Bob
