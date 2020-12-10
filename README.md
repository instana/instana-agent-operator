# Instana

## Introduction

Instana is an [APM solution](https://www.instana.com/product-overview/) built for microservices that enables IT Ops to build applications faster and deliver higher quality services by automating monitoring, tracing and root cause analysis. The solution is optimized for [Kubernetes](https://www.instana.com/automatic-kubernetes-monitoring/) and [OpenShift](https://www.instana.com/blog/automatic-root-cause-analysis-for-openshift-applications/).

## Instana Agent Operator

This repository contains the Kubernetes Operator to install and manage the Instana agent.

### Installing

There are two ways to install the operator:

* [Creating the required resources manually](https://www.instana.com/docs/setup_and_manage/host_agent/on/kubernetes/#install-operator-manually)
* [Using the Operator Lifecycle Manager (OLM)](https://www.instana.com/docs/setup_and_manage/host_agent/on/openshift/#install-operator-via-olm)

### Configuration

[This documentation section](https://www.instana.com/docs/setup_and_manage/host_agent/on/kubernetes#operator-configuration) describes configuration options you can set via the Instana Agent CRD and environment variables.

### Building

[![CircleCI](https://circleci.com/gh/instana/instana-agent-operator.svg?style=svg)](https://circleci.com/gh/instana/instana-agent-operator)

* [docs/build.md](docs/build.md) describes how to build the Docker image from source code.
* [docs/testing-with-kind.md](docs/testing-with-kind.md) shows how to test the operator in a local Kind cluster.
* [docs/run-operator-registry-locally.md](docs/run-operator-registry-locally.md) describes how to set up a local Operator Lifecycle Manager and Registry to test the OLM deployment locally.
