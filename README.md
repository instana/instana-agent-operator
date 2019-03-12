Instana Operator
================

Status
------

This is the initial commit. `instana-operator` can be deployed on Kubernetes and runs a small HTTP server on port 8080 showing the status of the leader election.

The next step is to replace the current leader election sidecar with the operator.

Should we support more than one `instana-operator` instance? If there are multiple `instana-operator` Pods running, they need to elect an `instana-operator` leader among themselves. The `instana-operator` leader determines which Intana agent will be the agent leader. A first implementation of leader election among `instana-operator` instances can be found in `LeaderElector.java`.

Build
-----

Requires OpenJDK.

**Option 1:** Build the Docker image manually:

```sh
mvn package
docker build -t instana/instana-operator .
```

**Option 2:** Build using the [Docker maven plugin](https://dmp.fabric8.io/):

```sh
mvn package docker:build
```

Deploy
------

Assumes that the Docker image `instana/instana-operator:latest` is available in the local repository on all Kubernetes nodes (see `imagePullPolicy: Never` in `instana-operator.yaml`).

```
kubectl apply -f instana-operator.yaml
```

Manualy Testing the Leader Election
-----------------------------------

The best way to test the leader election is to deploy individual Pods instead of a Deployment. That way, you can shut down single Pods without having Kubernetes restarting them.

* Create the `ServiceAccount`, `ClusterRole`, and `ClusterRoleBinding` as defined in `instana-operator.yaml`, but not the `Deployment` and `Service`.
* Start three Pods as follows:
  ```
  kubectl run --generator=run-pod/v1 instana-operator-1 --image=instana/instana-operator --image-pull-policy=Never --env="POD_NAME=instana-operator-1" --serviceaccount=instana-operator
  kubectl run --generator=run-pod/v1 instana-operator-2 --image=instana/instana-operator --image-pull-policy=Never --env="POD_NAME=instana-operator-2" --serviceaccount=instana-operator
  kubectl run --generator=run-pod/v1 instana-operator-3 --image=instana/instana-operator --image-pull-policy=Never --env="POD_NAME=instana-operator-3" --serviceaccount=instana-operator
  ```
* Leader election uses a `ConfigMap` as lock with the current leader as owner reference. Check that as follows:
  ```
  kubectl get configmaps instana-operator-leader-lock -o yaml
  ```
* If you delete a pod (example: `kubectl delete pod instana-operator-1`), the `ConfigMap` will be deleted and a new Pod should succeed to create the `ConfigMap` and become leader after max 10 seconds.
* When the last pod is deleted, the `ConfigMap` should be gone as well.


HTTP Interface for Debugging
----------------------------

1.  Get the service's [Cluster IP](https://kubernetes.io/docs/concepts/services-networking/service/):
    ```bash
    export IP=$(kubectl get service instana-operator -o=jsonpath='{.spec.clusterIP}')
    ```
2.  Access the pods through the service's Cluster IP:
    ```bash
    curl $IP
    ```
