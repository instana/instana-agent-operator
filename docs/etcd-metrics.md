# ETCD Metrics Configuration

The Instana Agent Operator automatically configures ETCD metrics collection for both OpenShift and vanilla Kubernetes clusters.

## OpenShift Clusters

On OpenShift clusters, the operator automatically:

1. **Discovers OpenShift ETCD resources**:
   - Checks for `etcd-metrics-ca-bundle` ConfigMap in `openshift-etcd` namespace
   - Checks for `etcd-metric-client` Secret in `openshift-etcd` namespace

2. **Copies ETCD credentials** to `instana-agent` namespace:
   - ConfigMap: `etcd-metrics-ca-bundle` (contains CA certificates)
   - Secret: `etcd-metric-client` (contains mTLS client certificates)
   - See [ADR: OpenShift ETCD Resource Copying](./adr-openshift-etcd-resource-copying.md) for architectural details

3. **Configures k8sensor Deployment** with:
   - `ETCD_METRICS_URL`: Points to OpenShift ETCD metrics endpoint
   - `ETCD_CA_FILE`: Path to mounted CA certificate
   - `ETCD_CERT_FILE`: Path to mounted client certificate
   - `ETCD_KEY_FILE`: Path to mounted client key
   - `ETCD_REQUEST_TIMEOUT`: 15s

4. **Handles certificate rotation**:
   - Tracks source `ResourceVersion` in annotations
   - Updates copied resources when OpenShift rotates certificates
   - Synchronizes on each reconcile (~10-60 second propagation delay)

5. **Automatic cleanup**:
   - Removes copied resources if source resources are deleted
   - Removes copied resources when InstanaAgent CR is deleted

**Note:** The 15s value for `ETCD_REQUEST_TIMEOUT` comes from testing ETCD request-round-trip times during our internal cluster benchmarks.
For single-datacenter setups it is intentionally conservative to avoid noisy retries during leader changes.
For inter-continental clusters (e.g., cross-Pacific) it is still below the upper bound suggested in the [ETCD tuning guide](https://etcd.io/docs/v3.4/tuning/)

### Why Resource Copying?

Kubernetes does not support cross-namespace volume mounts. Since k8sensor runs in `instana-agent` namespace but ETCD credentials exist in `openshift-etcd` namespace, the operator copies these resources during reconciliation. This approach:
- ✅ Maintains namespace isolation and security
- ✅ Gives k8sensor only local namespace permissions
- ✅ Leverages operator's existing cluster-level permissions
- ✅ Handles certificate rotation automatically

## Vanilla Kubernetes Clusters

On non-OpenShift clusters, the operator will automatically discover ETCD endpoints if:

1. A Service exists in the `kube-system` namespace with label `component=etcd`
2. The Service has a port named `metrics`

If no such labeled Service, the operator will try to find a Service named `etcd` or `etcd-metrics`.

To expose ETCD metrics in your cluster, create a Service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: etcd-metrics
  namespace: kube-system
  labels:
    component: etcd
spec:
  ports:
  - name: metrics
    port: 2379
    targetPort: 2379
  selector:
    component: etcd
```

## Environment Variables

The operator automatically sets these environment variables:

- `ETCD_TARGETS`: Comma-separated list of ETCD metrics endpoints (vanilla K8s)
- `ETCD_CA_FILE`: Path to the CA certificate for ETCD TLS
- `ETCD_METRICS_URL`: Direct URL to ETCD metrics (OpenShift)
- `ETCD_REQUEST_TIMEOUT`: Timeout for ETCD requests (default: 15s)

## Troubleshooting

### Common Issues

- **No ETCD metrics appearing**: Ensure the ETCD service exists in `kube-system` namespace with proper labels
- **TLS connection errors**: Verify CA certificates are properly mounted
- **Timeout errors**: Check network connectivity and consider adjusting `ETCD_REQUEST_TIMEOUT`

### Debugging

To verify ETCD configuration:

```bash
# Check if ETCD service exists
kubectl get svc -n kube-system -l component=etcd

# Verify operator logs
kubectl logs -n instana-agent deployment/instana-agent-operator

# Check agent environment variables
kubectl describe pod -n instana-agent -l app.kubernetes.io/name=instana-agent
```