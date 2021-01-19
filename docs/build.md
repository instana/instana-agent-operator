Building the Instana Agent Operator from Source
===============================================

The following command will build the `instana/instana-agent-operator` Docker image locally:

```bash
./mvnw -C -B clean verify
docker build -f src/main/docker/Dockerfile.jvm -t instana/instana-agent-operator .
```

To build the Docker image with GraalVM native image, use the following command:

> Note: The native image does not work yet because of [https://github.com/quarkusio/quarkus/issues/3077](https://github.com/quarkusio/quarkus/issues/3077)

```bash
./mvnw -C -B clean verify -Pnative -Dnative-image.docker-build=true
docker build -f src/main/docker/Dockerfile.native -t instana/instana-agent-operator .
```
