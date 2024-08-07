#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -x
set -e
set -o pipefail


case "${CLUSTER_TYPE}" in
    gke)
        echo 'Running on a GKE cluster'

        echo "${GCP_KEY_JSON}" > keyfile.json
        echo "${CLUSTER_INFO}" > cluster.info

        echo 'Authenticating with gcloud'

        gcloud auth activate-service-account --key-file keyfile.json

        GKE_PROJECT=$(jq -r .project < cluster.info)
        GKE_CLUSTER_NAME=$(jq -r .name < cluster.info)
        GKE_ZONE=$(jq -r .zone < cluster.info)
        readonly GKE_PROJECT GKE_CLUSTER_NAME GKE_ZONE

        # To have kubectl use the new binary plugin for authentication instead of using the default provider-specific code,
        # set the environment variable USE_GKE_GCLOUD_AUTH_PLUGIN=True (kubectl below 1.26)
        # The test image contains google-cloud-sdk-gke-gcloud-auth-plugin
        export USE_GKE_GCLOUD_AUTH_PLUGIN=True
        gcloud container clusters get-credentials "${GKE_CLUSTER_NAME}" --zone "${GKE_ZONE}" --project "${GKE_PROJECT}"

        ;;
    openshift)
        echo "${KUBECONFIG_SOURCE}" > kubeconfig
        chmod 0400 kubeconfig
        KUBECONFIG="$(pwd)/kubeconfig"
        export KUBECONFIG

        ;;
    *)
        echo "Undefined cluster"
        ;;
esac

cd pipeline-source
echo "Uninstalling previous 'instana-agent' release"
make undeploy &> /dev/null || true
make uninstall &> /dev/null || true


echo "Deleting instana-agent namespace"
if kubectl get namespace/instana-agent ; then
    # if the agent CR has a finalizer but no operator is running, it will block the namespace from terminating correctly
    kubectl patch agent -n instana-agent instana-agent -p '{"metadata":{"finalizers":null}}' --type=merge || true
    kubectl delete namespace/instana-agent
    kubectl wait --for=delete namespace/instana-agent --timeout=30s
    echo "Deleted namespace instana-agent"
else
    echo "Namespace instana-agent does not exist; skipping delete"
fi

echo "Done"