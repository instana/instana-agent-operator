# Enable the OCP Internal Image Registry

Step-by-step guide to enable the built-in OpenShift image registry, expose it externally, and verify it works by pushing and deploying a test image.

**Prerequisites:** `oc` logged in as a cluster-admin, `skopeo` installed locally.

## Check the current registry state

```bash
oc get configs.imageregistry.operator.openshift.io/cluster \
  -o jsonpath='{.spec.managementState}{"\n"}'
```

- **`Managed`** → already enabled, continue to the next section.
- **`Removed`** (common on bare-metal / vSphere / Nutanix) → run the patch below, then configure storage before continuing.

```bash
# Switch from Removed to Managed
oc patch configs.imageregistry.operator.openshift.io/cluster \
  --type merge -p '{"spec":{"managementState":"Managed"}}'
```

> **Storage note (bare-metal / vSphere):** The registry needs a PVC. If none exists,
> create one and reference it:
> ```bash
> oc patch configs.imageregistry.operator.openshift.io/cluster --type merge -p \
>   '{"spec":{"storage":{"pvc":{"claim":""}}}}'
> ```
> This creates a 100 Gi PVC automatically. Alternatively point at an existing claim by name.

Confirm the registry pod comes up:

```bash
oc get pods -n openshift-image-registry -l docker-registry=default
# Expected: 1/1 Running
```

## Enable the external route

By default the registry is only reachable inside the cluster. Enabling `defaultRoute`
creates an OpenShift Route with `reencrypt` TLS termination.

```bash
oc patch configs.imageregistry.operator.openshift.io/cluster \
  --type merge -p '{"spec":{"defaultRoute":true}}'
```

Retrieve the hostname (may take a few seconds):

```bash
REGISTRY=$(oc get route default-route \
  -n openshift-image-registry \
  -o jsonpath='{.spec.host}')
echo $REGISTRY
# e.g. default-route-openshift-image-registry.apps.<cluster>.example.com
```

## Obtain credentials

Capture your token for reuse in the copy commands below:

```bash
TOKEN=$(oc whoami -t)
```

## Copy an image into the internal registry

`skopeo copy` transfers directly from source to destination without a local
intermediate layer — ideal for air-gapped pre-population from a bastion host.

```bash
# Create (or reuse) the target namespace
oc new-project <your-namespace>

# Log in to the internal registry first (stores credentials in a temp auth file)
skopeo login \
  --tls-verify=false \
  --username kubeadmin --password "$TOKEN" \
  "$REGISTRY"

# Copy from a public registry straight into the OCP internal registry
# Pattern: docker://$REGISTRY/<namespace>/<name>:<tag>
skopeo copy \
  --dest-tls-verify=false \
  docker://<source-image> \
  docker://$REGISTRY/<your-namespace>/<image-name>:<tag>
```

> **`--dest-tls-verify=false`** skips verification of the cluster's self-signed
> ingress certificate. To drop this flag permanently, add the cluster CA to the
> system trust store (`/etc/pki/ca-trust/source/anchors/` on RHEL, then
> `update-ca-trust`).

**Copy from a local OCI directory** (useful when the bastion itself has no
internet access and images were transferred as tarballs):

```bash
# On a machine with internet access — save to OCI layout
skopeo copy docker://<source-image> oci:<local-dir>:<tag>

# Transfer <local-dir> to the bastion (rsync, USB, etc.), then push:
skopeo copy \
  --dest-tls-verify=false \
  oci:<local-dir>:<tag> \
  docker://$REGISTRY/<your-namespace>/<image-name>:<tag>
```

Verify the ImageStream was created inside the cluster:

```bash
oc get imagestream -n <your-namespace>
```

## Deploy using the internal registry

Reference the **in-cluster service hostname** so the kubelet pulls without
leaving the cluster network:

```bash
oc new-app \
  --image=image-registry.openshift-image-registry.svc:5000/<your-namespace>/<image-name>:<tag> \
  --name=<app-name> \
  -n <your-namespace>

oc rollout status deployment/<app-name> -n <your-namespace>
```

## Smoke test (optional)

```bash
oc expose service/<app-name> -n <your-namespace>

curl -s -o /dev/null -w "%{http_code}" \
  http://$(oc get route <app-name> -n <your-namespace> -o jsonpath='{.spec.host}')
# Expected: 200
```

## Quick reference

```
EXTERNAL push:  $REGISTRY/<namespace>/<image>:<tag>
INTERNAL pull:  image-registry.openshift-image-registry.svc:5000/<namespace>/<image>:<tag>
```

| Hostname | Used by |
|----------|---------|
| `default-route-openshift-image-registry.apps.*` | `skopeo copy` from your laptop / bastion |
| `image-registry.openshift-image-registry.svc:5000` | Pods / deployments inside the cluster |

## Mirror current Instana images into the internal registry

After the registry route is enabled, you can mirror the current Instana images and
preserve both `:latest` and the resolved version tag in the target registry.
The reusable helper script is [`mirror-instana-images.sh`](../scripts/mirror-instana-images.sh).

```bash
TOKEN=$(oc whoami -t)
REGISTRY=$(oc get route default-route \
  -n openshift-image-registry \
  -o jsonpath='{.spec.host}')
INSTANA_AGENT_DOWNLOAD_KEY=""

./mirror-instana-images.sh \
  --registry "$REGISTRY" \
  --namespace instana-mirror \
  --dest-creds "kubeadmin:$TOKEN" \
  --agent-download-key "$INSTANA_AGENT_DOWNLOAD_KEY" \
  --dest-tls-verify false
```

The script currently mirrors:

- `icr.io/instana/instana-agent-operator:latest`
- `containers.instana.io/instana/release/agent/static:latest`
- `icr.io/instana/k8sensor:latest`

It writes a shell-properties file with the resolved versions and digests, for example:

```text
operator_version=2.2.14
operator_digest=sha256:abc123…
operator_source=icr.io/instana/instana-agent-operator:latest
operator_destination=default-route-…/instana-mirror/instana-agent-operator
agent_version=1.320.2
agent_digest=sha256:def456…
agent_source=containers.instana.io/instana/release/agent/static:latest
agent_destination=default-route-…/instana-mirror/agent
k8sensor_version=1.2.3
k8sensor_digest=sha256:ghi789…
k8sensor_source=icr.io/instana/k8sensor:latest
k8sensor_destination=default-route-…/instana-mirror/k8sensor
```

Those values can be captured later for follow-up automation, such as updating the
`InstanaAgent` custom resource to point at the mirrored images.

## Cleanup

```bash
# Remove the test project
oc delete project <your-namespace>

# Disable the external route again (optional)
oc patch configs.imageregistry.operator.openshift.io/cluster \
  --type merge -p '{"spec":{"defaultRoute":false}}'
```
