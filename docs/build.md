Building the Instana Agent Operator from Source
===============================================

The following command will build the `instana/instana-agent-operator` Docker image locally:

```bash
mvn package docker:build
```

To build the Docker image with GraalVM native image, use the following command:

```bash
mvn package -Pnative
```
