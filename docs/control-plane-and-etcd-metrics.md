# Control Plane and ETCD Metrics Configuration

This document covers configuration for the most common control plane metrics (kube-apiserver, kube-controller-manager, kube-scheduler)
and the optional ETCD metrics collection across different Kubernetes distributions.

## Quick Navigation
- [OpenShift Clusters](#openshift-clusters) - Automatic ETCD discovery
- [K3s Clusters](#k3s-clusters) - Control plane and optional ETCD
- [RKE2 Clusters](#rke2-clusters) - Unified configuration
- [Vanilla Kubernetes](#vanilla-kubernetes-clusters) - ETCD service setup
- [Environment Variables](#environment-variables)
- [Troubleshooting](#troubleshooting)

## OpenShift Clusters

On OpenShift clusters, the operator automatically configures ETCD metrics collection:

1. **Discovers OpenShift ETCD resources**:
   - Checks for `etcd-metrics-ca-bundle` ConfigMap in `openshift-etcd` namespace
   - Checks for `etcd-metric-client` Secret in `openshift-etcd` namespace

2. **Copies ETCD credentials** to `instana-agent` namespace:
   - ConfigMap: `etcd-metrics-ca-bundle` (contains CA certificates)
   - Secret: `etcd-metric-client` (contains mTLS client certificates) <!-- pragma: allowlist secret -->
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

## K3s Clusters

### Control Plane Metrics

On K3s, depending on the exact K3s version by default the `kube-apiserver`, `kube-controller-manager` and `kube-scheduler`,
ports might be tied to the localhost `127.0.0.1` only which is insufficient for accessing it from within the cluster.
Ensure that your `k3s-server` processes are started with the following arguments:

```bash
--kube-apiserver-arg=bind-address=0.0.0.0 \                                                                                                                                                                                                                                                   
--kube-controller-manager-arg=bind-address=0.0.0.0 \                                                                                                                                                                                                                                          
--kube-scheduler-arg=bind-address=0.0.0.0 \
```

Furthermore the `kube-controller-manager` and `kube-scheduler` services from the `kube-system` might be missing by default.
Create these services by following the patterns below for [K3s ETCD Metrics](#etcd-metrics-optional), only change these parameters:
`etcd-metrics`, `etcd`, `2381`, `http` to:
1. `kube-controller-manager`, `kube-controller-manager`, `10257`, `https`
2. `kube-scheduler`, `kube-scheduler`, `10259`, `https`

### ETCD Metrics (Optional)

On K3s, depending on the initial cluster setup, ETCD might not be available at all, because K3s can operate with SQL databases instead (e.g. PostgreSQL, MySQL, MariaDB).
Only if you have explicitly [configured K3s for HA ETCD](https://docs.k3s.io/datastore/ha-embedded), and also enabled the etcd metric exposure with `--etcd-expose-metrics=true`,
only then can the ETCD metrics be collected.

1. By default the exposed ETCD metrics are available on port `2381`, ensure that they are indeed available:
```bash
 kubectl get --server http://<YOUR_SERVERS_FQDN_OR_IP_HERE>:2381 --raw /metrics | grep etcd_server_is_leader
```

2. Get a list of `InternalIP` for nodes with ETCD role:
```bash
kubectl get nodes -l node-role.kubernetes.io/etcd=true -o jsonpath='{range .items[*]}{.status.addresses[?(@.type=="InternalIP")].address}{"\n"}{end}'
```

3. Use the following service template, fill in the IPs from the previous steps, and create the service:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: etcd-metrics
  namespace: kube-system
  labels:
    component: etcd
    provider: kubernetes
spec:
  type: ClusterIP
  ports:
    # This has to be called `metrics` verbatim in order to be auto discovered by the operator,
    # also this should match with the name of the endpointslice port defined below in order to
    # reach a functional kubernetes service.
    - name: metrics
      protocol: TCP
      port: 2381       # Internal service port
      targetPort: 2381 # Port on the selected nodes
      # Do not use a selector here! We are abstracting an out of cluster backend here, see:
      # https://kubernetes.io/docs/concepts/services-networking/service/#services-without-selectors
---
apiVersion: discovery.k8s.io/v1
kind: EndpointSlice
metadata:
  name: etcd-metrics
  namespace: kube-system
  labels:
    kubernetes.io/service-name: etcd-metrics # The EndpointSlice is linked to the service by mathcing this.
    endpointslice.kubernetes.io/managed-by: cluster-admins
addressType: IPv4
ports:
  - name: metrics # This should match with the name of the service port defined above
    appProtocol: http
    protocol: TCP
    port: 2381
endpoints:
  - addresses:
    - "<FIRST_IP_HERE>"
  - addresses:
    - "<SECOND_IP_HERE>"
  - addresses:
    - "<THIRD_IP_HERE>"
```

## RKE2 Clusters

On RKE2, depending on the exact RKE2 version by default the `kube-apiserver`, `kube-controller-manager`, `kube-scheduler`, and ETCD `listen-metrics-url` are tied to the localhost `127.0.0.1` only which is insufficient for accessing them from within the cluster.

Ensure that your `/etc/rancher/rke2/config.yaml` file has the following:

```yaml
etcd-arg:                                                                                                                                                                                                                                                                                       
  - "listen-metrics-urls=http://0.0.0.0:2381"                                                                                                                                                                                                                                                   
kube-scheduler-arg:                                                                                                                                                                                                                                                                             
  - "bind-address=0.0.0.0"                                                                                                                                                                                                                                                                      
kube-controller-manager-arg:                                                                                                                                                                                                                                                                    
  - "bind-address=0.0.0.0"                                                                                                                                                                                                                                                                      
kube-apiserver-arg:                                                                                                                                                                                                                                                                             
  - "bind-address=0.0.0.0"
```

This should be present, either before cluster installation, or if you add it after, then make sure you restart the server nodes with:

```bash
sudo systemctl restart rke2-server.service
```

### ETCD Service Configuration

Once the configuration above is applied, follow the steps to create the `etcd-metrics` service as described in the [K3s ETCD Metrics](#etcd-metrics-optional) section.

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
- **k8sensor pod fails with "configmap 'etcd-ca-bundle' not found"**: This occurs on OpenShift when ETCD monitoring resources are missing from the `openshift-etcd` namespace. The operator will log "OpenShift ETCD CA bundle not found, ETCD monitoring will be disabled" but the pod should start successfully without ETCD monitoring. If the pod fails to start, this indicates the source ETCD resources (`etcd-metrics-ca-bundle` ConfigMap and `etcd-metric-client` Secret) need to be restored in the `openshift-etcd` namespace, or you can create empty placeholder resources in the `instana-agent` namespace as a temporary workaround.

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
