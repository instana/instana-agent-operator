#!/bin/bash
#
# (c) Copyright IBM Corp. 2025
# (c) Copyright Instana Inc.
#

echo "Setting up ocp mirror and ImageContentSourcePolicy to pull from internal registry (assumes valid e2e/.env config was sourced upfront)"
oc patch configs.imageregistry.operator.openshift.io/cluster --type merge -p '{"spec":{"defaultRoute":true}}'
HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
echo "Current registry host: ${HOST}"

echo "Creating project instana"
oc get project instana || oc new-project instana
# oc extract secret/$(oc get ingresscontroller -n openshift-ingress-operator default -o json | jq '.spec.defaultCertificate.name // "router-certs-default"' -r) -n openshift-ingress --confirm
skopeo login -u kubeadmin -p "$(oc whoami -t)" "${HOST}" --tls-verify=false
skopeo login -u _ -p "${INSTANA_API_KEY}" containers.instana.io
skopeo login -u "${ARTIFACTORY_USERNAME}" -p "${ARTIFACTORY_PASSWORD}" delivery.instana.io
set -x
skopeo copy docker://icr.io/instana/instana-agent-operator:latest "docker://${HOST}/instana/instana-agent-operator:latest" --dest-tls-verify=false
skopeo copy docker://containers.instana.io/instana/release/agent/static:latest "docker://${HOST}/instana/instana-agent-static:latest" --dest-tls-verify=false
skopeo copy docker://icr.io/instana/agent:latest "docker://${HOST}/instana/agent:latest" --dest-tls-verify=false
skopeo copy docker://icr.io/instana/k8sensor:latest "docker://${HOST}/instana/k8sensor:latest" --dest-tls-verify=false
skopeo copy "docker://${OPERATOR_IMAGE_NAME}:${OPERATOR_IMAGE_TAG}" "docker://$HOST/instana/instana-agent-operator:${OPERATOR_IMAGE_TAG}" --dest-tls-verify=false
set +x

# cat <<EOF > image-content-source-policy.yaml
# apiVersion: operator.openshift.io/v1alpha1
# kind: ImageContentSourcePolicy
# metadata:
#   name: registry-mirror
# spec:
#   repositoryDigestMirrors:
#   - mirrors:
#     - ${HOST}/instana/instana-agent-operator
#     source: icr.io/instana/instana-agent-operator
#     pullFromMirror: "all"
#   - mirrors:
#     - ${HOST}/instana/agent
#     source: icr.io/instana/agent
#     pullFromMirror: "all"
#   - mirrors:
#     - ${HOST}/instana/instana-agent-static
#     source: containers.instana.io/instana/release/agent/static
#     pullFromMirror: "all"
#   - mirrors:
#     - ${HOST}/instana/k8sensor
#     source: icr.io/instana/k8sensor
#     pullFromMirror: "all"
# EOF

# only available since OCP v4.13
cat <<EOF > image-tag-mirror-set.yaml
apiVersion: config.openshift.io/v1
kind: ImageTagMirrorSet
metadata:
  name: instana-image-tag-mirror-set
spec:
  imageTagMirrors:
  - mirrors:
    - ${HOST}/instana/instana-agent-operator
    source: icr.io/instana/instana-agent-operator
  - mirrors:
    - ${HOST}/instana/agent
    source: icr.io/instana/agent
  - mirrors:
    - ${HOST}/instana/instana-agent-static
    source: containers.instana.io/instana/release/agent/static
  - mirrors:
    - ${HOST}/instana/k8sensor
    source: icr.io/instana/k8sensor
  - mirrors:
    - ${HOST}/instana/instana-agent-operator
    source: ${OPERATOR_IMAGE_NAME}
EOF


#oc apply -f image-content-source-policy.yaml
# rm -f image-content-source-policy.yaml
oc apply -f image-tag-mirror-set.yaml
rm -f image-tag-mirror-set.yaml

oc policy add-role-to-group system:image-puller system:serviceaccounts:instana-agent -n instana
