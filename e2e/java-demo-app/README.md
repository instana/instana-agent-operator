# Java Demo App for Kubernetes

This is a simple Java application that runs a web server for testing monitoring tools in Kubernetes.

## Building the Docker Image

To build the Docker image:

```bash
docker build -t delivery.instana.io/int-docker-agent-local/instana-agent-operator/e2e/java-demo-app:latest .
```

If you need to push to a registry:

```bash
docker push delivery.instana.io/int-docker-agent-local/instana-agent-operator/e2e/java-demo-app:latest
```

## Deploying to Kubernetes

1. Ensure to be logged in into the delivery.instana.io registry with docker

```
export NAMESPACE=selective-monitoring-no-label
./deploy.sh
```

2. Verify the deployment:

```bash
kubectl get pods -l app=java-demo-app
```

