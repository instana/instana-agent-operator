#
# (c) Copyright IBM Corp. 2021, 2025
# (c) Copyright Instana Inc.
#

# Build the manager binary, always build on amd64 platform
FROM --platform=linux/amd64 registry.access.redhat.com/ubi9/ubi-minimal:latest AS builder

ARG TARGETPLATFORM='linux/amd64'
ARG VERSION=dev
ARG GIT_COMMIT=unspecified
ARG GO_VERSION=1.24.4
WORKDIR /workspace
ENV PATH="/usr/local/go/bin:/root/.local/bin:/root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
COPY installGolang.sh installGolang.sh
RUN ./installGolang.sh ${GO_VERSION}

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY version/ version/
COPY pkg/ pkg/

# Build, injecting VERSION and GIT_COMMIT directly in the code
RUN export ARCH=$(case "${TARGETPLATFORM}" in 'linux/amd64') echo 'amd64' ;; 'linux/arm64') echo 'arm64' ;; 'linux/s390x') echo 's390x' ;; 'linux/ppc64le') echo 'ppc64le' ;; esac) \
    && CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" GO111MODULE=on \
	go build -ldflags="-X 'github.com/instana/instana-agent-operator/version.Version=${VERSION}' -X 'github.com/instana/instana-agent-operator/version.GitCommit=${GIT_COMMIT}'" -a -o manager main.go

# Resulting image with actual Operator
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ARG TARGETPLATFORM='linux/amd64'
ARG VERSION=dev
ARG BUILD=1
ARG GIT_COMMIT=unspecified
ARG DATE=""

LABEL name="instana-agent-operator" \
      vendor="Instana Inc" \
      maintainer="Instana Inc" \
      version=$VERSION \
      release=$VERSION \
      build=$BUILD \
      build-date=$DATE \
      git-commit=$GIT_COMMIT \
      summary="Kubernetes / OpenShift Operator for the Instana APM Agent" \
      description="This operator will deploy a daemon set to run the Instana APM Agent on each cluster node." \
      url="https://catalog.redhat.com/software/containers/instana/instana-agent-operator/5cd2efc469aea3638b0fcff3" \
      io.k8s.display-name="Instana Agent Operator" \
      io.openshift.tags="" \
      io.k8s.description="" \
      com.redhat.build-host="" \
      com.redhat.component="" \
      org.opencontainers.image.authors="Instana, support@instana.com"

ENV OPERATOR=instana-agent-operator \
    USER_UID=1001 \
    USER_NAME=instana-agent-operator

RUN microdnf update -y \
  && microdnf clean all

WORKDIR /
COPY --from=builder /workspace/manager .
COPY LICENSE /licenses/

RUN mkdir -p .cache/helm/repository/
RUN chown -R ${USER_UID}:${USER_UID} .cache
RUN chmod -R 777 .cache

USER ${USER_UID}:${USER_UID}
ENTRYPOINT ["/manager"]
