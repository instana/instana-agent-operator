FROM registry.access.redhat.com/ubi9/ubi:latest

# Setup for gcloud sdk
RUN mkdir -p /etc/yum.repos.d && \
    echo '[google-cloud-cli]' > /etc/yum.repos.d/google-cloud-sdk.repo && \
    echo 'name=Google Cloud CLI' >> /etc/yum.repos.d/google-cloud-sdk.repo && \
    echo 'baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el9-x86_64' >> /etc/yum.repos.d/google-cloud-sdk.repo && \
    echo 'enabled=1' >> /etc/yum.repos.d/google-cloud-sdk.repo && \
    echo 'gpgcheck=1' >> /etc/yum.repos.d/google-cloud-sdk.repo && \
    echo 'repo_gpgcheck=0' >> /etc/yum.repos.d/google-cloud-sdk.repo && \
    echo 'gpgkey=https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg' >> /etc/yum.repos.d/google-cloud-sdk.repo 

# Install tools
RUN dnf update -y && dnf install -y \
    make \
    python3-devel \
    gcc \
    git \
    jq \
    git \
    zip \
    google-cloud-cli \
    skopeo \
    kubectl \
    && pip3 install pyyaml

# Set environment variables for Go
ENV GOPATH=/go
ENV GO_VERSION=1.22.3
ENV PATH="$PATH:/usr/local/go/bin"

# Install go
RUN GO_SHA256="8920ea521bad8f6b7bc377b4824982e011c19af27df88a815e3586ea895f1b36 go${GO_VERSION}.linux-amd64.tar.gz"  \
    && curl -L --fail --show-error --silent "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o "go${GO_VERSION}.linux-amd64.tar.gz" \
    && echo "${GO_SHA256}" | sha256sum --check \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz" \
    && mkdir -p "${GOPATH}"

# Verify Go installation
RUN go version

# Install required gcloud package
RUN yum install -y google-cloud-sdk-gke-gcloud-auth-plugin
