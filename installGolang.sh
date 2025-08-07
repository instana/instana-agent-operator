#!/bin/bash
# (c) Copyright IBM Corp. 2025
set -euo pipefail
GO_VERSION="${1}"
ARCHITECTURE="${2}"

echo "=== Installing Golang ${GO_VERSION} ==="
echo "Downloading golang binaries"
curl -sLo "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz" "https://go.dev/dl/go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz"

echo "Get checksum"
GO_SHA256=$(curl -s "https://go.dev/dl/?mode=json&include=all" | jq -r '.[] | select(.version=="go'"${GO_VERSION}"'") | .files[] | select(.filename=="go'"${GO_VERSION}"'.linux-'"${ARCHITECTURE}"'.tar.gz") | .sha256')
echo "GO_SHA256=${GO_SHA256}"

echo "Validating checksum"
echo "${GO_SHA256} go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz" | sha256sum --check

echo "Validate signature"
curl -sLo "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz.asc" "https://go.dev/dl/go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz.asc"
curl -sLo linux_signing_key.pub https://dl.google.com/dl/linux/linux_signing_key.pub

gpg --import linux_signing_key.pub
gpg --verify "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz.asc" "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz"

echo "All right, we have legit go binaries, installing it"
tar -C /usr/local -xzf "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz"
rm -f "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz" "go${GO_VERSION}.linux-${ARCHITECTURE}.tar.gz.asc" linux_signing_key.pub