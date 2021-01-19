#!/bin/bash
#
# (c) Copyright IBM Corp. 2021
# (c) Copyright Instana Inc.
#


set -e

VERSION=${1:-dev}
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
BASE_DIR="$SCRIPT_DIR/../../"
OS_NAME=$(uname | tr '[:upper:]' '[:lower:]')
KIND_CONFIG_FILE="$SCRIPT_DIR/kind-config-$OS_NAME.yaml"

printf "%s\n" "Creating cluster with kind-config-$OS_NAME.yaml"
kind --config $KIND_CONFIG_FILE create cluster

kubectl get nodes

printf "%s\n" "Build and load Operator image into kind cluster"
./mvnw -C -B clean package
docker build -f $BASE_DIR/src/main/docker/Dockerfile.jvm -t instana/instana-agent-operator:$VERSION $BASE_DIR
kind load docker-image instana/instana-agent-operator

printf "%s\n" "Load Agent image into kind cluster"
docker pull instana/agent
kind load docker-image instana/agent
