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

mkdir -p bin

curl -LO \
  https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 \
  && install minikube-linux-amd64 bin/
  
kv=$(curl -sSL https://dl.k8s.io/release/stable.txt)
curl -LO \
  https://dl.k8s.io/$kv/bin/linux/amd64/kubectl \
  && install kubectl bin/

PATH=$PATH:$(pwd)/bin
echo "DOCKER_HOST=$DOCKER_HOST"
export DOCKER_HOST=tcp://localhost:2376
echo "setting DOCKER_HOST=tcp://localhost:2376 explicitly"

minikube-linux-amd64 config set WantUpdateNotification false
minikube-linux-amd64 start --driver=docker --force

kubectl get nodes -o wide
kubectl get pods
kubectl get ns

echo "===== minikube.sh - stop ====="