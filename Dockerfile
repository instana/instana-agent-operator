#
# (c) Copyright IBM Corp. 2021, 2025
#
ARG BUILDPLATFORM

FROM --platform=${BUILDPLATFORM} registry.access.redhat.com/ubi9/ubi-minimal:latest AS builder

ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG VERSION=dev
ARG GIT_COMMIT=unspecified
WORKDIR /workspace

# Install packages necessary for preparing the builder
RUN microdnf install -y make jq tar gzip gpg && microdnf clean all


# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Install go using custom installation script
ENV PATH="/usr/local/go/bin:/root/.local/bin:/root/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
COPY installGolang.sh installGolang.sh
RUN export BUILDER_ARCHITECTURE="$(echo ${BUILDPLATFORM} | cut -d'/' -f2)" && ./installGolang.sh ${BUILDER_ARCHITECTURE}

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY version/ version/
COPY pkg/ pkg/
COPY bin/ bin/
COPY Makefile Makefile

RUN make generate

# Build, injecting VERSION and GIT_COMMIT directly in the code
RUN export TARGET_ARCHITECTURE="$(echo ${TARGETPLATFORM} | cut -d'/' -f2)" \
  && CGO_ENABLED=0 GOOS=linux GOARCH="${TARGET_ARCHITECTURE}" GO111MODULE=on  \
  go build -ldflags="-X 'github.com/instana/instana-agent-operator/version.Version=${VERSION}' -X 'github.com/instana/instana-agent-operator/version.GitCommit=${GIT_COMMIT}'" -a -o manager main.go

# Resulting image with actual Operator
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

ARG TARGETPLATFORM
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
