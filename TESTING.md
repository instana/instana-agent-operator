Testing with Kind
-----------------

One of the main features of the `instana-agent-operator` is to do leader election. In order to test this, we need a Kubernetes cluster where we can _destroy nodes_, so that we can observe if a new leader is elected in that case.

This page describes how to set up a local Kubernetes cluster with [Kind](https://kind.sigs.k8s.io/) and test the `instana-agent-operator` in that cluster.

Kind (**K**ubernetes **in** **D**ocker) will set up a local cluster where all nodes and the master run in Docker containers.

### Set up a local Kubernetes Cluster with Kind

Create `kind-config.yaml` with the following content (replace `/home` with `/Users` on macOS):

```yaml
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
nodes:
- role: control-plane
  extraMounts:
    - containerPath: /hosthome
      hostPath: /home
- role: worker
  extraMounts:
    - containerPath: /hosthome
      hostPath: /home
- role: worker
  extraMounts:
    - containerPath: /hosthome
      hostPath: /home
```

Install [kind](https://kind.sigs.k8s.io/) (a single executable that can be downloaded from the [Github release page](https://github.com/kubernetes-sigs/kind/releases)) and run the following commands to create the cluster:

```sh
kind --config kind-config.yaml create cluster
export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"
kubectl get nodes
```

You should see a cluster with one control-pane and two worker nodes up and running.

### Deploy the Instana Operator

In the `instana-agent-operator` project, build the `instana/instana-agent-operator` Docker image and push it to the local Kind cluster:

```sh
mvn package docker:build
kind load docker-image instana/instana-agent-operator
```

Pull the `instana/agent` Docker image and push it to the local Kind cluster:

```sh
docker pull instana/agent
kind load docker-image instana/agent
```

Use the files `instana-agent-operator-rbac.yaml` and `instana-agent-operator-deploy.yaml` to install the operator:

```sh
kubectl apply -f instana-agent-operator-rbac.yaml
kubectl apply -f instana-agent-operator-deploy.yaml
```


### Expected result

```sh
kubectl -n instana-agent get pods
```

should show two instances of the `instana-agent-operator` (see the number of `replicas` configured in `instana-agent-operator-deploy.yaml`), and two instances of `instana-agent` (one on each node in the cluster).

### Clean up

```sh
kind delete cluster
unset KUBECONFIG
```
