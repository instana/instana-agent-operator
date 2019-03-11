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

Test
----

1.  Get the service's [Cluster IP](https://kubernetes.io/docs/concepts/services-networking/service/):
    ```bash
    export IP=$(kubectl get service instana-operator -o=jsonpath='{.spec.clusterIP}')
    ```
2.  Access the pods through the service's Cluster IP:
    ```bash
    curl $IP
    ```
