# Instana

## Introduction

Instana is an [APM solution](https://www.ibm.com/products/instana) built for microservices that enables IT Ops to build applications faster and deliver higher quality services by automating monitoring, tracing and root cause analysis. The solution is optimized for [Kubernetes](https://www.ibm.com/products/instana/kubernetes-monitoring) and [OpenShift](https://www.ibm.com/products/instana/supported-technologies/openshift-monitoring).

## Instana Agent Operator

This repository contains the Kubernetes Operator to install and manage the Instana agent.

### Installing

There are two ways to install the operator:

* [Creating the required resources manually](https://www.ibm.com/docs/en/instana-observability/current?topic=agents-installing-host-agent-kubernetes#install-the-operator-manually)
* [Using the Operator Lifecycle Manager (OLM)](https://www.ibm.com/docs/en/instana-observability/current?topic=agents-installing-host-agent-openshift#installing-the-operator-by-using-olm)

### Configuration

[This documentation section](https://www.ibm.com/docs/en/instana-observability/current?topic=agents-installing-host-agent-kubernetes#operator-configuration) describes configuration options you can set via the Instana Agent CRD and environment variables.

### Contributing

Please see the guidelines in [CONTRIBUTING.md](CONTRIBUTING.md).

## Local Development

Developing (and running) the Operator is possible in two easy ways:
- Running as Go application outside the Cluster
- Running as Deployment inside the Cluster

Both are described below.

### Running Go Operator locally against a (Minikube) cluster

1. Create a copy of the file `config/samples/instana_v1_instanaagent.yaml`, for the below steps we're assuming `config/samples/instana_v1_instanaagent_demo.yaml`
2. In this file, put correct values for e.g. the Agent `key`, `endpointHost` and `endpointPort`.
3. Create a minikube cluster: https://minikube.sigs.k8s.io/docs/start/
   Note: When minikube runs on docker (at least on RHEL 8), there are network issues for pods reaching the internet. This causes connection issues for the agent and will prevent auto-updates or connections to the backend. To avoid this, use kvm2 instead: `minikube start --driver=kvm2`.
4. Install the CRD: `make install`, check if the CRD agents.instana.io afterwards by issuing `kubectl get crd`
5. Run the Go application, either from your IDE, or from command-line: `make run`.
6. Create a new namespace on the cluster: `kubectl apply -f config/samples/instana_agent_namespace.yaml`
7. Deploy the custom resource earlier created using `kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml`

To stop, take the following actions:
- `kubectl delete -f config/samples/instana_v1_instanaagent_demo.yaml`
- `make uninstall`

### Running Deployment inside the cluster

The Instana Agent Operator can be developed and tested easily against a local Minikube cluster or any other configured
Kubernetes cluster. Therefore, follow the below steps:

1. Create a copy of the file `config/samples/instana_v1_instanaagent.yaml`, for the below steps we're assuming `config/samples/instana_v1_instanaagent_demo.yaml`
2. In this file, put correct values for e.g. the Agent `key`, `endpointHost` and `endpointPort`.
3. Build the Operator image: `make docker-build`
4. For deploying on Minikube, there's a convenient target `make deploy-minikube`. For any other environment you would
   need to first push the Docker image to a valid repository using `make docker-push`, then do the deployment
   using `make deploy` to deploy the Operator to the cluster configured for `kubectl`.
5. Deploy the custom resource earlier created using `kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml`

Now you should have a successful running Operator.
To remove the Operator again, run:
- `kubectl delete -f config/samples/instana_v1_instanaagent_demo.yaml`
- `make undeploy`.

