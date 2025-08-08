#!/usr/bin/env bash
set -eo pipefail
set +u
echo "===== setup.sh - start ====="

# use CEL expression on trigger if commit should be skipped: header['x-github-event'] == 'push' && body.ref == 'refs/heads/ko-sps' && !body.head_commit.message.contains('[skip ci]')
# see: https://cloud.ibm.com/docs/ContinuousDelivery?topic=ContinuousDelivery-tekton-pipelines&interface=ui#configure_triggering_events

echo "Installing dependencies"
dnf -y install rsync
echo "Installing helm"
curl -L --silent --fail --show-error https://get.helm.sh/helm-v3.17.3-linux-amd64.tar.gz | tar -zx linux-amd64/helm
mv linux-amd64/helm /bin/helm

GO_VERSION=1.24.3
echo "=== Installing Golang ${GO_VERSION} ==="
echo "Downloading golang binaries"
curl -sLo "go${GO_VERSION}.linux-amd64.tar.gz" "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"

echo "Get checksum"
GO_SHA256=$(curl -s "https://go.dev/dl/?mode=json&include=all" | jq -r '.[] | select(.version=="go'${GO_VERSION}'") | .files[] | select(.filename=="go'${GO_VERSION}'.linux-amd64.tar.gz") | .sha256')
echo "GO_SHA256=${GO_SHA256}"

echo "Validating checksum"
echo "${GO_SHA256} go${GO_VERSION}.linux-amd64.tar.gz" | sha256sum --check

echo "Validate signature"
curl -sLo go${GO_VERSION}.linux-amd64.tar.gz.asc "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz.asc"
curl -sLo linux_signing_key.pub https://dl.google.com/dl/linux/linux_signing_key.pub

gpg --import linux_signing_key.pub
gpg --verify go${GO_VERSION}.linux-amd64.tar.gz.asc go${GO_VERSION}.linux-amd64.tar.gz

echo "All right, we have legit go binaries, installing it"
tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
rm -f go${GO_VERSION}.linux-amd64.tar.gz

echo "=== Installing kubectl ==="
curl --silent --fail --show-error -L "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl
chmod u+x /usr/local/bin/kubectl

if [ "${SKIP_INSTALL_GCLOUD}" == "true" ]; then
    echo "Skipping gcloud installation as not needed in this stage"
else
    echo "=== Installing gloud cli ==="
    tee -a /etc/yum.repos.d/google-cloud-sdk.repo << EOM
[google-cloud-cli]
name=Google Cloud CLI
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el9-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOM
    dnf install -y google-cloud-cli google-cloud-sdk-gke-gcloud-auth-plugin
fi

echo "Showing available disk space"
df -h
echo "===== setup.sh - end ====="