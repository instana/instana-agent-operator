# Instana

## Introduction

Instana is an [APM solution](https://www.instana.com/product-overview/) built for microservices that enables IT Ops to build applications faster and deliver higher quality services by automating monitoring, tracing and root cause analysis. The solution is optimized for [Kubernetes](https://www.instana.com/automatic-kubernetes-monitoring/) and [OpenShift](https://www.instana.com/blog/automatic-root-cause-analysis-for-openshift-applications/).

## Instana Agent Operator

This repository contains the Kubernetes Operator to install and manage the Instana agent.

### Installing

There are two ways to install the operator:

* [Creating the required resources manually](https://www.instana.com/docs/setup_and_manage/host_agent/on/kubernetes/#install-operator-manually)
* [Using the Operator Lifecycle Manager (OLM)](https://www.instana.com/docs/setup_and_manage/host_agent/on/openshift/#install-operator-via-olm)

**IMPORTANT NOTE** For the conversion Mutating WebHook, converting older `v1beta1` Specs to `v1` Spec, the Cert-Manager needs to
be installed (unless using OLM). See the [Cert Manager docs](https://cert-manager.io/docs/installation/) for how to
install and configure the Cert-Manager properly.
Alternatively, every release will include the [instana-agent-operator-no-conversion-webhook.yaml](https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator-no-conversion-webhook.yaml) variant for easily installing the Operator without the conversion WebHook.

### Configuration

[This documentation section](https://www.instana.com/docs/setup_and_manage/host_agent/on/kubernetes#operator-configuration) describes configuration options you can set via the Instana Agent CRD and environment variables.

### Building

[![CircleCI](https://circleci.com/gh/instana/instana-agent-operator.svg?style=svg)](https://circleci.com/gh/instana/instana-agent-operator)

* [docs/build.md](docs/build.md) describes how to build the Docker image from source code.
* [docs/testing-with-kind.md](docs/testing-with-kind.md) shows how to test the operator in a local Kind cluster.
* [docs/run-operator-registry-locally.md](docs/run-operator-registry-locally.md) describes how to set up a local Operator Lifecycle Manager and Registry to test the OLM deployment locally.

### Contributing

Please see the guidelines in [CONTRIBUTING.md](CONTRIBUTING.md).

## Local Development

Developing (and running) the Operator is possible in two easy ways:
- Running as Go application outside the Cluster
- Running as Deployment inside the Cluster

Both are described below.

### Running Go Operator locally against a K8s cluster

1. Create a copy of the file `config/samples/instana_v1_instanaagent.yaml`, for the below steps we're assuming `config/samples/instana_v1_instanaagent_demo.yaml.yaml`
2. In this file, put correct values for e.g. the Agent `key`, `endpointHost` and `endpointPort`.
3. Install the CRD and run the Go application: `make install run`.
4. Deploy the custom resource earlier created using `kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml`

To stop, take the following actions:
- `kubectl delete -f config/samples/instana_v1_instanaagent_demo.yaml`
- `make uninstall`

### Running Deployment inside the cluster

The Instana Agent Operator can be developed and tested easily against a local Minikube cluster or any other configured
Kubernetes cluster. Therefore, follow the below steps:

1. Create a copy of the file `config/samples/instana_v1_instanaagent.yaml`, for the below steps we're assuming `config/samples/instana_v1_instanaagent_demo.yaml.yaml`
2. In this file, put correct values for e.g. the Agent `key`, `endpointHost` and `endpointPort`.
3. Build the Operator image: `make docker-build`
4. For deploying on Minikube, there's a convenient target `make deploy-minikube`. For any other environment you would
   need to first push the Docker image to a valid repository using `make docker-push`, then do the deployment
   using `make deploy` to deploy the Operator to the cluster configured for `kubectl`.
5. Deploy the custom resource earlier created using `kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml.yaml`

Now you should have a successful running Operator.
To remove the Operator again, run:
- `kubectl delete -f config/samples/instana_v1_instanaagent_demo.yaml.yaml`
- `make undeploy`.

