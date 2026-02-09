# Agent Deployment and Scheduling

This document explains where Instana agents are deployed in a Kubernetes cluster and the conditions that determine agent pod scheduling.

## Overview

The Instana Agent Operator deploys agents as a DaemonSet, which means Kubernetes attempts to run one agent pod on each node in the cluster. However, several factors determine whether an agent pod will actually be scheduled on a specific node.

## Default Deployment Behavior

### Simple Deployments

In basic configurations without custom scheduling rules, agents are deployed on **worker nodes by default**. Master/control plane nodes typically have taints that prevent regular workloads (including agents) from being scheduled on them.

### Host Coverage

The "host coverage" metric shown in the Kubernetes cluster dashboard represents the percentage of nodes that have an agent pod running. A coverage of less than 100% indicates that some nodes do not have agents, which can occur for several reasons explained below.

## Factors Affecting Agent Deployment

### 1. Node Taints

**Taints** are the primary mechanism that prevents agent pods from being scheduled on certain nodes. Kubernetes uses taints to mark nodes as unsuitable for certain workloads.

#### Common Taints That Block Agent Deployment

- **Master/Control Plane Nodes**: Typically have taints like:
  - `node-role.kubernetes.io/master:NoSchedule`
  - `node-role.kubernetes.io/control-plane:NoSchedule`
  
- **Infrastructure Nodes**: May have custom taints such as:
  - `node-role.kubernetes.io/infra:NoSchedule`
  
- **Specialized Nodes**: GPU nodes, high-memory nodes, or other specialized hardware often have taints like:
  - `nvidia.com/gpu:NoSchedule`
  - Custom taints defined by cluster administrators

#### How Taints Work

By default, the agent DaemonSet does **not** include tolerations for these taints. This means:
- Agents **will** be scheduled on untainted worker nodes
- Agents **will not** be scheduled on tainted nodes (master, infra, GPU, etc.)

### 2. Node Selectors

Node selectors allow you to target specific nodes based on labels. If configured, agents will only be scheduled on nodes matching the selector criteria.

### 3. Node Affinity Rules

Affinity rules provide more sophisticated node selection logic, allowing you to express preferences or requirements for node characteristics.

### 4. Resource Availability

Even if a node is eligible for agent deployment, the agent pod may not be scheduled if:
- The node lacks sufficient CPU or memory resources
- Resource quotas or limits prevent pod creation

## Configuring Agent Deployment

### Adding Tolerations

To deploy agents on tainted nodes, you must add tolerations to your [`InstanaAgent`](../api/v1/instanaagent_types.go) custom resource:

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  agent:
    key: <your-agent-key>
    endpointHost: <your-endpoint>
    endpointPort: "443"
    pod:
      tolerations:
      # Tolerate master/control plane nodes
      - key: node-role.kubernetes.io/master
        operator: Exists
        effect: NoSchedule
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule
      # Tolerate infrastructure nodes
      - key: node-role.kubernetes.io/infra
        operator: Exists
        effect: NoSchedule
      # Tolerate GPU nodes
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
```

**Important**: Adding tolerations allows agents to be scheduled on tainted nodes but does not guarantee deployment. The node must still have sufficient resources and meet any other scheduling constraints.

### Using Node Selectors

To deploy agents only on specific nodes:

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  agent:
    key: <your-agent-key>
    endpointHost: <your-endpoint>
    endpointPort: "443"
    pod:
      nodeSelector:
        node-type: monitoring-enabled
```

### Using Node Affinity

For more complex scheduling requirements:

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  agent:
    key: <your-agent-key>
    endpointHost: <your-endpoint>
    endpointPort: "443"
    pod:
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: node-role.kubernetes.io/worker
                operator: Exists
```

## Multi-Zone Deployments

For complex cluster topologies with different node pools, you can deploy separate agent DaemonSets per zone with zone-specific scheduling rules. See the [multi-zone configuration example](../config/samples/instana_v1_multizone_instanaagent.yaml).

### Zone-Based Configuration

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  agent:
    key: <your-agent-key>
    endpointHost: <your-endpoint>
    endpointPort: "443"
  zones:
  - name: gpu-pool
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: node-type
              operator: In
              values:
              - gpu
    tolerations:
    - key: nvidia.com/gpu
      operator: Exists
      effect: NoSchedule
  - name: general-pool
    affinity:
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
          - matchExpressions:
            - key: node-type
              operator: In
              values:
              - general
```

When zones are configured:
- A separate DaemonSet is created for each zone (e.g., `instana-agent-gpu-pool`, `instana-agent-general-pool`)
- Each DaemonSet uses the affinity and toleration rules defined for its zone
- This allows fine-grained control over which agents run on which nodes

## Troubleshooting Low Host Coverage

If your host coverage is below 100%, follow these steps:

### 1. Identify Nodes Without Agents

```bash
# List all nodes
kubectl get nodes

# Check which nodes have agent pods
kubectl get pods -n instana-agent -o wide

# Compare to find nodes without agents
```

### 2. Check Node Taints

```bash
# Inspect taints on a specific node
kubectl describe node <node-name> | grep -A 5 Taints
```

### 3. Check Agent Pod Events

```bash
# View events for the agent DaemonSet
kubectl describe daemonset instana-agent -n instana-agent

# Check for scheduling failures
kubectl get events -n instana-agent --sort-by='.lastTimestamp'
```

### 4. Common Issues and Solutions

| Issue | Symptom | Solution |
|-------|---------|----------|
| **Tainted nodes** | Nodes have taints, no tolerations configured | Add appropriate tolerations to `spec.agent.pod.tolerations` |
| **Node selector mismatch** | Nodes don't match selector | Update `spec.agent.pod.nodeSelector` or add labels to nodes |
| **Resource constraints** | Insufficient CPU/memory | Increase node resources or adjust `spec.agent.pod.requests` |
| **Affinity rules** | Nodes don't match affinity | Update `spec.agent.pod.affinity` rules |

### 5. Verify Agent Configuration

```bash
# Check the current agent configuration
kubectl get instanaagent instana-agent -n instana-agent -o yaml

# Verify DaemonSet configuration
kubectl get daemonset instana-agent -n instana-agent -o yaml
```

## Best Practices

1. **Start Simple**: Begin with default configuration and add tolerations only as needed
2. **Document Taints**: Maintain documentation of custom taints in your cluster
3. **Monitor Coverage**: Regularly check host coverage metrics and investigate drops
4. **Test Changes**: Test scheduling configuration changes in non-production environments first
5. **Use Zones for Complexity**: For clusters with diverse node types, use zone-based configuration rather than complex global rules

## Related Documentation

- [Multi-Zone Configuration Example](../config/samples/instana_v1_multizone_instanaagent.yaml)
- [Kubernetes Taints and Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
- [Kubernetes Node Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity)