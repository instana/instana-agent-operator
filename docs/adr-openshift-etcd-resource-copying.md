# Architecture Decision Record: OpenShift ETCD Resource Copying Strategy

## Status
Accepted

## Context

The Instana Agent operator needs to monitor OpenShift ETCD metrics to provide comprehensive control plane observability. OpenShift ETCD exposes metrics via mTLS-protected endpoints that require:

1. **CA Certificate**: Located in `openshift-etcd/etcd-metrics-ca-bundle` ConfigMap
2. **Client Certificates**: Located in `openshift-etcd/etcd-metric-client` Secret

The k8sensor Deployment runs in the `instana-agent` namespace and needs access to these credentials to collect ETCD metrics.

### Problem

Kubernetes does not support cross-namespace volume mounts for ConfigMaps and Secrets. Pods can only mount volumes from resources in their own namespace. This creates a fundamental challenge: how can the k8sensor pod access ETCD credentials that exist in a different namespace?

## Decision

**We copy the required ETCD ConfigMap and Secret from `openshift-etcd` namespace to `instana-agent` namespace during operator reconciliation.**

The operator (not the k8sensor pod) performs this copying using its existing cluster-level permissions. The copied resources are:
- Tracked with labels and annotations for proper lifecycle management
- Synchronized on each reconcile to detect certificate rotation
- Automatically cleaned up when ETCD resources are removed

## Alternatives Considered

### 1. Cross-Namespace Volume Mounting
**Status**: Not Supported
- Kubernetes fundamentally does not allow pods to mount ConfigMaps/Secrets from other namespaces
- This is a security feature, not a limitation we can work around

### 2. Deploy k8sensor in openshift-etcd Namespace
**Status**: Rejected - Violates Namespace Isolation
- **Cons**:
  - Breaks namespace isolation and security boundaries
  - openshift-etcd is a critical system namespace
  - Mixing application workloads with system components is an anti-pattern
  - Complicates RBAC and security policies

### 3. Init Container to Fetch Credentials
**Status**: Rejected - Adds Complexity and Security Risk
- **Cons**:
  - Requires granting k8sensor ServiceAccount cross-namespace GET permissions
  - Violates principle of least privilege (pod doesn't need this access)
  - Adds container overhead and initialization complexity
  - Doesn't solve certificate rotation problem elegantly

### 4. External Secrets Management (e.g., External Secrets Operator)
**Status**: Rejected - Unnecessary Dependency
- **Cons**:
  - Adds external dependency to operator installation
  - Increases system complexity
  - Not all customers use External Secrets Operator
  - Overkill for this specific use case

### 5. Service Account Token Volume Projection with Cross-Namespace Access
**Status**: Rejected - Still Requires RBAC for Pod
- **Cons**:
  - Still requires granting k8sensor pod cross-namespace permissions
  - Doesn't simplify the RBAC model
  - Adds token management complexity

## Trade-offs

### Pros
✅ **Simple**: Leverages existing operator permissions
✅ **Secure**: k8sensor pod has no cross-namespace access
✅ **Standard Kubernetes Pattern**: Resource copying is a common operator pattern
✅ **Automated Sync**: Operator handles certificate rotation automatically
✅ **Clean Lifecycle**: Resources are garbage-collected with the InstanaAgent CR

### Cons
⚠️ **Resource Duplication**: ConfigMap and Secret exist in two namespaces
⚠️ **Eventual Consistency**: Updates propagate on next reconcile (not real-time)
⚠️ **Operator Dependency**: k8sensor cannot function without operator running

## Implementation Details

### Resource Tracking

Copied resources are annotated with:
```yaml
annotations:
  instana.io/source-namespace: "openshift-etcd"
  instana.io/source-name: "etcd-metrics-ca-bundle"
  instana.io/source-resource-version: "123456"  # For sync detection
  instana.io/instana-agent-name: "instana-agent"
```

And labeled with:
```yaml
labels:
  app.kubernetes.io/name: "instana-agent"
  app.kubernetes.io/component: "k8sensor"
  app.kubernetes.io/managed-by: "instana-agent-operator"
  instana.io/copied-from: "openshift-etcd"
```

### Synchronization Strategy

**Option A: Reconcile-Based Sync (Chosen)**
- Check source `ResourceVersion` on each reconcile
- Update copied resource if versions differ
- **Pros**: Simple, no additional watches needed
- **Cons**: ~10-60 second delay on certificate rotation (acceptable for ETCD cert rotation scenarios)

**Option B: Watch-Based Sync** (Future Enhancement)
- Add watch on openshift-etcd namespace
- Trigger reconcile on source resource changes
- **Pros**: Near real-time sync
- **Cons**: Additional watch overhead, more complex

### Cleanup Strategy

When ETCD resources don't exist or InstanaAgent is deleted:
1. Operator deletes copied ConfigMap
2. Operator deletes copied Secret
3. k8sensor Deployment automatically recreates pods without ETCD volumes
4. ETCD monitoring is disabled gracefully

### Security Model

- **Operator**: Has ClusterRole with `configmaps` and `secrets` GET/CREATE/UPDATE/DELETE across all namespaces
- **k8sensor Pod**: Has NO cross-namespace permissions. Can only read resources in `instana-agent` namespace
- **Principle of Least Privilege**: The component doing the work (k8sensor) has minimal permissions

## Consequences

### Positive
- k8sensor remains namespace-scoped with minimal permissions
- No new RBAC requirements for k8sensor ServiceAccount
- Standard Kubernetes operator pattern
- Easy to understand and maintain

### Negative
- Duplicated resources consume minimal additional etcd space (~10KB)
- Certificate rotation has eventual consistency delay (~10-60 seconds)
- Requires operator to be running for initial setup

### Neutral
- This pattern is already used by many operators (e.g., cert-manager, prometheus-operator)
- Trade-off between simplicity and real-time sync is acceptable for ETCD monitoring use case

## Validation

Certificate rotation testing confirms:
1. Operator detects ResourceVersion changes on reconcile
2. Copied resources are updated within 60 seconds (default reconcile interval)
3. k8sensor pods pick up new certificates via volume mount update
4. No metrics collection gaps observed during rotation

## References

- [Kubernetes Cross-Namespace Mounts Discussion](https://github.com/kubernetes/kubernetes/issues/40610)
- [Operator Best Practices - Resource Copying](https://sdk.operatorframework.io/docs/best-practices/best-practices/)
- [Cert-Manager CA Injection Pattern](https://cert-manager.io/docs/concepts/ca-injector/)

## Review History

- **2025-11-11**: Initial decision recorded
- **Author**: Instana Agent Team
- **Reviewers**: Staff Engineer, Security Team

---

*This ADR documents a fundamental architectural choice. Any changes to this strategy should be carefully reviewed for security and operational implications.*
