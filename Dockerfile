#
# (c) Copyright IBM Corp. 2021
# (c) Copyright Instana Inc.
#

# Build the manager binary
FROM golang:1.15 as builder

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
COPY version/ version/
COPY logger/ logger/
COPY controllers/ controllers/
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /
COPY --from=builder /workspace/manager .
RUN mkdir -p .cache/helm/repository/
RUN chmod 777 .cache/helm/repository/
USER 65532:65532
ENTRYPOINT ["/manager"]