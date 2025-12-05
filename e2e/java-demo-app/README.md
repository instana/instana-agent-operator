# Java Demo App for Kubernetes

This is a simple Java application that runs a web server for testing monitoring tools in Kubernetes.

## Building the Docker Image

To build the Docker image:

```bash
docker build -t java-demo-app:latest .
```

If you need to push to a registry:

```bash
skopeo copy --all --dest-username iamapikey --dest-password "xxx" docker-daemon:java-demo-app:latest docker://icr.io/instana-int/int-docker-agent-local/instana-agent-operator/e2e/java-demo-app:latest
```

## Deploying to Kubernetes

1. Deploy the app
```
export NAMESPACE=selective-monitoring-no-label
kubectl create ns ${NAMESPACE}
kubectl -n ${NAMESPACE} create secret docker-registry icr-io-pull-secret \
    --docker-server=icr.io \
    --docker-username=iamapikey \
    --docker-password=xxx
kubectl apply -f deployment.yaml -n ${NAMESPACE}
```

2. Verify the deployment:

```bash
kubectl get pods -l app=java-demo-app
```

