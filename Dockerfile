#
# (c) Copyright IBM Corp. 2021
# (c) Copyright Instana Inc.
#

# Build the manager binary
FROM golang:1.15 as builder

ARG VERSION=dev
ARG GIT_COMMIT=unspecified

WORKDIR /workspace

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

# Build, injecting VERSION and GIT_COMMIT directly in the code
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on \
	go build -ldflags="-X 'github.com/instana/instana-agent-operator/version.Version=${VERSION}' -X 'github.com/instana/instana-agent-operator/version.GitCommit=${GIT_COMMIT}'" -a -o manager main.go

# Resulting image with actual Operator
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
MAINTAINER Instana, support@instana.com

ARG VERSION=dev
ARG BUILD=1
ARG GIT_COMMIT=unspecified

LABEL name="instana-agent-operator" \
      vendor="Instana Inc" \
      maintainer="Instana Inc" \
      version=$VERSION \
      build=$BUILD \
      git-commit=$GIT_COMMIT \
      summary="Kubernetes / OpenShift Operator for the Instana APM Agent" \
      description="This operator will deploy a daemon set to run the Instana APM Agent on each cluster node."

ENV OPERATOR=instana-agent-operator \
    USER_UID=1001 \
    USER_NAME=instana-agent-operator

WORKDIR /
COPY --from=builder /workspace/manager .
COPY LICENSE /licenses/

RUN mkdir -p .cache/helm/repository/
RUN chown -R ${USER_UID}:${USER_UID} .cache

USER ${USER_UID}:${USER_UID}
ENTRYPOINT ["/manager"]
