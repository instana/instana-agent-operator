Run an Operator Registry Locally
--------------------------------

[install-via-olm.md](install-via-olm.md) describes how to install the `instana-agent-operator` using the [Operator Lifecycle Manager (OLM)](https://github.com/operator-framework/operator-lifecycle-manager). By default, the OLM will download the `instana-agent-operator` bundle from [operatorhub.io](https://operatorhub.io).

Most users should use the bundle from [operatorhub.io](https://operatorhub.io). However, if you want to make changes to the operator bundle, it is convenient to have your own operator registry running locally, so that you can experiment with your changes. This document shows how to run the OLM and an [operator registry](https://github.com/operator-framework/operator-registry) locally.

### Step 1: Run the Operator Livecycle Manager (OLM)

If you don't have a OLM running, you can install it as follows:

Get the `operator-lifecycle-manager`:

```
git clone https://github.com/operator-framework/operator-lifecycle-manager
```

Run the `operator-lifecycle-manager` locally in Minikube:

```
minikube start --memory 4096 --cpus 4
cd operator-lifecycle-manager
make run-local
./scripts/run_console_local.sh
```

The OLM should now be accessable on [http://localhost:9000](http://localhost:9000)

### Step 2: Run an Operator Registry for the Instana Agent Operator

The OLM will get the `instana-agent-operator` from a registry. In order to test this locally, we create our own registry as a Docker image. The image is based on the official `example-registry` image with our instana-agent-operator CSV added.

Get the `operator-registry`:

```
git clone https://github.com/operator-framework/operator-registry.git
cd operator-registry
```

Copy the operator configuration into the operator-registry's example `./manifests/` directory:

```
export PR_BRANCH=https://raw.githubusercontent.com/instana/community-operators/instana-agent-operator/upstream-community-operators/instana-agent

mkdir ./manifests/instana-agent
cd ./manifests/instana-agent
curl -OL $PR_BRANCH/instana-agent.package.yaml

mkdir ./0.0.2
cd ./0.0.2
curl -OL $PR_BRANCH/instana-agent.crd.yaml
curl -OL $PR_BRANCH/instana-agent.v0.0.2.clusterserviceversion.yaml

cd ../../../
```

Build the `example-registry` Docker image:

```
eval $(minikube docker-env)
docker build -t example-registry:latest -f upstream-example.Dockerfile .
```

_TODO: It seems the OLM always pulls the registry image from a remote reopository, so it might be necessary to `docker push` the registry to a remote repository._

### Step 3: Use the Registry as a Catalog Source in the Local OLM

On [http://localhost:9000](http://localhost:9000) click on _Add_ -> _Import YAML_ in the top right corner, then copy and paste the following:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: example
  namespace: local
spec:
  displayName: Instana Operators
  publisher: instana.io
  sourceType: grpc
  image: example-registry:latest
```

_TODO: If you have pushed the `example-registry` image to a remote repository above, change the `image:` prefix accordingly so that the image is pulled from there._

Result
------

On [http://localhost:9000](http://localhost:9000) under _Operator Management_ -> _Create Subscription_ you should now be able to choose the `instana-agent-operator`. If it's not there immediately, wait a minute and reload the page, as it takes a while until the OLM runs your example registry Pod.

Make sure that you execute the steps in [install-via-olm.md](install-via-olm.md) before creating the subscription:

* `instana-agent` namespace
* `instana-agent` operator group
* `instana-agent` custom resoure
