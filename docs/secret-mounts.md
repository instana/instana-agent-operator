# Secret Mounts Feature

## Overview

The Secret Mounts feature improves security by mounting sensitive information as files instead of exposing them as environment variables in Kubernetes pods. This is a security best practice that prevents credentials from being visible during command line debugging or in process listings.

## How It Works

When enabled (default), the Instana Agent Operator:

1. Creates a secrets directory at `/opt/instana/agent/etc/instana/secrets/`
2. Mounts sensitive information as files in this directory
3. Skips setting sensitive environment variables
4. Configures the agent to read secrets from files when available

## Configuration

The feature is controlled by the `useSecretMounts` flag at the top level of the spec:

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  useSecretMounts: true  # Set to false to disable secret mounts
  agent:
    key: <your-agent-key>
    endpointHost: <your-endpoint-host>
    endpointPort: <your-endpoint-port>
```

The same configuration applies to `InstanaAgentRemote`:

```yaml
apiVersion: instana.io/v1
kind: InstanaAgentRemote
metadata:
  name: remote-agent
  namespace: instana-agent
spec:
  useSecretMounts: true  # Set to false to disable secret mounts
  agent:
    key: <your-agent-key>
    endpointHost: <your-endpoint-host>
    endpointPort: <your-endpoint-port>
```

### Default Behavior

- The feature is **enabled by default** (useSecretMounts: true)
- No action is required to use this feature in new deployments

### Disabling the Feature

If you need to revert to the previous behavior (using environment variables for secrets):

```yaml
spec:
  useSecretMounts: false
```

## Secret File Paths

When the feature is enabled, the following secrets are mounted as files:

| Secret | File Path |
|--------|-----------|
| Agent Key | `/opt/instana/agent/etc/instana/secrets/INSTANA_AGENT_KEY` |
| Download Key | `/opt/instana/agent/etc/instana/secrets/INSTANA_DOWNLOAD_KEY` |
| Proxy User | `/opt/instana/agent/etc/instana/secrets/INSTANA_AGENT_PROXY_USER` |
| Proxy Password | `/opt/instana/agent/etc/instana/secrets/INSTANA_AGENT_PROXY_PASSWORD` |
| HTTPS Proxy | `/opt/instana/agent/etc/instana/secrets/HTTPS_PROXY` |
| Release Repo Mirror Username | `/opt/instana/agent/etc/instana/secrets/AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME` |
| Release Repo Mirror Password | `/opt/instana/agent/etc/instana/secrets/AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD` |
| Shared Repo Mirror Username | `/opt/instana/agent/etc/instana/secrets/INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME` |
| Shared Repo Mirror Password | `/opt/instana/agent/etc/instana/secrets/INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD` |

## Compatibility

- The feature is backward compatible. When disabled, the operator will continue to use environment variables for secrets.
- The agent entrypoint scripts have been updated to check for secret files first, then fall back to environment variables if files are not available.
- The k8s sensor has been updated to use file-mounted secrets when available.

## K8Sensor Implementation

For the k8sensor deployment, a specialized implementation is used:

- Only the necessary secret files (INSTANA_AGENT_KEY and HTTPS_PROXY) are mounted in the k8sensor deployment
- The HTTPS_PROXY secret file is only mounted if a proxy host is configured
- The k8sensor is started with the `-agent-key-file` argument to read the agent key from the mounted file
- When a proxy is configured, the `-https-proxy-file` argument is added to read the HTTPS_PROXY value from the mounted file
- This ensures that sensitive proxy credentials are not exposed in environment variables

## Security Benefits

- Prevents secrets from being exposed in environment variables
- Reduces risk of credential exposure during debugging
- Follows Kubernetes security best practices for handling sensitive information
- Improves compliance with security standards that require protection of credentials