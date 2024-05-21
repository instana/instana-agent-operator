#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -x #TODO: remove before merging
set -e
set -o pipefail

echo "running e2e tests"

source pipeline-source/ci/scripts/cluster-authentication.sh