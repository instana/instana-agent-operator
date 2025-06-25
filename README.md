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
* [Minikube](https://minikube.sigs.k8s.io/docs/) or some other context
* Containerization solution like [Docker](https://www.docker.com/) or [Podman](https://podman.io/)
* Instana Agent key

Majority of actions one might do in this repository in regards to local development is handled through make. Run `make help` to see details on the commands available.

Developing (and running) the Operator is easiest in two ways:

### Docker and Podman usage

To be able to run [./Containerfile](./Containerfile) with Docker or Podman, it's necessary to include what platforms are used:
- Specify `--build-arg=TARGETPLATFORM` with the compilation target
- Specify `--build-arg=BUILDPLATFORM` with the build architecture

Examples:
```shell
❯ docker build --build-arg=TARGETPLATFORM=linux/TARGET_ARCHITECTURE --build-arg=BUILDPLATFORM=linux/YOUR_ARCHITECTURE -t instana-agent-operator:latest -f Containerfile .
...
❯ podman build --build-arg=TARGETPLATFORM=linux/TARGET_ARCHITECTURE --build-arg=BUILDPLATFORM=linux/YOUR_ARCHITECTURE -t instana-agent-operator:latest .
```


### Preparing the development environment

After cloning the repository and completing gathering the prerequisites, run:

```shell
# Same as make install-githooks install-tools generate-mocks generate-manifests generate-deepcopies
make install
```

### **Option 1:** Running the operator locally against a **Minikube** cluster

#### 1. Make a **copy** of the [sample file](config/samples/instana_v1_instanaagent.yaml) in `config/samples/instana_v1_instanaagent.yaml`

```shell
# Copy/Duplicate the sample file with a "demo" suffix
cp config/samples/instana_v1_instanaagent.yaml config/samples/instana_v1_instanaagent_demo.yaml
```

#### 2. Change the placeholder values in the [**demo file**](config/samples/instana_v1_instanaagent_demo.yaml) to your preferred values e.g. the Agent `key`, `endpointHost` and `endpointPort`

#### 3. kubectl apply all necessary yaml-files from the config-directory

> Note: Make sure that your minikube instance is running at this point

Run `make kubectl-apply`and optionally extend it with the path to your Instana Agent .yaml-file if it differs from this example.

```shell
make kubectl-apply INSTANA_AGENT_YAML=config/samples/instana_v1_instanaagent_demo.yaml
```

#### 4. Run the `instana-agent-operator` Go application, either from your IDE, or from command-line: `make run`.

```shell
make run
```

#### 5. Verification

Verify that the operator logs resolve like expected:

```shell
❯ make run
...
TIMESTAMP  DEBUG  instana.events  most recent reconcile of agent CR completed without issue
TIMESTAMP  DEBUG  instana.events  All desired Instana Agents are available and using up-to-date configuration 
TIMESTAMP  DEBUG  instana.events  All desired K8sSensors are available and using up-to-date configuration
```

Verify that the pods started as expected:

```shell
❯ kubectl get pods -n instana-agent
NAME                                                READY   STATUS    RESTARTS   AGE
instana-agent-controller-manager                    1/1     Running   0          1s
instana-agent-k8sensor                              1/1     Running   0          1s
instana-agent                                       1/1     Running   0          1s
```

Verify that the agent is reporting to the **IBM Instana infrastructure-page**. Your `instana-agent` should appear there with the identifier `minikube`

#### 6. Once done with everyhing, delete by running

```shell
make kubectl-delete
```

### **Option 2:** Running everything inside a cluster

The Instana Agent Operator can be developed and tested easily against a local Minikube cluster or any other configured
Kubernetes cluster. Therefore, follow the below steps:

1. Create a copy of the file `config/samples/instana_v1_instanaagent.yaml`, for the below steps we're assuming `config/samples/instana_v1_instanaagent_demo.yaml`
2. In this file, put correct values for e.g. the Agent `key`, `endpointHost` and `endpointPort`.
3. Overwrite the default image name with a dev build `export IMG=delivery.instana.io/dev-sandbox-docker-all/${USER}/instana-agent-operator:latest` and build the Operator image: `make container-build`
4. For deploying on Minikube, there's a convenient target `make deploy-minikube`. For any other environment you would
   need to first push the Docker image to a valid repository using `make container-push`, then do the deployment
   using `make deploy` to deploy the Operator to the cluster configured for `kubectl`. Note: For non-public registries you might need to create a pull secret first, see `make create-pull-secret` for Instana's Artifactory usage.
5. Deploy the custom resource earlier created using `kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml`

Now you should have a successful running Operator.
To remove the Operator again, run:
```shell
make kubectl-delete
```

### Testing and linting

#### Linter

Run `make lint` to get print a report of the linting issues in the project.

#### Unit tests

Run `make test` to run unit tests.

#### End-to-end tests

To run end-to-end tests on a local environment, you'll only need a 

1. Create a copy of dotenv file from the [e2e/.env.example](./e2e/.env.example) as `./e2e/.env`.
2. Adjust the fields in the file accordingly
3. Execute by running `make e2e` or using your IDE

### Troubleshooting

   #### Timeouts too fast on VSCode with timeout 30s

   In some situations, like running a slow e2e test, one might want to extend the timeout time. Extending your settings.json in your workspace will give you the ability to extend it as needed.

   `.vscode/settings.json`:
   ```json
   {
      "go.testFlags": [
         "-timeout=2m"
      ]
   }
   ```

   #### Issue with running **minikube** on **Linux** (RHEL8)

   At least `RHEL 8` can have issues reaching the internet which can prevent auto-updates and connections. Try kvm2-driver with by `minikube start --driver=kvm2`. Make sure to have sufficient CPUs and Memory defined before starting minikube.

   ```shell
   minikube config set driver kvm2
   minikube config set cpus 4
   minikube config set memory 16384
   ```

  #### Issue with running **minikube** on **macOS** with **Podman**

   Macs using Podman have been successfully run with using `minikube start --driver=podman --container-runtime=cri-o`. More info [here](https://minikube.sigs.k8s.io/docs/drivers/podman/). Make sure to be able to reach outside podman. With default install, one can reach outside by: `podman system connection default podman-machine-default-root`

   ```shell
   minikube start --driver=podman --container-runtime=cri-o
   podman system connection default podman-machine-default-root
   ```