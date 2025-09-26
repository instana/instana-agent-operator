# Understanding externalKeysSecret in Instana Agent Operator

This document explains how the `externalKeysSecret` mechanism works in the Instana Agent Operator, comparing it to inline key notation, and detailing how it functions with additionalBackends. It also covers how secrets are used in different deployment scenarios.

## Table of Contents

1. [Key Configuration Methods](#key-configuration-methods)
2. [externalKeysSecret vs. Inline Keys](#externalkeyssecret-vs-inline-keys)
3. [Working with additionalBackends](#working-with-additionalbackends)
4. [Secret Mounting Mechanisms](#secret-mounting-mechanisms)
5. [Deployment Scenarios](#deployment-scenarios)
   - [Agent DaemonSet](#agent-daemonset)
   - [Remote Agent Deployment](#remote-agent-deployment)
   - [K8Sensor Deployment](#k8sensor-deployment)

## Key Configuration Methods

The Instana Agent Operator supports two methods for providing agent keys:

1. **Inline Keys**: Directly specifying keys in the Custom Resource (CR)
   ```yaml
   agent:
     key: your-agent-key
     downloadKey: your-download-key
   ```

2. **External Keys Secret**: Referencing a pre-created Kubernetes Secret
   ```yaml
   agent:
     keysSecret: instana-agent-key
   ```

## externalKeysSecret vs. Inline Keys

### Inline Keys Approach

When using inline keys, the agent key and download key are specified directly in the CR:

```yaml
agent:
  key: your-agent-key
  downloadKey: your-download-key
  endpointHost: ingress-red-saas.instana.io
  endpointPort: "443"
```

With this approach:
- Keys are stored as part of the CR definition
- The operator creates a Secret containing these keys
- Less secure as keys are visible in the CR YAML

### externalKeysSecret Approach

When using external keys secret, you create a separate Kubernetes Secret and reference it:

1. Create a Secret:
   ```yaml
   apiVersion: v1
   stringData:
     key: your-agent-key
     downloadKey: your-download-key
   kind: Secret
   metadata:
     name: instana-agent-key
     namespace: instana-agent
   type: Opaque
   ```

2. Reference it in the CR:
   ```yaml
   agent:
     keysSecret: instana-agent-key
     endpointHost: ingress-red-saas.instana.io
     endpointPort: "443"
   ```

Benefits of this approach:
- Keys are managed separately from the CR
- Better security practices (separation of configuration from secrets)
- Easier to rotate keys without modifying the CR
- Compatible with external secret management systems

## Working with additionalBackends

The Instana Agent can report to multiple backends simultaneously. When using `additionalBackends` with `externalKeysSecret`, the keys for all backends must be stored in the same Secret.

### Using Inline Keys with additionalBackends

```yaml
agent:
  key: primary-backend-key
  endpointHost: first-backend.instana.io
  endpointPort: "443"
  additionalBackends:
  - endpointHost: second-backend.instana.io
    endpointPort: "443"
    key: secondary-backend-key
```

### Using externalKeysSecret with additionalBackends

1. Create a Secret with keys for all backends:
   ```yaml
   apiVersion: v1
   stringData:
     key: primary-backend-key
     key-1: secondary-backend-key
   kind: Secret
   metadata:
     name: instana-agent-key
     namespace: instana-agent
   type: Opaque
   ```

2. Reference it in the CR:
   ```yaml
   agent:
     keysSecret: instana-agent-key
     endpointHost: first-backend.instana.io
     endpointPort: "443"
     additionalBackends:
     - endpointHost: second-backend.instana.io
       endpointPort: "443"
   ```

The operator will:
- Use `key` for the primary backend
- Use `key-1`, `key-2`, etc. for additional backends (indexed from 1)

## Secret Mounting Mechanisms

The Instana Agent Operator supports two methods for providing secrets to the agent:

1. **Environment Variables** (legacy approach)
2. **Secret File Mounts** (default, more secure approach)

This is controlled by the `useSecretMounts` flag in the CR:

```yaml
spec:
  useSecretMounts: true  # Default is true
```

### Environment Variables Approach

When `useSecretMounts: false`, secrets are passed as environment variables:
- `AGENT_KEY` environment variable contains the agent key
- Less secure as secrets are exposed in the process environment

### Secret File Mounts Approach

When `useSecretMounts: true` (default):
- Secrets are mounted as files in `/opt/instana/agent/etc/instana/secrets/`
- Agent key is mounted as `/opt/instana/agent/etc/instana/secrets/INSTANA_AGENT_KEY`
- More secure as secrets are not exposed in the environment

## Deployment Scenarios

### Agent DaemonSet

In the agent DaemonSet scenario:
- One DaemonSet is deployed with pods on each node
- Each pod can connect to multiple backends
- When using `externalKeysSecret`, the operator:
  1. Reads the keys from the referenced Secret
  2. Creates configuration for each backend
  3. Mounts the keys as files (if `useSecretMounts: true`)
  4. Each pod has all backend configurations

The agent DaemonSet will have multiple backend configuration files mounted into a single pod, allowing it to report to multiple backends simultaneously.

### Remote Agent Deployment

For remote agents:
- Deployed as a Deployment (not DaemonSet)
- Uses the same `keysSecret` mechanism as the regular agent
- Configuration is similar to the agent DaemonSet
- Each pod can connect to multiple backends

### K8Sensor Deployment

The K8Sensor deployment is different:
- **One deployment per backend** (unlike agent DaemonSet)
- Each deployment is dedicated to a single backend
- When using `externalKeysSecret`:
  1. The operator reads the keys from the referenced Secret
  2. Creates separate deployments for each backend
  3. Each deployment gets only its specific backend key

This is a key architectural difference: while the agent DaemonSet has multiple backend configurations in a single pod, the K8Sensor uses separate deployments for each backend.

For example, with two backends:
- Two K8Sensor deployments will be created
- First deployment connects to the primary backend using `key` from the Secret
- Second deployment connects to the secondary backend using `key-1` from the Secret

This design allows for better resource allocation and isolation between backend connections for the K8Sensor component.