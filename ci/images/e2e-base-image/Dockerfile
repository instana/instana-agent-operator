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
    wget \
    zip \
    google-cloud-cli \
    skopeo \
    kubectl \
    && pip3 install pyyaml

# Set environment variables for Go
ENV GOPATH=/go
ENV GO_VERSION=1.23.0
ENV PATH="$PATH:/usr/local/go/bin:$GOPATH/bin"

# Install go
RUN GO_SHA256="905a297f19ead44780548933e0ff1a1b86e8327bb459e92f9c0012569f76f5e3 go${GO_VERSION}.linux-amd64.tar.gz"  \
    && curl -L --fail --show-error --silent "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o "go${GO_VERSION}.linux-amd64.tar.gz" \
    && echo "${GO_SHA256}" | sha256sum --check \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz" \
    && mkdir -p "${GOPATH}" \
    && go version

# Install required gcloud package
RUN yum install -y google-cloud-sdk-gke-gcloud-auth-plugin

# Install yq
RUN wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/bin/yq && chmod +x /usr/bin/yq

# Install manifest-tool
RUN curl -o /tmp/manifest-tool.tar.gz -L https://github.com/estesp/manifest-tool/releases/download/v2.1.6/binaries-manifest-tool-2.1.6.tar.gz \
    && tar -xf /tmp/manifest-tool.tar.gz -C /tmp \
    && mv /tmp/manifest-tool-linux-amd64 /usr/local/bin/manifest-tool \
    && chmod +x /usr/local/bin/manifest-tool \
    && rm /tmp/manifest-tool.tar.gz \
    && rm -rf /tmp/* \
    && manifest-tool --version

ENV PATH="$PATH:/usr/local/bin"
