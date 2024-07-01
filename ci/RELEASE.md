# Operator Catalog Release

Most of the release process is automated as part of the CI/CD pipeline, but the promotion to operator catalogs can cause issues which might require manual fixes.
This document will walk thru the manual steps to produce/fix those PRs accordingly.

The following catalogs require updates:
- https://github.com/k8s-operatorhub/community-operators
- https://github.com/redhat-openshift-ecosystem/certified-operators
- https://github.com/redhat-openshift-ecosystem/redhat-marketplace-operators

## Community and Certified Operator Catalogs
The process to provide a community or certified operator catalog entry is identical except for pushing to different repositories.

```bash
#!/bin/bash
set -xe
# adjust version accordingly
OPERATOR_VERSION=2.1.1
# adjust owner and repo:
# either:
# OWNER=redhat-openshift-ecosystem
# REPO=certified-operators
# or:
# OWNER=k8s-operatorhub
# REPO=community-operators
OWNER=k8s-operatorhub
REPO=community-operators

# no need for changes going forward
curl -OL https://github.com/instana/instana-agent-operator/releases/download/v${OPERATOR_VERSION}/olm-${OPERATOR_VERSION}.zip
OLM_BUNDLE_ZIP=$(ls olm*.zip)
OPERATOR_RELEASE_VERSION=$(echo $OLM_BUNDLE_ZIP | sed 's/olm-\(.*\)\.zip/\1/')
COMMIT_MESSAGE="operator instana-agent-operator ($OPERATOR_RELEASE_VERSION)"
OLM_BUNDLE_ZIP_PATH="$(pwd)/$OLM_BUNDLE_ZIP"

rm -rf community-operators
git clone --depth=1 git@github.com:${OWNER}/${REPO}.git

pushd ${REPO}/operators/instana-agent-operator
git checkout -b "instana-operator-${OPERATOR_RELEASE_VERSION}"
git remote add instana git@github.com:instana/${REPO}.git

mkdir -p $OPERATOR_RELEASE_VERSION
unzip -o $OLM_BUNDLE_ZIP_PATH -d $OPERATOR_RELEASE_VERSION

git add .
git commit -s -m "$COMMIT_MESSAGE" 
git push instana "instana-operator-${OPERATOR_RELEASE_VERSION}"

popd
```

Create the PR afterwards, they should be auto-merged once the CI/CD pipeline passes.

## Red Hat Marketplace Operator

The Marketplace Operator requires file editing before pushing, otherwise the process is equal

```bash
#!/bin/bash
set -xe
# adjust version accordingly
OPERATOR_VERSION=2.1.1

OWNER=redhat-openshift-ecosystem
REPO=redhat-marketplace-operators

# no need for changes going forward
curl -OL https://github.com/instana/instana-agent-operator/releases/download/v${OPERATOR_VERSION}/olm-${OPERATOR_VERSION}.zip

OLM_BUNDLE_ZIP=$(ls olm*.zip)
OPERATOR_RELEASE_VERSION=$(echo $OLM_BUNDLE_ZIP | sed 's/olm-\(.*\)\.zip/\1/')
COMMIT_MESSAGE="operator instana-agent-operator ($OPERATOR_RELEASE_VERSION)"
OLM_BUNDLE_ZIP_PATH="$(pwd)/$OLM_BUNDLE_ZIP"

rm -rf ${REPO}
git clone --depth=1 git@github.com:${OWNER}/${REPO}.git

pushd ${REPO}/operators/instana-agent-operator-rhmp
git checkout -b "instana-operator-${OPERATOR_RELEASE_VERSION}"
git remote add instana git@github.com:instana/${REPO}.git

mkdir -p $OPERATOR_RELEASE_VERSION
unzip -o $OLM_BUNDLE_ZIP_PATH -d $OPERATOR_RELEASE_VERSION
pushd $OPERATOR_RELEASE_VERSION

pushd manifests
yq -i '.metadata.annotations += {"marketplace.openshift.io/remote-workflow": "https://marketplace.redhat.com/en-us/operators/instana-agent-operator-rhmp/pricing?utm_source=openshift_console"}' instana-agent-operator.clusterserviceversion.yaml
yq -i '.metadata.annotations += {"marketplace.openshift.io/support-workflow": "https://marketplace.redhat.com/en-us/operators/instana-agent-operator-rhmp/support?utm_source=openshift_console"}' instana-agent-operator.clusterserviceversion.yaml
mv instana-agent-operator.clusterserviceversion.yaml instana-agent-operator-rhmp.clusterserviceversion.yaml
popd

pushd metadata
yq -i '.annotations."operators.operatorframework.io.bundle.package.v1" |= "instana-agent-operator-rhmp"' annotations.yaml
popd

popd

git add .
git commit -s -m "$COMMIT_MESSAGE" 
git push instana "instana-operator-${OPERATOR_RELEASE_VERSION}"

popd
```

Create the PR afterwards, they should be auto-merged once the CI/CD pipeline passes.