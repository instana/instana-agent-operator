# ETCD Metrics Configuration

The Instana Agent Operator automatically configures ETCD metrics collection for both OpenShift and vanilla Kubernetes clusters.

## OpenShift Clusters

On OpenShift clusters, the operator automatically:

1. Creates a ConfigMap with the `service.beta.openshift.io/inject-cabundle: "true"` annotation
2. Mounts the injected CA certificate at `/etc/service-ca/service-ca.crt`
3. Sets `ETCD_METRICS_URL` to point to the OpenShift etcd metrics endpoint
4. Sets `ETCD_CA_FILE` to the mounted certificate path
5. Sets `ETCD_REQUEST_TIMEOUT` to 15s

**Note:** The 15s value for `ETCD_REQUEST_TIMEOUT` comes from testing ETCD request-round-trip times during our internal cluster benchmarks.
For single-datacenter setups it is intentionally conservative to avoid noisy retries during leader changes.
For inter-continental clusters (e.g., cross-Pacific) it is still below the upper bound suggested in the [ETCD tuning guide](https://etcd.io/docs/v3.4/tuning/)

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