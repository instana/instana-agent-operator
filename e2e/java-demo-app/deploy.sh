#!/bin/bash
# (c) Copyright IBM Corp. 2025

if [ -z "${NAMESPACE}" ]; then
    echo "Error: NAMESPACE variable is not set"
    echo "Please set the NAMESPACE variable before running this script"
    echo "Example: NAMESPACE=your-namespace ./deploy.sh"
    exit 1
fi

kubectl create ns "${NAMESPACE}"

mkdir -p .tmp
jq '{auths: {"delivery.instana.io": .auths["delivery.instana.io"]}}' ~/.docker/config.json > .tmp/filtered-docker-config.json
echo "Checking if secret delivery-instana-io-pull-secret exists in namespace ${NAMESPACE}..."
if kubectl get secret delivery-instana-io-pull-secret -n ${NAMESPACE} >/dev/null 2>&1; then
    echo "Updating existing secret delivery-instana-io-pull-secret..."
    kubectl delete secret delivery-instana-io-pull-secret -n ${NAMESPACE}
else
    echo "Creating secret delivery-instana-io-pull-secret..."
fi

kubectl create secret generic delivery-instana-io-pull-secret \
    --from-file=.dockerconfigjson=.tmp/filtered-docker-config.json \
    --type=kubernetes.io/dockerconfigjson \
    -n ${NAMESPACE}

rm -rf .tmp

kubectl apply -f deployment.yaml

echo ""
echo "To check if pods are deployed correctly, run:"
echo "kubectl get pods -l app=java-demo-app -n ${NAMESPACE}"