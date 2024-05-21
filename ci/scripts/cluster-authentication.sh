#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

function cleanup_namespace() {
    echo "Deleting the namespaces"
    for ns in instana-agent test-apps ; do
    if kubectl get namespace/$ns ; then
        kubectl delete namespace/$ns
        kubectl wait --for=delete namespace/$ns --timeout=30s
        echo "Deleted namespace $ns"
    else
        echo "Namespace $ns does not exist; skipping delete"
    fi
    done
    echo "OK"
}

case "${CLUSTER_TYPE}" in
    gke)
        echo 'Testing on a GKE cluster'

        echo "${GCP_KEY_JSON}" > keyfile.json
        echo "${CLUSTER_INFO}" > cluster.info

        echo 'Authenticating with gcloud'

        gcloud auth activate-service-account --key-file keyfile.json

        readonly GKE_PROJECT=$(jq -r .project < cluster.info)
        readonly GKE_CLUSTER_NAME=$(jq -r .name < cluster.info)
        readonly GKE_ZONE=$(jq -r .zone < cluster.info)

        # To have kubectl use the new binary plugin for authentication instead of using the default provider-specific code,
        # set the environment variable USE_GKE_GCLOUD_AUTH_PLUGIN=True (kubectl below 1.26)
        # The test image contains google-cloud-sdk-gke-gcloud-auth-plugin
        export USE_GKE_GCLOUD_AUTH_PLUGIN=True
        gcloud container clusters get-credentials "${GKE_CLUSTER_NAME}" --zone "${GKE_ZONE}" --project "${GKE_PROJECT}"

        CLUSTER_NAME="${GKE_CLUSTER_NAME}"
        cleanup_namespace
        ;;
    openshift)
        echo "${KUBECONFIG_SOURCE}" > kubeconfig
        chmod 0400 kubeconfig
        KUBECONFIG="$(pwd)/kubeconfig"
        export KUBECONFIG

        docker_email=$(echo "${GCP_KEY_JSON//$'\n'/}" | jq -r '.client_email')
        cleanup_namespace

        # Ensure we can pull test images
        if ! kubectl get namespace instana-autotrace-webhook 2>&1 > /dev/null; then
            kubectl create namespace instana-autotrace-webhook || true
        fi
        kubectl delete secret gcr.io --namespace instana-autotrace-webhook || true
        kubectl create secret docker-registry gcr.io --namespace instana-autotrace-webhook --docker-server=gcr.io --docker-username=_json_key --docker-email="${docker_email}" --docker-password="${GCP_KEY_JSON//$'\n'/}" || true

        # Ensure we can pull test images
        if ! kubectl get namespace test-apps 2>&1 > /dev/null; then
            kubectl create namespace test-apps || true
        fi
        kubectl delete secret gcr.io --namespace test-apps || true
        kubectl create secret docker-registry gcr.io --namespace test-apps --docker-server=gcr.io --docker-username=_json_key --docker-email="${docker_email}" --docker-password="${GCP_KEY_JSON//$'\n'/}" || true


        # Ensure we can pull test images
        for (( i = 0; i < ${#controller_versions_under_test[@]} ; i++ )); do
            RELEASE_SUFFIX="${controller_versions_under_test[$i]//./-}"
            if ! kubectl get namespace ingress-nginx-${RELEASE_SUFFIX} 2>&1 > /dev/null; then
                kubectl create namespace ingress-nginx-${RELEASE_SUFFIX} || true
            fi
            kubectl delete secret gcr.io --namespace ingress-nginx-${RELEASE_SUFFIX} || true
            kubectl create secret docker-registry gcr.io --namespace ingress-nginx-${RELEASE_SUFFIX} --docker-server=gcr.io --docker-username=_json_key --docker-email="${docker_email}" --docker-password="${GCP_KEY_JSON//$'\n'/}" || true

        done

		echo 'Download OpenShift CLI to modify rights for default ServiceAccount, so it has priviliged access'
		curl -L --fail --show-error --silent https://mirror.openshift.com/pub/openshift-v4/clients/oc/latest/linux/oc.tar.gz -o /tmp/oc.tar.gz
		tar -xzvf /tmp/oc.tar.gz --overwrite --directory /tmp

        echo "openshift:
  enabled: true
webhook:
  imagePullSecrets:
    - name: 'gcr.io'" > values.yaml

        AGENT_HELM_ARGUMENTS='--set openshift=true'
        export AGENT_HELM_ARGUMENTS

        EXTRA_AUTOTRACE_CHART_OPTIONS="--values $(pwd)/values.yaml"
        export EXTRA_AUTOTRACE_CHART_OPTIONS

		echo 'Testing on a OpenShift cluster, overriding some parameters for security etc'
		EXTRA_NGINX_HELM_PARAMS="--set serviceAccount.name=ingress-nginx"

        TEST_APPS_HELM_ARGUMENTS='--set imagePullSecrets[0].name=gcr.io'
        export TEST_APPS_HELM_ARGUMENTS

        CLUSTER_NAME=$(echo "${CLUSTER_INFO}" | jq -r .name)
        ;;
    *)
        CLUSTER_NAME=$(kubectl config current-context)
        ;;
esac
