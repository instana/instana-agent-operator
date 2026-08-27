# Deploying the Instana Agent Operator on Kubernetes in an Air-Gapped Environment

This guide walks through deploying the Instana Agent Operator on a Kubernetes cluster
that has no direct internet access. Images are mirrored from their public sources to a
private registry using a jump host (bastion), and the agent is configured to reach the
Instana backend through an HTTP proxy.

The examples use a GKE cluster, but the steps apply to any conformant Kubernetes
distribution.

---

## Mirror Images to IBM Cloud Container Registry

In a fully air-gapped environment the cluster nodes cannot pull images from
`icr.io` or `containers.instana.io` directly. The solution is to copy every required
image to a registry that the cluster *can* reach. This guide uses a custom namespace in
[IBM Cloud Container Registry (ICR)](https://cloud.ibm.com/registry), but any OCI
compliant container registry reachable from a jump host that has internet access and from
the target cluster will work.

```
[internet] ──pull──▶ jump host ──push──▶ icr.io/instana-airgap-mirror ◀──pull── K8s cluster
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
> not mirrored and is not required for this deployment pattern unless you explicitly want
> to do dynamic automatic agent updates.

---

### Prerequisites

Install the following tools on the **jump host** before proceeding:

| Tool | Purpose | Install guide |
|---|---|---|
| [`skopeo`](https://github.com/containers/skopeo) | Copy container images between registries without a Docker daemon | `dnf install skopeo` / `brew install skopeo` |
| [`jq`](https://jqlang.github.io/jq/) | Parse image metadata JSON returned by `skopeo inspect` | `dnf install jq` / `brew install jq` |
| [`kubectl`](https://kubernetes.io/docs/tasks/tools/) | Kubernetes CLI — used to authenticate and interact with the cluster | See Kubernetes docs |
| [`ibmcloud` CLI](https://cloud.ibm.com/docs/cli) + Container Registry plugin (only if icr.io is used as target registry) | Create and authenticate with the ICR namespace | `ibmcloud plugin install container-registry` |

Verify they are available:

```bash
skopeo --version
# skopeo version 1.17.0

jq --version
# jq-1.7.1

kubectl version --client
# Client Version: v1.34.1

ibmcloud cr --version
# Container Registry plug-in version 1.3.21
```

---

### Step 1 — Create the ICR Namespace (only icr.io)

If you do not use icr.io, skip this step and make sure that you have a target registry
prepared.

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
export ICR_API_KEY
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

> **Tip:** Store this file alongside your deployment manifests. The [Deploy the Instana Agent](#deploy-the-instana-agent)
> phase uses the image coordinates it contains to configure the `InstanaAgent` custom resource.

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
pull them once the registry pull secret is configured in the next phase.

---

## Deploy the Operator on Kubernetes

With the images mirrored, the next step is to give the Kubernetes cluster credentials
to pull from ICR, and then deploy the operator using the official release manifest with
the image reference replaced by the mirrored location.

Unlike OpenShift, plain Kubernetes has no `oc secrets link` command or SCCs. Pull
secret association is done by patching `imagePullSecrets` directly onto the relevant
service accounts using `kubectl patch`.

---

### Step 1 — Create the Namespace and Registry Pull Secret

Create the target namespace and the pull secret in one step:

```bash
kubectl create namespace instana-agent
```

```
namespace/instana-agent created
```

```bash
kubectl create secret docker-registry icr-pull-secret \
  --docker-server=icr.io \
  --docker-username=iamapikey \
  --docker-password="$ICR_API_KEY" \
  -n instana-agent
```

```
secret/icr-pull-secret created
```

---

### Step 2 — Patch the Operator Manifest

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
> [Deploy the Instana Agent](#deploy-the-instana-agent).

---

### Step 3 — Apply the Manifest

```bash
kubectl apply -f instana-agent-operator-mirrored.yaml
```

```
namespace/instana-agent configured
customresourcedefinition.apiextensions.k8s.io/agents.instana.io created
customresourcedefinition.apiextensions.k8s.io/agentsremote.instana.io created
serviceaccount/instana-agent-operator created
role.rbac.authorization.k8s.io/instana-agent-clusterrole created
role.rbac.authorization.k8s.io/leader-election-role created
clusterrole.rbac.authorization.k8s.io/instana-agent-clusterrole created
rolebinding.rbac.authorization.k8s.io/leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/instana-agent-clusterrolebinding created
configmap/manager-config created
deployment.apps/instana-agent-controller-manager created
```

---

### Step 4 — Link the Pull Secret to the Operator Service Account

On plain Kubernetes, `imagePullSecrets` must be patched onto the service account
directly — there is no `oc secrets link` equivalent. Do this immediately after `apply`
so the pod does not enter `ErrImagePull` before the patch is in place:

```bash
kubectl patch serviceaccount instana-agent-operator -n instana-agent \
  -p '{"imagePullSecrets": [{"name": "icr-pull-secret"}]}'
```

```
serviceaccount/instana-agent-operator patched
```

> **If you applied the manifest before patching the SA** and the pod is already in
> `ErrImagePull`, patch the SA and then delete the failing pod so it is rescheduled:
>
> ```bash
> kubectl delete pod -n instana-agent -l app.kubernetes.io/name=instana-agent-operator
> ```

---

### Step 5 — Verify the Operator is Running

Wait for the rollout and confirm the pod pulled the image from the mirrored registry:

```bash
kubectl rollout status deployment/instana-agent-controller-manager \
  -n instana-agent --timeout=120s
```

```
Waiting for deployment "instana-agent-controller-manager" rollout to finish: 0 of 1 updated replicas are available...
deployment "instana-agent-controller-manager" successfully rolled out
```

```bash
kubectl get pods -n instana-agent
```

```
NAME                                                READY   STATUS    RESTARTS   AGE
instana-agent-controller-manager-79c5587bd6-tvrqm   1/1     Running   0          28s
```

Confirm the running pod is using the mirrored image:

```bash
kubectl get pod -n instana-agent -l app.kubernetes.io/name=instana-agent-operator \
  -o jsonpath='{.items[0].spec.containers[0].image}{"\n"}'
```

```
icr.io/instana-airgap-mirror/instana-agent-operator:2.2.14
```

The operator is running and pulling exclusively from the internal mirror. The next phase
covers creating the `InstanaAgent` custom resource to deploy the agent itself, pointing
all three images at the mirrored registry and routing backend traffic through the HTTP
proxy.

---

## Deploy the Instana Agent

The operator is controlled by an `InstanaAgent` custom resource. This phase creates one
that:

- points the agent and k8sensor images at the mirrored registry
- routes backend traffic to the Instana SaaS endpoint through an HTTP proxy
- provides registry credentials via `imagePullSecrets` patched onto the agent service accounts

The starting point is the YAML produced by the Instana UI under
**Agents & Collectors → Install Agents → Kubernetes Operator** after entering a cluster
name and agent zone.

---

### Step 1 — Pre-create and Patch the Agent Service Accounts

The operator creates two dedicated service accounts (`instana-agent` and
`instana-agent-k8sensor`) when the `InstanaAgent` CR is first applied. Pre-create both
SAs and patch the pull secret onto them before applying the CR — this avoids an
`ErrImagePull` on first pod scheduling:

```bash
kubectl create serviceaccount instana-agent -n instana-agent
kubectl create serviceaccount instana-agent-k8sensor -n instana-agent

kubectl patch serviceaccount instana-agent -n instana-agent \
  -p '{"imagePullSecrets": [{"name": "icr-pull-secret"}]}'
kubectl patch serviceaccount instana-agent-k8sensor -n instana-agent \
  -p '{"imagePullSecrets": [{"name": "icr-pull-secret"}]}'
```

```
serviceaccount/instana-agent created
serviceaccount/instana-agent-k8sensor created
serviceaccount/instana-agent patched
serviceaccount/instana-agent-k8sensor patched
```

Verify the secret appears on both SAs:

```bash
kubectl get sa instana-agent instana-agent-k8sensor -n instana-agent \
  -o jsonpath='{range .items[*]}{.metadata.name}{": "}{.imagePullSecrets}{"\n"}{end}'
```

```
instana-agent: [{"name":"icr-pull-secret"}]
instana-agent-k8sensor: [{"name":"icr-pull-secret"}]
```

> **If you already applied the CR and pods are in `ErrImagePull`:** run the two
> `kubectl patch serviceaccount` commands above, then delete the failing pods so they
> are rescheduled:
>
> ```bash
> kubectl delete pods -n instana-agent -l app.kubernetes.io/name=instana-agent
> kubectl delete pods -n instana-agent -l app.kubernetes.io/name=instana-agent-k8sensor
> ```

---

### Step 2 — Create the InstanaAgent CR

Create the following manifest. Replace `<INSTANA_AGENT_KEY>` with the agent key and
`<INSTANA_AGENT_DOWNLOAD_KEY>` with the download key from the Instana UI. Adjust the
proxy fields to match your environment — omit the `proxy*` fields entirely if your
cluster has direct connectivity to the Instana backend, or set `proxyUser` and
`proxyPassword` to empty strings if the proxy requires no authentication.

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
    name: instana-k8s-airgapped-mirror-example
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
| `agent.image.tag` | Set to the version resolved during mirroring (`1.320.3`) |
| `agent.proxyHost/Port/Protocol/User/Password` | Added for HTTP proxy egress |
| `k8s_sensor.image.name` | Replaced with `icr.io/instana-airgap-mirror/k8sensor` |
| `k8s_sensor.image.tag` | Set to the version resolved during mirroring (`1.4.16`) |

> **No `pullSecrets` field in the CR is required** — image pull credentials are
> provided by patching `icr-pull-secret` directly onto the `instana-agent` and
> `instana-agent-k8sensor` service accounts in Step 1.

Apply it:

```bash
kubectl apply -f instana-agent-cr.yaml
```

```
instanaagent.instana.io/instana-agent created
```

---

### Step 3 — Verify All Pods are Running

```bash
kubectl get pods -n instana-agent
```

```
NAME                                                READY   STATUS    RESTARTS   AGE
instana-agent-9rjjw                                 1/1     Running   0          3m50s
instana-agent-controller-manager-79c5587bd6-tvrqm   1/1     Running   0          5m51s
instana-agent-jls9c                                 1/1     Running   0          3m50s
instana-agent-k8sensor-8dfb78956-7wprf              1/1     Running   0          3m48s
instana-agent-k8sensor-8dfb78956-d6g42              1/1     Running   0          3m48s
instana-agent-k8sensor-8dfb78956-kfjdz              1/1     Running   0          3m48s
instana-agent-qjjqq                                 1/1     Running   0          3m50s
```

All three components are running:

| Workload | Kind | Pods |
|---|---|---|
| `instana-agent-controller-manager` | Deployment | 1 |
| `instana-agent` | DaemonSet (one per node) | 3 |
| `instana-agent-k8sensor` | Deployment | 3 |

Confirm all images are pulling from the mirror:

```bash
# Agent image
kubectl get pod -n instana-agent -l app.kubernetes.io/name=instana-agent \
  -o jsonpath='{.items[0].spec.containers[0].image}{"\n"}'
# icr.io/instana-airgap-mirror/agent:1.320.3

# k8sensor image
kubectl get pod -n instana-agent instana-agent-k8sensor-8dfb78956-7wprf \
  -o jsonpath='{.spec.containers[0].image}{"\n"}'
# icr.io/instana-airgap-mirror/k8sensor:1.4.16
```

The agent pods will connect to `ingress-red-saas.instana.io:443` through the configured
proxy. Once the backend connection is established, the cluster will appear in the Instana
UI under the configured zone and cluster name.
