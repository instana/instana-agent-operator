#!/bin/bash
# (c) Copyright IBM Corp. 2025
set -euo pipefail
# renovate: datasource=golang-version depName=golang
GO_VERSION="1.26.2"
ARCHITECTURE="${1}"

# Make sure to remove the old go version, otherwise weird runtime errors may occur
# Example error when 1.26.1 is installed on 1.25 without removing the old install first
# internal/abi
# /usr/local/go/src/internal/abi/map_swiss.go:25:2: ctrlEmpty redeclared in this block
# 	/usr/local/go/src/internal/abi/map.go:25:2: other declaration of ctrlEmpty
# /usr/local/go/src/internal/abi/map_swiss.go:26:2: bitsetLSB redeclared in this block
# 	/usr/local/go/src/internal/abi/map.go:26:2: other declaration of bitsetLSB
echo === Removing old golang if present ===
rm -rf /usr/local/go

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