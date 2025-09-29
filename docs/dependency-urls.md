# DependencyURLs Feature

The Instana Agent Operator now supports downloading external dependency files via an init container and sharing them with the agent pod. This is useful for scenarios where you need to provide additional runtime dependencies to the agent, such as JDBC drivers or other libraries.

## How it works

When the `dependencyURLs` property is defined in the agent specification, the operator will:

1. Create an init container that downloads the files from the specified URLs
2. Store the files in a shared volume (`instanadeploy`)
3. Mount this volume to the agent container at `/opt/instana/agent/deploy`

This allows you to provide external dependencies to the agent without having to build a custom agent image.

## Usage

To use this feature, add the `dependencyURLs` property to your InstanaAgent custom resource:

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  # ... other configuration ...
  agent:
    key: your-agent-key
    endpointHost: ingress-red-saas.instana.io
    endpointPort: "443"
    dependencyURLs:
      - "https://repo1.maven.org/maven2/mysql/mysql-connector-java/8.0.28/mysql-connector-java-8.0.28.jar"
      - "https://repo1.maven.org/maven2/org/postgresql/postgresql/42.3.3/postgresql-42.3.3.jar"
    # ... other agent configuration ...
```

## Example

A complete example is available in the samples directory: `config/samples/instana_v1_instanaagent_with_dependency_urls.yaml`

## Notes

- The files will be downloaded to `/opt/instana/agent/deploy/` in the agent container
- The original filenames from the URLs will be preserved
- You can specify multiple dependency URLs, and all files will be downloaded to the same directory
- Make sure the URLs are accessible from the Kubernetes cluster where the agent is running
- For security reasons, consider using HTTPS URLs and trusted sources