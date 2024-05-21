#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -x #TODO: remove before merging
set -e
set -o pipefail

echo "running e2e tests"


function get_public_image() {
    git pull -r
    VERSION=$(git tag | sort -r --version-sort | head -n1)
    echo "Latest release is ${VERSION}"
    export PREFIX="v"
    export OPERATOR_DOCKER_VERSION=${VERSION#"$PREFIX"}

    export IMG="icr.io/instana/instana-agent-operator:$OPERATOR_DOCKER_VERSION"
    echo "IMG=$IMG"
}

source pipeline-source/ci/scripts/cluster-authentication.sh

## deploy operator from main branch
pushd pipeline-source
    get_public_image

    make install
    # make deploy
popd

# check if the operator and agent pods are running

## upgrade to the operator from the release branch
# check if the operator and agent pods are running
