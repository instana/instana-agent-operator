## Configuration

### Custom Resource Values

The Instana Agent Custom resource supports the following values:

| Parameter | Description |
| --- | --- |
| `agent.key` | Instana agent key |
| `agent.endpoint` | Reporting endpoint from agent management page |
| `agent.endpoint.port` | Reporting port `"443"` (wrapped in quotes) |
| `agent.zone.name` | Name of zone to display for things discovered by these agents |
| `cluster.name` | Name of this Kubernetes cluster to display in Instana |
| `agent.env` | (optional) Can be used to specify environment variables for the agent, for instance, proxy configuration. See possible environment values [here](https://docs.instana.io/quick_start/agent_setup/container/docker/) |
| `agent.image` | (optional) Can be used to override the agent image (defaults to `instana/agent:latest`) |
| `config.files` | (optional) Additional files to mount for configuration. Each entry in this object is mounted in the agent as a file in `/root/<key>` |
| `agent.downloadKey` | (optional) Download key for agent artifacts (usually not required) |
| `agent.cpuReq` | (optional) CPU requests for agent in CPU cores |
| `agent.cpuLimit` | (optional) CPU limits for agent in CPU cores |
| `agent.memReq` | (optional) Memory requests for agent in Mi |
| `agent.memLimit` | (optional) Memory limits for agent in Mi |

### Environment variables

Currently, it is also possible to configure the `agent.image` via the `RELATED_IMAGE_INSTANA_AGENT` environment variable specified in the [`instana-agent-operator` deployment](../deploy/instana-agent-operator.yaml#L197):

```yaml
---
        env:
        - name: "RELATED_IMAGE_INSTANA_AGENT",
          value: "instana/agent:latest"
```

The operator first looks at the `agent.image` parameter in the CRD to determine the agent image. If this is null, it then checks the environment variable above. Finally if both of these are null, it usess the default `instana/agent:latest`.