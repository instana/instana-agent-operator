Instana Operator
================

Status
------

Runs an HTTP server on port 8080. The server provides an ASCII table with the current environment variables (useful to check which container is serving the request).

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
