#!/usr/bin/env bash
set -eo pipefail
set +u
echo "===== setup.sh - start ====="

# use CEL expression on trigger if commit should be skipped: header['x-github-event'] == 'push' && body.ref == 'refs/heads/ko-sps' && !body.head_commit.message.contains('[skip ci]')
# see: https://cloud.ibm.com/docs/ContinuousDelivery?topic=ContinuousDelivery-tekton-pipelines&interface=ui#configure_triggering_events

echo "Installing dependencies"
source $WORKSPACE/$APP_REPO_FOLDER/installGolang.sh 1.24.4 amd64
export PATH=$PATH:/usr/local/go/bin

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