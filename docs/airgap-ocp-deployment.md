# Deploying the Instana Agent Operator on OpenShift in an Air-Gapped Environment

This tutorial walks through deploying the Instana Agent Operator on an OpenShift cluster
that has no direct internet access. Images are mirrored from their public sources to a
private registry using a jump host (bastion), and the agent is configured to reach the
Instana backend through an HTTP proxy.

---

## Section 1 — Mirror Images to IBM Cloud Container Registry

### Overview

In a fully air-gapped environment the OpenShift cluster nodes cannot pull images from
`icr.io` or `containers.instana.io` directly. The solution is to copy every required
image to a registry that the cluster *can* reach, this guide will use a custom namespace in
[IBM Cloud Container Registry (ICR)](https://cloud.ibm.com/registry), but every OCI compliant container registry which is reachable
from a jump host that has internet access and the target cluster will work.

```
[internet] ──pull──▶ jump host ──push──▶ icr.io/instana-airgap-mirror ◀──pull── OCP cluster
                      (skopeo)                  (private namespace)
```

The script [`scripts/mirror-instana-images.sh`](../scripts/mirror-instana-images.sh)
handles all three components in one invocation:

| Component | Source image |
|---|---|
| Operator | `icr.io/instana/instana-agent-operator:latest` |
| Agent (static) | `containers.instana.io/instana/release/agent/static:latest` |
| Kubernetes sensor | `icr.io/instana/k8sensor:latest` |

> **Note:** The script copies the **static** agent image. The dynamic agent image is
> not mirrored and is not required for this deployment pattern unless you explicitly want to do dynamic automatic agent updates.

---

### Prerequisites

Install the following tools on the **jump host** before proceeding:

| Tool | Purpose | Install guide |
|---|---|---|
| [`skopeo`](https://github.com/containers/skopeo) | Copy container images between registries without a Docker daemon | `dnf install skopeo` / `brew install skopeo` |
| [`jq`](https://jqlang.github.io/jq/) | Parse image metadata JSON returned by `skopeo inspect` | `dnf install jq` / `brew install jq` |
| [`oc`](https://docs.openshift.com/container-platform/latest/cli_reference/openshift_cli/getting-started-cli.html) | OpenShift CLI — used to authenticate and interact with the cluster | Distributed with OpenShift |
| [`ibmcloud` CLI](https://cloud.ibm.com/docs/cli) + Container Registry plugin (only if icr.io is used as target registry) | Create and authenticate with the ICR namespace | `ibmcloud plugin install container-registry` |

Verify they are available:

```bash
skopeo --version
# skopeo version 1.17.0

jq --version
# jq-1.7.1

oc version --client
# Client Version: 4.17.0

ibmcloud cr --version
# Container Registry plug-in version 1.3.21
```

---

### Step 1 — Create the ICR Namespace (only icr.io)

If you do not use icr.io, skip over this section and make sure that you have a target registry prepared.

For icr.io log in to IBM Cloud and create a dedicated namespace for the mirrored images.

```bash
ibmcloud login --sso
ibmcloud cr region-set global
```

Create the namespace (names must be 4–30 characters, lowercase letters, numbers,
hyphens, and underscores only):

```bash
ibmcloud cr namespace-add instana-airgap-mirror
```

```
Adding namespace 'instana-airgap-mirror' in resource group 'sh-k8s' for account Instana Development in registry icr.io...

Successfully added namespace 'instana-airgap-mirror'

OK
```

Confirm the namespace is listed:

```bash
ibmcloud cr namespace-list
```

```
Listing namespaces for account 'Instana Development' in registry 'icr.io'...

Namespace
instana-airgap-mirror

OK
```

---

### Step 2 — Authenticate skopeo to ICR

Create an IAM API key for non-interactive registry access and pass it as the
skopeo destination credential. Keep the key value out of shell history by reading
it from an environment variable.

```bash
# Create a dedicated API key (the secret value is printed once — store it securely)
ibmcloud iam api-key-create instana-mirror-key -d "skopeo mirror key"
```

Export the key **without** echoing it to the terminal:

```bash
read -rs ICR_API_KEY
```

Verify login works:

```bash
skopeo login \
  --username iamapikey \
  --password "$ICR_API_KEY" \
  icr.io
```

```
Login Succeeded!
```

> **Troubleshooting — `currently logged in, auth file contains an Identity token`**
>
> A previous `ibmcloud cr login` wrote an `identitytoken` field into `~/.docker/config.json`
> that skopeo cannot overwrite (and `skopeo logout` cannot remove). Open the file, delete
> the `icr.io` entry, save it, and retry `skopeo login`.

---

### Step 3 — Obtain an Instana Agent Download Key

The static agent image lives on `containers.instana.io` which requires an agent
download key. Retrieve it from the Instana backend UI under
**Settings → Agents → Install Agent** or ask your Instana account team.

Store it without echoing:

```bash
read -rs INSTANA_AGENT_DOWNLOAD_KEY
```

---

### Step 4 — Run the Mirror Script

From the jump host, run the helper script. It resolves the current `:latest` version
for each image, copies both `:latest` and the versioned tag to the destination, and
writes a properties file with all resolved coordinates.

```bash
./scripts/mirror-instana-images.sh \
  --registry      icr.io \
  --namespace     instana-airgap-mirror \
  --dest-creds    "iamapikey:$ICR_API_KEY" \
  --dest-tls-verify true \
  --agent-download-key "$INSTANA_AGENT_DOWNLOAD_KEY"
```

Expected output (versions will reflect the current release at the time you run it):

```
[2026-06-23T11:18:11Z] START: skopeo login containers.instana.io
[2026-06-23T11:18:12Z] DONE: skopeo login containers.instana.io (1s, exit=0)
[2026-06-23T11:18:13Z] Destination credentials stored in temp auth file (not in process args)
[2026-06-23T11:18:13Z] Resolving operator version from icr.io/instana/instana-agent-operator:latest
[2026-06-23T11:18:13Z] Inspecting digest for icr.io/instana/instana-agent-operator:latest
[2026-06-23T11:18:18Z] Checking icr.io/instana/instana-agent-operator:2.2.14 against latest digest
[2026-06-23T11:18:20Z] Matched latest digest with icr.io/instana/instana-agent-operator:2.2.14
[2026-06-23T11:18:20Z] Resolving agent version from containers.instana.io/instana/release/agent/static:latest
[2026-06-23T11:18:20Z] Inspecting label version for containers.instana.io/instana/release/agent/static:latest
[2026-06-23T11:18:23Z] Resolving k8sensor version from icr.io/instana/k8sensor:latest
[2026-06-23T11:18:23Z] Inspecting digest for icr.io/instana/k8sensor:latest
[2026-06-23T11:18:28Z] Checking icr.io/instana/k8sensor:2.0.0 against latest digest
[2026-06-23T11:18:30Z] Checking icr.io/instana/k8sensor:1.4.16 against latest digest
[2026-06-23T11:18:33Z] Matched latest digest with icr.io/instana/k8sensor:1.4.16
[2026-06-23T11:18:33Z] START: Copying icr.io/instana/instana-agent-operator:latest -> icr.io/instana-airgap-mirror/instana-agent-operator:latest
[2026-06-23T11:18:59Z] DONE: Copying icr.io/instana/instana-agent-operator:latest -> icr.io/instana-airgap-mirror/instana-agent-operator:latest (26s, exit=0)
[2026-06-23T11:18:59Z] START: Copying icr.io/instana/instana-agent-operator:latest -> icr.io/instana-airgap-mirror/instana-agent-operator:2.2.14
[2026-06-23T11:19:04Z] DONE: Copying icr.io/instana/instana-agent-operator:latest -> icr.io/instana-airgap-mirror/instana-agent-operator:2.2.14 (5s, exit=0)
[2026-06-23T11:19:07Z] START: Copying containers.instana.io/instana/release/agent/static:latest -> icr.io/instana-airgap-mirror/agent:latest
[2026-06-23T11:37:41Z] DONE: Copying containers.instana.io/instana/release/agent/static:latest -> icr.io/instana-airgap-mirror/agent:latest (1114s, exit=0)
[2026-06-23T11:37:41Z] START: Copying containers.instana.io/instana/release/agent/static:latest -> icr.io/instana-airgap-mirror/agent:1.320.3
[2026-06-23T11:37:47Z] DONE: Copying containers.instana.io/instana/release/agent/static:latest -> icr.io/instana-airgap-mirror/agent:1.320.3 (6s, exit=0)
[2026-06-23T11:37:50Z] START: Copying icr.io/instana/k8sensor:latest -> icr.io/instana-airgap-mirror/k8sensor:latest
[2026-06-23T11:38:17Z] DONE: Copying icr.io/instana/k8sensor:latest -> icr.io/instana-airgap-mirror/k8sensor:latest (27s, exit=0)
[2026-06-23T11:38:17Z] START: Copying icr.io/instana/k8sensor:latest -> icr.io/instana-airgap-mirror/k8sensor:1.4.16
[2026-06-23T11:38:22Z] DONE: Copying icr.io/instana/k8sensor:latest -> icr.io/instana-airgap-mirror/k8sensor:1.4.16 (5s, exit=0)
[2026-06-23T11:38:25Z] Wrote mirror properties to ./mirror-instana-images.properties
```

The generated `mirror-instana-images.properties` file will look like:

```properties
operator_version=2.2.14
operator_digest=sha256:8bcfb873a8b7741766ce3a70551541fd5ca9d6f3c144346d1c1b9d8987693141
operator_source=icr.io/instana/instana-agent-operator:latest
operator_destination=icr.io/instana-airgap-mirror/instana-agent-operator
agent_version=1.320.3
agent_digest=sha256:ffae7e9c629316f7caa8f9d4a75caf5561d2d2915e0f988119d0fdb7e74360fc
agent_source=containers.instana.io/instana/release/agent/static:latest
agent_destination=icr.io/instana-airgap-mirror/agent
k8sensor_version=1.4.16
k8sensor_digest=sha256:e541cc7a2dacd2a5b48338b2cd6dc99e5665cea197e79dab61a79b32db596b37
k8sensor_source=icr.io/instana/k8sensor:latest
k8sensor_destination=icr.io/instana-airgap-mirror/k8sensor
```

> **Tip:** Store this file alongside your deployment manifests. The next section uses
> the image coordinates it contains to configure the `InstanaAgent` custom resource.

---

### Step 5 — Verify the Images in ICR

Confirm all three repositories are present and have the expected tags:

```bash
ibmcloud cr images --restrict instana-airgap-mirror
```

```
Listing images...

Repository                                          Tag       Digest         Namespace               Created          Size     Security status
icr.io/instana-airgap-mirror/agent                  1.320.3   sha256:ffae…   instana-airgap-mirror   2 minutes ago    897 MB   No Issues
icr.io/instana-airgap-mirror/agent                  latest    sha256:ffae…   instana-airgap-mirror   2 minutes ago    897 MB   No Issues
icr.io/instana-airgap-mirror/instana-agent-operator 2.2.14    sha256:8bcf…   instana-airgap-mirror   20 minutes ago   67 MB    No Issues
icr.io/instana-airgap-mirror/instana-agent-operator latest    sha256:8bcf…   instana-airgap-mirror   20 minutes ago   67 MB    No Issues
icr.io/instana-airgap-mirror/k8sensor               1.4.16    sha256:e541…   instana-airgap-mirror   2 minutes ago    43 MB    No Issues
icr.io/instana-airgap-mirror/k8sensor               latest    sha256:e541…   instana-airgap-mirror   2 minutes ago    43 MB    No Issues

OK
```

The images are now available at `icr.io/instana-airgap-mirror/` and the cluster can
pull them once the registry pull secret is configured (covered in the next section).

---

## Section 2 — Deploy the Operator on OpenShift

### Overview

With the images mirrored, the next step is to give the OpenShift cluster credentials to
pull from ICR, and then deploy the operator using the official release manifest with the
image reference replaced by the mirrored location.

---

### Step 1 — Prepare the Namespace and Security Context Constraints

Create the target namespace and grant the service accounts the SCCs they require.
The agent pod runs privileged (host network, host PID), and the remote-agent SA needs
`anyuid` for its init container.

```bash
INSTANA_AGENT_NAMESPACE=instana-agent

oc new-project ${INSTANA_AGENT_NAMESPACE}
oc adm policy add-scc-to-user privileged -z instana-agent -n "${INSTANA_AGENT_NAMESPACE}"
oc adm policy add-scc-to-user anyuid -z instana-agent-remote -n "${INSTANA_AGENT_NAMESPACE}"
```

```
Now using project "instana-agent" on server "https://api.<cluster>:6443".
clusterrole.rbac.authorization.k8s.io/system:openshift:scc:privileged added: "instana-agent"
clusterrole.rbac.authorization.k8s.io/system:openshift:scc:anyuid added: "instana-agent-remote"
```

---

### Step 2 — Create the Registry Pull Secret

Create a Kubernetes pull secret with the same IAM API key used in Section 1 and link
it to the operator service account.

> **Order matters:** the `instana-agent-operator` service account is created by the
> operator manifest in the next step. Pre-create it here so the pull secret is already
> linked before the Deployment pod is scheduled — this avoids an `ImagePullBackOff`
> on first startup.

```bash
# Create the pull secret
oc create secret docker-registry icr-pull-secret \
  --docker-server=icr.io \
  --docker-username=iamapikey \
  --docker-password="$ICR_API_KEY" \
  -n instana-agent
```

```
secret/icr-pull-secret created
```

```bash
# Pre-create the operator service account and link the secret before deploying
oc create serviceaccount instana-agent-operator -n instana-agent
oc secrets link instana-agent-operator icr-pull-secret --for=pull -n instana-agent
```

```
serviceaccount/instana-agent-operator created
```

---

### Step 3 — Patch the Operator Manifest

Download the latest release manifest and replace the upstream operator image reference
with the mirrored one using `sed`:

```bash
curl -fsSL \
  https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml \
  -o instana-agent-operator.yaml

sed 's|image: icr.io/instana/instana-agent-operator:|image: icr.io/instana-airgap-mirror/instana-agent-operator:|g' \
  instana-agent-operator.yaml > instana-agent-operator-mirrored.yaml

# Verify the replacement
grep 'image: icr.io' instana-agent-operator-mirrored.yaml
```

```
image: icr.io/instana-airgap-mirror/instana-agent-operator:2.2.14
```

> **Note:** Only the operator image line is replaced here. The `InstanaAgent` custom
> resource that references the agent and k8sensor images is configured separately in
> Section 3.

---

### Step 4 — Apply the Manifest

```bash
oc apply -f instana-agent-operator-mirrored.yaml
```

```
namespace/instana-agent configured
customresourcedefinition.apiextensions.k8s.io/agents.instana.io created
customresourcedefinition.apiextensions.k8s.io/agentsremote.instana.io created
serviceaccount/instana-agent-operator configured
role.rbac.authorization.k8s.io/instana-agent-clusterrole created
role.rbac.authorization.k8s.io/leader-election-role created
clusterrole.rbac.authorization.k8s.io/instana-agent-clusterrole created
rolebinding.rbac.authorization.k8s.io/leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/instana-agent-clusterrolebinding created
configmap/manager-config created
deployment.apps/instana-agent-controller-manager created
```

---

### Step 5 — Verify the Operator is Running

Wait for the rollout and confirm the pod pulled the image from the mirrored registry:

```bash
oc rollout status deployment/instana-agent-controller-manager \
  -n instana-agent --timeout=120s
```

```
Waiting for deployment "instana-agent-controller-manager" rollout to finish: 0 of 1 updated replicas are available...
deployment "instana-agent-controller-manager" successfully rolled out
```

```bash
oc get pods -n instana-agent
```

```
NAME                                               READY   STATUS    RESTARTS   AGE
instana-agent-controller-manager-5dd4b68f7-schm2   1/1     Running   0          27s
```

Confirm the running pod is using the mirrored image:

```bash
oc get pod -n instana-agent -l app.kubernetes.io/name=instana-agent-operator \
  -o jsonpath='{.items[0].spec.containers[0].image}{"\n"}'
```

```
icr.io/instana-airgap-mirror/instana-agent-operator:2.2.14
```

The operator is running and pulling exclusively from the internal mirror. The next
section covers creating the `InstanaAgent` custom resource to deploy the agent itself,
pointing all three images at the mirrored registry and routing backend traffic through
the HTTP proxy.

---

## Section 3 — Deploy the Instana Agent

### Overview

The operator is controlled by an `InstanaAgent` custom resource. This section creates
one that:

- points the agent and k8sensor images at the mirrored registry
- routes backend traffic to the Instana SaaS endpoint through an HTTP proxy
- relies on the namespace-wide pull secret set up in Section 2

The starting point is the YAML produced by the Instana UI under
**Agents & Collectors → Install Agents → OpenShift Operator** after entering a cluster
name and agent zone.

---

### Step 1 — Link the Pull Secret to the Operator Service Accounts

The operator creates two dedicated service accounts (`instana-agent` and
`instana-agent-k8sensor`) when the `InstanaAgent` CR is first applied. OpenShift does
**not** propagate pull secrets from the `default` SA to pods running under these
operator-managed SAs, so the secret must be linked to each one explicitly.

Pre-create both SAs now so the links are in place before the CR is applied — this
avoids an `ErrImagePull` on first pod scheduling:

```bash
oc create serviceaccount instana-agent -n instana-agent
oc create serviceaccount instana-agent-k8sensor -n instana-agent

oc secrets link instana-agent icr-pull-secret --for=pull -n instana-agent
oc secrets link instana-agent-k8sensor icr-pull-secret --for=pull -n instana-agent
```

Verify the secret appears on both SAs:

```bash
oc get sa instana-agent instana-agent-k8sensor -n instana-agent \
  -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.imagePullSecrets}{"\n"}{end}'
```

```
instana-agent: [{"name":"instana-agent-dockercfg-xxxxx"},{"name":"icr-pull-secret"}]
instana-agent-k8sensor: [{"name":"instana-agent-k8sensor-dockercfg-xxxxx"},{"name":"icr-pull-secret"}]
```

> **If you already applied the CR and pods are in `ErrImagePull`:** the SAs exist but
> the secret is not linked yet. Run the two `oc secrets link` commands above, then
> delete the failing pods so they are rescheduled:
>
> ```bash
> oc delete pods -n instana-agent -l app.kubernetes.io/name=instana-agent
> oc delete pods -n instana-agent -l app.kubernetes.io/name=instana-agent-k8sensor
> ```

---

### Step 2 — Create the InstanaAgent CR

Create the following manifest. Replace `<INSTANA_AGENT_KEY>` with the agent key and
`<INSTANA_AGENT_DOWNLOAD_KEY>` with the download key from the Instana UI. Adjust the
proxy fields to match your environment — set `proxyUser` and `proxyPassword` to empty
strings if the proxy requires no authentication.

```yaml
apiVersion: instana.io/v1
kind: InstanaAgent
metadata:
  name: instana-agent
  namespace: instana-agent
spec:
  zone:
    name: airgapped
  cluster:
    name: instana-ocp-airgapped-mirror-example
  agent:
    key: <INSTANA_AGENT_KEY>
    downloadKey: <INSTANA_AGENT_DOWNLOAD_KEY>
    endpointHost: ingress-red-saas.instana.io
    endpointPort: "443"
    proxyHost: proxy.example.com
    proxyPort: "3128"
    proxyProtocol: http
    proxyUser: ""
    proxyPassword: ""
    image:
      name: icr.io/instana-airgap-mirror/agent
      tag: "1.320.3"
    env: {}
    configuration_yaml: |
      # You can leave this empty, or use this to configure your instana agent.
      # See https://docs.instana.io/setup_and_manage/host_agent/on/kubernetes/
  k8s_sensor:
    image:
      name: icr.io/instana-airgap-mirror/k8sensor
      tag: "1.4.16"
```

Key differences from the UI-generated YAML:

| Field | Change |
|---|---|
| `agent.image.name` | Replaced with `icr.io/instana-airgap-mirror/agent` |
| `agent.image.tag` | Set to the version resolved in Section 1 (`1.320.3`) |
| `agent.proxyHost/Port/Protocol/User/Password` | Added for HTTP proxy egress |
| `k8s_sensor.image.name` | Replaced with `icr.io/instana-airgap-mirror/k8sensor` |
| `k8s_sensor.image.tag` | Set to the version resolved in Section 1 (`1.4.16`) |

> **No `pullSecrets` in the CR are required** — image pull credentials are provided by
> linking `icr-pull-secret` directly to the `instana-agent` and `instana-agent-k8sensor`
> service accounts in Step 1.

Apply it:

```bash
oc apply -f instana-agent-cr.yaml
```

```
instanaagent.instana.io/instana-agent created
```

---

### Step 3 — Verify All Pods are Running

```bash
oc get pods -n instana-agent
```

```
NAME                                               READY   STATUS    RESTARTS   AGE
instana-agent-82w5l                                1/1     Running   0          2m38s
instana-agent-bl54n                                1/1     Running   0          2m37s
instana-agent-controller-manager-5dd4b68f7-schm2   1/1     Running   0          20m
instana-agent-dp778                                1/1     Running   0          2m36s
instana-agent-k8sensor-b79fdfcbd-4ppkb             1/1     Running   0          2m37s
instana-agent-k8sensor-b79fdfcbd-rj7k4             1/1     Running   0          2m38s
instana-agent-k8sensor-b79fdfcbd-xk9tq             1/1     Running   0          2m38s
```

All three components are running:

| Workload | Kind | Pods |
|---|---|---|
| `instana-agent-controller-manager` | Deployment | 1 |
| `instana-agent` | DaemonSet (one per node) | 3 |
| `instana-agent-k8sensor` | Deployment | 3 |

The agent pods will connect to `ingress-red-saas.instana.io:443` through the configured
proxy. Once the backend connection is established, the cluster will appear in the Instana
UI under the configured zone and cluster name.
