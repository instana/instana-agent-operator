#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail
# note: PIPELINE_CONFIG_REPO_PATH will point to config, not to the app folder with the current branch, use APP_REPO_FOLDER instead
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
    trap env EXIT
    env
    set -x
fi
echo "===== minikube.sh - start ====="

# do not fail, just try things
set +e
mkdir -p bin

kv=$(curl -sSL https://dl.k8s.io/release/stable.txt)
curl -LO \
  https://dl.k8s.io/$kv/bin/linux/amd64/kubectl \
  && install kubectl bin/

PATH=$PATH:$(pwd)/bin
export DOCKER_HOST=tcp://localhost:2376
export DOCKER_TLS_VERIFY=1
export DOCKER_CERT_PATH=/certs/client
echo "setting DOCKER_HOST=tcp://localhost:2375 explicitly"

curl -LO \
  https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
  && install minikube-linux-amd64 bin/
minikube-linux-amd64 config set WantUpdateNotification false
minikube-linux-amd64 start \
  --driver=docker \
  --force \
  --container-runtime=docker \
  --extra-config=kubelet.cgroup-driver=cgroupfs \
  --wait=all \
  --wait-timeout=10m

# Install kind
# curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.30.0/kind-linux-amd64
# chmod +x ./kind
# mv ./kind bin/kind

# # Create cluster
# kind create cluster --wait 5m

kubectl get nodes -o wide
kubectl get pods
kubectl get ns

docker ps

echo "===== minikube.sh - stop ====="