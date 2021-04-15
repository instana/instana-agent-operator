Testing the Instana Agent Operator with Kind
============================================

One of the main features of the `instana-agent-operator` is to do leader election. In order to test this manually, we need a Kubernetes cluster where we can _destroy nodes_, so that we can observe if a new leader is elected in that case.

This page describes how to set up a local Kubernetes cluster with [Kind](https://kind.sigs.k8s.io/) and test the `instana-agent-operator` in that cluster.

Kind (**K**ubernetes **in** **D**ocker) will set up a local cluster where all nodes and the master run in Docker containers.

Set up a local Kubernetes Cluster with Kind
-------------------------------------------

Install [kind](https://kind.sigs.k8s.io/) (a single executable that can be downloaded from the [Github release page](https://github.com/kubernetes-sigs/kind/releases)) and run the following commands to create the cluster:

```sh
./e2e-testing/with-kind/create-cluster.sh
```

You should see a cluster with one control-pane and two worker nodes up and running.

This script will also do the following:
- Build the `instana/instana-agent-operator` Docker image locally and load it into the local Kind cluster
- Pull the latest `instana/agent` Docker image locally and load it into the local Kind cluster

Install the Operator
--------------------

Follow the steps described in [Install Operator Manually](https://www.instana.com/docs/setup_and_manage/host_agent/on/kubernetes/#install-operator-manually).

If your changes include the `instana-agent-operator.yaml`, you'll need to generate a new version of that file:
```sh
./olm/create-artifacts.sh dev olm
```
Otherwise, you can download `instana-agent-operator.yaml` file from the latest [GitHub release](https://github.com/instana/instana-agent-operator/releases)

Then in either case, change the `imagePullPolicy` for the `Deployment` from `Always` to `IfNotPresent`.
This will ensure that it uses the locally built `instana/instana-agent-operator` image instead of pulling from the remote registry.

Expected result
---------------

```sh
kubectl -n instana-agent get pods
```

This should show one instance of the `instana-agent-operator` (see the number of `replicas` configured in `olm/operator-resources/instana-agent-operator.yaml`), and two instances of `instana-agent` (one on each worker node in the cluster).

Delete a node
-------------

Delete one of the worker nodes that is running the `instana-agent` leader pod. The operator should reassign leadership to another agent pod.
You should see the reassignment if you tail the logs for the operator pod.

Clean up
--------

```sh
./e2e-testing/with-kind/delete-cluster.sh
```
