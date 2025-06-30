# Environment Variables in Instana Agent Operator

The Instana Agent Operator supports two ways to define environment variables for the agent pods:

## 1. Legacy Method: Simple Key-Value Pairs

The original method uses a simple key-value map in the `agent.env` field:

```yaml
spec:
  agent:
    env:
      INSTANA_AGENT_TAGS: dev,test
      CUSTOM_ENV_VAR: custom-value
```

This method is simple but limited to string values only.

## 2. Enhanced Method: Full Kubernetes EnvVar Support

The new method uses the standard Kubernetes EnvVar structure in the `agent.pod.env` field, which provides more flexibility:

```yaml
spec:
  agent:
    pod:
      env:
        # Simple value
        - name: INSTANA_AGENT_TAGS
          value: "kubernetes,production,custom"
        
        # From field reference
        - name: MY_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        
        # From secret
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: db-password
              optional: true
        
        # From ConfigMap
        - name: APP_CONFIG
          valueFrom:
            configMapKeyRef:
              name: app-config
              key: config.json
              optional: true
```

## Supported ValueFrom Sources

The enhanced method supports all standard Kubernetes environment variable sources:

1. **Field References**: Access pod metadata and status fields
   ```yaml
   valueFrom:
     fieldRef:
       fieldPath: metadata.name
   ```

2. **Resource Field References**: Access container resource limits and requests
   ```yaml
   valueFrom:
     resourceFieldRef:
       containerName: instana-agent
       resource: requests.cpu
       divisor: 1m
   ```

3. **ConfigMap References**: Get values from ConfigMaps
   ```yaml
   valueFrom:
     configMapKeyRef:
       name: my-config
       key: my-key
       optional: true
   ```

4. **Secret References**: Get values from Secrets
   ```yaml
   valueFrom:
     secretKeyRef:
       name: my-secret
       key: my-key
       optional: true
   ```

## Precedence

If both `agent.env` and `agent.pod.env` are defined, both will be applied to the agent container. In case of duplicate environment variable names, the values from `agent.pod.env` will take precedence.

## Example

See the complete example in [config/samples/instana_v1_env_vars_example.yaml](../config/samples/instana_v1_env_vars_example.yaml).

// Made with Bob
