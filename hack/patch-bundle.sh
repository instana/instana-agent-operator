#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2021
# (c) Copyright Instana Inc. 2021

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(git rev-parse --show-toplevel)
BUNDLE_DOCKERFILE="$ROOT/bundle.Dockerfile"
BUNDLE_ANNOTATIONS="$ROOT/bundle/metadata/annotations.yaml"

# Patch Dockerfile
cat <<EOF >> $BUNDLE_DOCKERFILE

# Allow bundle to be published on 4.5 and beyond
LABEL com.redhat.openshift.versions="v4.5"
LABEL com.redhat.delivery.operator.bundle=true
LABEL com.redhat.delivery.backport=false
EOF

# Patch Dockerfile
cat <<EOF >> $BUNDLE_ANNOTATIONS

  # Allow bundle to be published on 4.5 and beyond
  com.redhat.openshift.versions: "v4.5"
  com.redhat.delivery.operator.bundle: true
  com.redhat.delivery.backport: false
EOF
