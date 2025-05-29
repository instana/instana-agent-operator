# Instana

## Introduction

Instana is an [APM solution](https://www.ibm.com/products/instana) built for microservices that enables IT Ops to build applications faster and deliver higher quality services by automating monitoring, tracing and root cause analysis. The solution is optimized for [Kubernetes](https://www.ibm.com/products/instana/kubernetes-monitoring) and [OpenShift](https://www.ibm.com/products/instana/supported-technologies/openshift-monitoring).

## Instana Agent Operator

This repository contains the Kubernetes Operator to install and manage the Instana agent.

### Installing

There are two ways to install the operator:

* [Creating the required resources manually](https://www.ibm.com/docs/en/instana-observability/current?topic=agents-installing-kubernetes#install-the-operator-manually)
* [Using the Operator Lifecycle Manager (OLM)](https://www.ibm.com/docs/en/instana-observability/current?topic=openshift-installing-agent-red-hat#installing-the-operator-by-using-olm)

### Configuration

[This documentation section](https://www.ibm.com/docs/en/instana-observability/current?topic=agents-installing-kubernetes#operator-configuration) describes configuration options you can set via the Instana Agent CRD and environment variables.

### Contributing

Please see the guidelines in [CONTRIBUTING.md](CONTRIBUTING.md).

## Local Development

Prerequisites:

* [Make](https://www.gnu.org/software/make/) ([Makefile](Makefile) used as a utility CMD )
* [Go](https://go.dev) (for the supported version, see the [go.mod](go.mod)-file)
* [Kubernetes](http://kubernetes.io)
* [Minikube](https://minikube.sigs.k8s.io/docs/)
* [Operator SDK](https://sdk.operatorframework.io/docs/installation/#install-from-homebrew-macos)
* Something like [Docker](https://www.docker.com/) or [Podman](https://podman.io/)
* Instana Agent key

Developing (and running) the Operator is easiest in two ways:

### **Option 1:** Running Go Operator locally against a **Minikube** cluster

1. Start minikube ([minikube docs](https://minikube.sigs.k8s.io/docs/start/))
   > [!NOTE] RHEL8 & KVM
   > At least `RHEL 8` can have issues reaching the internet which can prevent auto-updates and connections. Try kvm2-driver with by `minikube start --driver=kvm2`. Make sure to have sufficient CPUs and Memory defined before starting minikube.
   > ```shell
   > minikube config set driver kvm2
   > minikube config set cpus 4
   > minikube config set memory 16384
   > ```

   > [!NOTE] Macs
   > Macs using Podman have been successfully run with using `minikube start --driver=podman --container-runtime=cri-o`. More info [here](https://minikube.sigs.k8s.io/docs/drivers/podman/). Make sure to be able to reach outside podman. With default install, one can reach outside by: `podman system connection default podman-machine-default-root`
   

   ```shell
   minikube start
   ```

2. Install the CRD by running `make install` at the root of the repository

   ```shell
   # Install command in root of the repository (installs custom resource to k8s)
   make install
   # List CRD to verify it appears in the list
   kubectl get crd
   ```

3. Create `instana-agent` namespace on the cluster:

   ```shell
   kubectl apply -f config/samples/instana_agent_namespace.yaml
   # List namespaces to verify it appears in the list
   kubectl get ns -n instana-agent
   ```

4. Run the `instana-agent-operator` Go application, either from your IDE, or from command-line: `make run`.

   ```shell
   # Starts the operator using make with additional fmt vet gen functionality
   make run
   ```

5. **Duplicate** agent [sample file](config/samples/instana_v1_instanaagent.yaml) in `config/samples/instana_v1_instanaagent.yaml`
   > [!NOTE]
   for this demonstration the duplicate will be named as `instana_v1_instanaagent_demo.yaml`

   ```shell
   # Copy/Duplicate the sample file with a "demo" suffix
   cp config/samples/instana_v1_instanaagent.yaml config/samples/instana_v1_instanaagent_demo.yaml
   ```

6. Change the placeholder values in the [**duplicated file**](config/samples/instana_v1_instanaagent_demo.yaml) to your preferred values e.g. the Agent `key`, `endpointHost` and `endpointPort`
   > [!TIP]
   In the configuration, there is a field `spec.zone.name`. Changing this to something more identifiable and personalised will help you find your infrastructure easier in the frontend-client.
7. Deploy the custom resource earlier created using

   ```shell
   kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml
   ```

   Verify that the operator reacted to the application of the yaml file by looking into the logs of the running operator
8. Depending on your local configurations, the environment should appear **IBM Instana infrastructure-page**. Standard minikube configuration should appear there as `minikube`.

To stop, take the following actions:

   ```shell
   # Remove the instance from your kubernetes instance
   kubectl delete -f config/samples/instana_v1_instanaagent_demo.yaml
   # Final cleanup e.g `kubectl delete -k config/crd`
   make uninstall
   # Will stop the service
   minikube stop
   # Will reset the whole set-up
   minikube delete
   ```

### **Option 2:** Running Deployment inside the cluster

The Instana Agent Operator can be developed and tested easily against a local Minikube cluster or any other configured
Kubernetes cluster. Therefore, follow the below steps:

1. Create a copy of the file `config/samples/instana_v1_instanaagent.yaml`, for the below steps we're assuming `config/samples/instana_v1_instanaagent_demo.yaml`
2. In this file, put correct values for e.g. the Agent `key`, `endpointHost` and `endpointPort`.
3. Overwrite the default image name with a dev build `export IMG=delivery.instana.io/dev-sandbox-docker-all/${USER}/instana-agent-operator:latest` and build the Operator image: `make docker-build`
4. For deploying on Minikube, there's a convenient target `make deploy-minikube`. For any other environment you would
   need to first push the Docker image to a valid repository using `make docker-push`, then do the deployment
   using `make deploy` to deploy the Operator to the cluster configured for `kubectl`. Note: For non-public registries you might need to create a pull secret first, see `make create-pull-secret` for Instana's Artifactory usage.
5. Deploy the custom resource earlier created using `kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml` or via `make create-cr`

Now you should have a successful running Operator.
To remove the Operator again, run:
* `kubectl delete -f config/samples/instana_v1_instanaagent_demo.yaml`
* `make undeploy`.

If you want to wipe all cluster-wide resources or a broken installation, use `make purge`.

### Running tests

Unit tests can be executed by running `make test` without adjustments of the local environment.

For end-to-end testing, it is necessary to have a valid kubeconfig in the default location and to export variables before starting the test.
An example template file is available in [e2e/.env.template](./e2e/.env.template), copy it to `./e2e/.env` and adjust it accordingly.

The test can be executed by sourcing the config `source ./e2e/.env` and running `make e2e` or by using the VSCode.

For VSCode, ensure to have a valid `.vscode/settings.json` in your root folder.

Example:

```json
{
    "wcaForGP.enable": true,
    "go.testEnvVars": {
        "KUBEBUILDER_ASSETS": "~/.local/share/kubebuilder-envtest/k8s/1.30.0-linux-amd64",
        "INSTANA_API_KEY": "xxx",
        "ARTIFACTORY_USERNAME": "xxx",
        "ARTIFACTORY_PASSWORD": "xxx",
        "OPERATOR_IMAGE_NAME": "xxx",
        "OPERATOR_IMAGE_TAG": "xxx"
    },
    "wca.enable": false,
    "go.testTimeout": "600s",
    "go.testFlags": ["-v"]
}
```
