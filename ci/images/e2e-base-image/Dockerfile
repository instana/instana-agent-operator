FROM redhat/ubi9

# Setup for gcloud sdk
RUN tee -a /etc/yum.repos.d/google-cloud-sdk.repo <<EOF
\[google-cloud-cli\]
name=Google Cloud CLI
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el9-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF

# Install tools
RUN dnf update -y && dnf install -y \
    make \
    python3-devel \
    gcc \
    git \
    jq \
    git \
    google-cloud-cli \
    skopeo \
    && pip3 install pyyaml

# Install go
RUN export PATH="$PATH:/usr/local/go/bin" \
    && export GOPATH=/go \
    && export GO_VERSION=1.22.3  \
    && export GO_SHA256="8920ea521bad8f6b7bc377b4824982e011c19af27df88a815e3586ea895f1b36 go${GO_VERSION}.linux-amd64.tar.gz"  \
    && curl -L --fail --show-error --silent "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o "go${GO_VERSION}.linux-amd64.tar.gz" \
    && echo "${GO_SHA256}" | sha256sum --check \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz" \
    && mkdir -p "${GOPATH}"


