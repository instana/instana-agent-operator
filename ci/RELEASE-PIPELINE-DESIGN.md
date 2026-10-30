# Release Pipeline Design: Unified Operator + Helm Chart

> Status: **Draft** — not yet implemented  
> Replaces: `ci/pipeline.yaml` (Concourse main pipeline)  
> Scope: `main` branch releases only. FedRAMP branch handling is out of scope.

---

## Overview

The current Concourse main pipeline builds the operator in isolation; the helm chart is built in a separate SPS pipeline in `instana-agent-charts`. This design unifies both into a single release flow owned by the `instana-agent-operator` repository, introduced as two separate SPS pipelines with clear responsibilities:

| Pipeline | Trigger | Purpose |
|---|---|---|
| `release-build` | Manual (UI); optional weekly schedule (disabled initially) | Builds everything, runs E2E, stages artifacts to COS |
| `release-promotion` | Manual (UI); optional weekly schedule (disabled initially) | Promotes staged artifacts to all public destinations |

---

## Version Alignment

The current versions are:
- Operator: `v2.2.21`
- Helm chart: `2.0.53`

From the next release onward, both the operator and the helm chart share a single version number. The first joint release will be `2.2.22` — a patch bump from the operator's current latest tag. The helm chart version jumps from `2.0.53` to `2.2.22` as a one-time alignment. The `versions/INSTANA_AGENT_CHART_VERSION` file in `instana-agent-charts` is no longer used; the version is inferred from the most recent operator git tag at pipeline start.

Git tag convention:
- Operator repo (`instana-agent-operator`): tag uses the `v` prefix → `v2.2.22`
- Helm chart repo (`instana-agent-charts`): tag has no `v` prefix → `2.2.22`

---

## Pipeline 1: `release-build`

**File:** `instana-agent-operator/ci/.pipeline-config-release-build.yaml`
**Trigger:** Manual (SPS toolchain UI). A weekly timed trigger can optionally be enabled once the pipeline is proven stable. Commits to the operator or charts repo do **not** trigger this pipeline.

> **SPS v2 task-naming constraint:** User-defined task names that are not present in the base compliance pipeline cannot be used in `runAfter` references — Tekton's admission webhook rejects the generated `PipelineRun` with `task X depends on Y but Y wasn't present in Pipeline`. The workarounds used here are:
> 1. The `prepare` stage is mapped to the built-in **`code-setup`** slot (the standard pre-build setup task in the base CI pipeline). The script file `ci/sps-scripts/prepare.sh` retains its name; only the pipeline-config task key changes.
> 2. The four E2E tests are defined under the **`async-stage`** task and triggered via `trigger-task` by `code-build` (`build-and-stage.sh`).

### Pipeline properties

| Property | Default | Description |
|---|---|---|
| `release-line` | *(required)* | The `MAJOR.MINOR` release line to publish, e.g. `2.2`. The patch level is computed automatically from existing tags (see stage `prepare`). |
| `release-branch-operator` | `main` | Branch of the operator repo to read the current HEAD SHA from |
| `release-branch-charts` | `main` | Branch of the charts repo to read the current HEAD SHA from |

### Stage overview

```
code-setup   (SPS slot; runs prepare.sh)
    │
    ├── code-checks (async)
    │
    └── code-build
            │ (triggers via trigger-task)
            ├── e2e-operator-ocp-latest     (async-stage)
            ├── e2e-operator-gke-lowest     (async-stage)
            ├── e2e-helm-ocp-lowest         (async-stage)
            └── e2e-helm-gke-latest         (async-stage)
```

The pipeline ends after firing the 4 async E2E stages. There is no fan-in — `release-promotion` acts as the fan-in by reading the COS status files before proceeding.

---

### Stage: `code-setup` (runs `prepare.sh`)

Runs first. Produces the BOM and COS state path used by all subsequent stages. This stage occupies the built-in `code-setup` SPS slot; the implementation script is `ci/sps-scripts/prepare.sh`.

**Steps:**

1. Clone operator repo at tip of `release-branch-operator`, record `HEAD` SHA → `operator-sha`
2. Clone charts repo (`instana-agent-charts`) at tip of `release-branch-charts`, record `HEAD` SHA → `charts-sha`
3. Read `release-line` pipeline property (required, format `MAJOR.MINOR`, e.g. `2.2`). Fail immediately if unset or malformed.
4. Resolve the next patch version for that release line:
   ```bash
   git tag --list "v${MAJOR}.${MINOR}.*" | sed 's/^v//' | sort -V
   ```
   - No tags found for this line → use `MAJOR.MINOR.0`
   - Tags found → take the highest patch and increment by 1

   Tags from other minor lines (e.g. `v2.3.1`) are ignored entirely.

   Examples:
   | `release-line` | Tags present | Result |
   |---|---|---|
   | `2.2` | none | `2.2.0` |
   | `2.2` | `v2.2.0` | `2.2.1` |
   | `2.2` | `v2.2.0`, `v2.2.5`, `v2.3.1` | `2.2.6` |

   Strip the leading `v` for the bare version number; `release-tag` for the operator keeps the `v` prefix (`v2.2.22`); the charts tag has no prefix (`2.2.22`).
5. Construct the unique COS state path using the computed version and a UTC timestamp (precision to the second, format `YYYYMMDDHHmm`):
   ```
   pipeline-state/release-2.2.22/202609251155/
   ```
   Lexicographic sort equals chronological sort, making it immediately obvious which run is newest. Re-running `release-build` for the same version never overwrites a previous run's artifacts. Old run directories expire naturally via the COS lifecycle policy.

   ```bash
   BUILD_TIMESTAMP=$(date -u '+%Y%m%d%H%M')
   COS_STATE_PATH="pipeline-state/release-${VERSION}/${BUILD_TIMESTAMP}"
   ```

6. Write initial `bom.json`:

```json
{
  "version": "2.2.22",
  "release-tag": "v2.2.22",
  "source-branch": "main",
  "cos-state-path": "pipeline-state/release-2.2.22/202609251155",
  "pipeline-run-id": "<pipeline-run-id>",
  "sources": {
    "operator-sha": "<frozen SHA>",
    "charts-sha": "<frozen SHA>"
  },
  "images": {},
  "artifacts": {}
}
```

7. Push `bom.json` to two COS locations:
   - `pipeline-state/release-2.2.22/202609251155/bom.json` (timestamp-scoped, permanent record)
   - `pipeline-state/latest-release-candidate/bom.json` (fixed pointer, overwritten on each new run)
8. Export `cos-state-path=pipeline-state/release-2.2.22/202609251155` via `set_env` / `export_env` for downstream stages

---

### Stage: `code-checks`

Runs async in parallel with `code-build`. Standard SPS compliance gates.

**Steps:** `detect-secrets`, `compliance-checks`, `static-scan`

---

### Stage: `code-build`

Clones both repos at their frozen SHAs from the BOM. Builds everything. Updates and re-pushes the BOM to COS at the end.

**Steps:**

1. Read BOM from COS (`pipeline-state/latest-release-candidate/bom.json`)
2. Clone operator repo at `operator-sha`, clone charts repo at `charts-sha`
3. Run Go unit tests (`make test`)
4. Build all 4 arch images and push a single multiarch manifest to dev ICR in one `docker buildx build` invocation:
   ```bash
   docker buildx build \
     --platform linux/amd64,linux/arm64,linux/s390x,linux/ppc64le \
     --tag icr.io/instana-agent-dev/instana-agent-operator:<operator-sha> \
     --push \
     .
   ```
   Record the manifest digest from the push output → `operator-manifest-digest`
6. Run `helm package` against the charts repo at `charts-sha`, producing `instana-agent-2.2.22.tgz`

   The chart build previously fetched `instana-agent-operator.yaml` from the latest public GitHub release (`github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml`) to extract CRDs, the operator image reference, and the operator version. In the new unified pipeline this artifact is produced locally in the same stage (step 7 below), so the order of operations is reversed:

   a. Build `instana-agent-operator.yaml` first — steps 6 and 7 are executed in reverse order relative to their numbering; step 7 runs first so the artifact exists before the helm package step
   b. Pass `OPERATOR_YAML` into `ci/build.sh` as an environment variable so it reads from the locally-built file rather than downloading from the public release:
      ```bash
      OPERATOR_YAML="${WORKSPACE}/instana-agent-operator.yaml" \
      CHART_VERSION="${VERSION}" \
        bash ci/build.sh
      ```
   c. `ci/build.sh` is updated to check for `OPERATOR_YAML` and use it directly when set, falling back to the public download only when the variable is absent (preserving the existing standalone build path in the charts repo's own PR pipeline)

   `CHART_VERSION` is also injected because `versions/INSTANA_AGENT_CHART_VERSION` is deleted from the charts repo as part of this change; `ci/build.sh` is updated to prefer `CHART_VERSION` over the file when the variable is set.

   The chart is packaged with dev ICR image references at this stage (`icr.io/instana-agent-dev/instana-agent-operator:<operator-sha>`); public references are substituted at publish time by `repackage-helm.sh`.

7. Build OLM bundle (`make bundle`) + `instana-agent-operator.yaml` (`make controller-yaml`) using the dev ICR manifest digest (runs before step 6 — see note in step 6a above)
8. Push artifacts to COS under the timestamp-scoped path `pipeline-state/release-2.2.22/202609251155/`:
   - `helm/instana-agent-2.2.22.tgz`
   - `olm/olm-2.2.22.zip`
   - `olm/instana-agent-operator.yaml`
9. Update BOM with image and artifact entries, push back to both COS locations:

```json
{
  "images": {
    "operator-manifest":        "icr.io/instana-agent-dev/instana-agent-operator:<operator-sha>",
    "operator-manifest-digest": "sha256:..."
  },
  "artifacts": {
    "helm-chart":    "pipeline-state/release-2.2.22/202609251155/helm/instana-agent-2.2.22.tgz",
    "olm-bundle":    "pipeline-state/release-2.2.22/202609251155/olm/olm-2.2.22.zip",
    "operator-yaml": "pipeline-state/release-2.2.22/202609251155/olm/instana-agent-operator.yaml"
  }
}
```

---

### Async E2E stages

All 4 stages share the same pattern:

1. Read BOM from COS to get the operator manifest digest and chart tarball path
2. Claim a reslock on the target cluster before any test work begins. The reslock must be released in a `finally` block so it is always freed, even if the stage is cancelled or fails
3. Record a UTC run timestamp at stage start: `E2E_TIMESTAMP=$(date -u '+%Y%m%d%H%M')`
4. Run the E2E test against the target cluster using the dev ICR image, capturing all output to a local log file
5. Determine status (`success` or `failure`) — the stage script must not exit early; always continue to the upload steps regardless of test outcome
6. Release the reslock
7. Upload the log file to COS with a status suffix in the name — **always uploaded, whether the test passed or failed**:
   ```
   pipeline-state/release-2.2.22/202609251155/e2e/<cluster-id>/test-202609251155-success.log
   pipeline-state/release-2.2.22/202609251155/e2e/<cluster-id>/test-202609251155-failure.log
   ```
   Multiple restarts produce multiple timestamped log files under the same `<cluster-id>/` directory without overwriting each other.
8. Write (overwrite) the status pointer file — **always written, pointing to the outcome of the most recent run**:
   ```
   pipeline-state/release-2.2.22/202609251155/e2e/<cluster-id>.json
   ```
   ```json
   {
     "status": "success",
     "cluster": "ocp-fyre-latest",
     "timestamp": "2026-09-25T11:55:00Z",
     "log": "pipeline-state/release-2.2.22/202609251155/e2e/ocp-fyre-latest/test-202609251155-success.log"
   }
   ```
9. After each COS upload, echo the public download path (no credentials inlined):
   ```
   COS upload: cos://<bucket>/pipeline-state/release-2.2.22/202609251155/e2e/ocp-fyre-latest/test-202609251155-success.log
   COS upload: cos://<bucket>/pipeline-state/release-2.2.22/202609251155/e2e/ocp-fyre-latest.json
   ```
10. Post git commit status to the operator repo at `operator-sha` (context: `tekton/e2e-<cluster-id>`), reflecting the final test status
11. Exit with a non-zero code if status is `failure`, so the SPS stage is marked failed in the toolchain UI

#### Cluster assignment

| Stage | Cluster | Tests |
|---|---|---|
| `e2e-operator-ocp-latest` | `ocp-fyre-latest` | Operator behaviour |
| `e2e-operator-gke-lowest` | `gke-operator-lowest` | Operator behaviour |
| `e2e-helm-ocp-lowest` | `ocp-fyre-lowest` | Helm chart install/upgrade |
| `e2e-helm-gke-latest` | `gke-operator-latest` | Helm chart install/upgrade |

Each stage is independently restartable from the SPS toolchain UI without re-running the build.

---

## BOM: Bill of Materials

The BOM (`bom.json`) is the single hand-off artifact between all three pipelines. It lives in COS at:

- **Versioned + timestamp-scoped:** `pipeline-state/release-<version>/<YYYYMMDDHHmm>/bom.json` — permanent record, lexicographically sortable, safe to re-run without collision
- **Pointer:** `pipeline-state/latest-release-candidate/bom.json` — overwritten on each new run; always points to the most recent candidate

### Full BOM shape

```json
{
  "version": "2.2.22",
  "release-tag": "v2.2.22",
  "source-branch": "main",
  "cos-state-path": "pipeline-state/release-2.2.22/202609251155",
  "pipeline-run-id": "<pipeline-run-id>",
  "sources": {
    "operator-sha": "<40-char SHA>",
    "charts-sha":   "<40-char SHA>"
  },
  "images": {
    "operator-manifest":         "icr.io/instana-agent-dev/instana-agent-operator:<operator-sha>",
    "operator-manifest-digest":  "sha256:..."
  },
  "artifacts": {
    "helm-chart":    "pipeline-state/release-2.2.22/202609251155/helm/instana-agent-2.2.22.tgz",
    "olm-bundle":    "pipeline-state/release-2.2.22/202609251155/olm/olm-2.2.22.zip",
    "operator-yaml": "pipeline-state/release-2.2.22/202609251155/olm/instana-agent-operator.yaml"
  }
}
```

No test results are stored in the BOM. E2E results are separate per-cluster COS files.

---

## Human gate: QA sign-off

After `release-build` completes, a Slack notification is posted to `#tech-agent-delivery-notifications` containing:
- The computed version
- Links to the 4 E2E git commit statuses on the operator repo
- A direct link to run `release-promotion` in the SPS toolchain UI
- The versioned COS path for the BOM

The QA team tests against the dev ICR images. The release owner manually updates `canonical/CHANGELOG.md` in the `instana-agent-charts` repository (direct push or PR to `main`) before triggering promotion. The act of triggering `release-promotion` is the approval — no separate approval tracking is required at this stage.

---

## Pipeline 2: `release-promotion`

**File:** `instana-agent-operator/ci/.pipeline-config-release-promotion.yaml`
**Trigger:** Manual (SPS toolchain UI). A weekly timed trigger can optionally be enabled once the pipeline is proven stable. Commits to the operator or charts repo do **not** trigger this pipeline.

### Pipeline properties

| Property | Default | Description |
|---|---|---|
| `release-state-path` | `pipeline-state/latest-release-candidate/bom.json` | COS path to the BOM. Override to re-publish an older candidate |
| `skip-e2e-check` | `false` | Set to `true` to bypass E2E status file validation (emergency use only) |

### Stage: `validate`

1. Read BOM from COS at `release-state-path`
2. Unless `skip-e2e-check=true`: read all 4 E2E status files from COS. For each:
   - If file is **missing**: fail with `"E2E status missing for <cluster> — stage may not have completed"`
   - If `status != "success"`: fail with `"E2E failed for <cluster>"`
3. Confirm all required BOM fields are present (version, SHAs, image digest, artifact paths)
4. Export the selected candidate's `cos-state-path`. All remaining stages read
   `<cos-state-path>/bom.json`, so a concurrent build cannot change the candidate
   during promotion. The default still selects the latest candidate when the
   manually approved promotion starts.

### Stage: `promote-images`

Uses the frozen manifest digest from the BOM. Applies to public releases only (version matches `^v[0-9]+\.[0-9]+\.[0-9]+$`).

1. `skopeo copy` dev manifest → `icr.io/instana/instana-agent-operator:<version>` (strips `v` prefix)
2. `skopeo copy` dev manifest → `icr.io/instana/instana-agent-operator:latest` (only if `source-branch == "main"`)
3. `skopeo copy` dev manifest → `icr.io/instana-rel/rel-docker-agent-local/instana-agent-operator:<version>`
4. `skopeo copy` dev manifest → `quay.io/redhat-isv-containers/5e961c2c93604e02afa9ebdf:<version>`

### Stage: `preflight-scan`

Runs Red Hat preflight scan against the Red Hat registry image promoted in the previous stage. **Must pass before any publication steps run.**

```bash
preflight check container \
  quay.io/redhat-isv-containers/5e961c2c93604e02afa9ebdf:<version> \
  --certification-component-id=5e961c2c93604e02afa9ebdf \
  --submit \
  --pyxis-api-token=<token>
```

### Stage: `publish-helm-and-release` (`runAfter: preflight-scan`)

Steps run in sequence.

1. **Prepare public artifacts**: download the helm tarball, operator YAML, and OLM
   bundle from the selected candidate in COS. Repackage the chart with public ICR
   image references and rewrite the operator YAML and OLM image namespaces from
   dev ICR to public ICR, preserving their version tags or digests. Keep the staged
   COS artifacts unchanged: release-build and E2E continue using dev images.
2. **Push git tags** at the frozen SHAs from the BOM — tags must exist before the GitHub release is created:
   - Push tag `v2.2.22` on the operator repo at `operator-sha`
   - Push tag `2.2.22` on the charts repo at `charts-sha`
3. **Publish helm chart** (parallel):
   - Push tarball to `github.com/instana/helm-charts` (commit to main)
   - Push tarball to GCP bucket `gs://agents.instana.io/helm/`, update `index.yaml`
   - Push tarball to TaaS Artifactory `instana-team-release-agent-helm-local`
4. **Create public GitHub release** on `github.com/instana/instana-agent-operator` at tag `v2.2.22`:
    - Attach the repackaged public `olm-2.2.22.zip`
    - Attach the repackaged public `instana-agent-operator.yaml`
   - Attach helm chart tarball
   - Release is created as a real (non-draft) release with auto-generated release notes (`generate_release_notes: true`); no changelog editing by the pipeline

### Post-publish

The creation of the public `github.com` release with `olm-2.2.22.zip` attached **automatically triggers** the existing `olm-release-pr-creation` Concourse pipeline, which submits PRs in parallel to:
- `k8s-operatorhub/community-operators`
- `redhat-openshift-ecosystem/certified-operators`
- `redhat-marketplace-operators`

No changes are required to that pipeline.

A final Slack notification is posted to `#tech-agent-delivery-notifications` confirming the published version and links to the public GitHub release.

---

## What changes in `instana-agent-charts`

- `.pipeline-config.yaml` and `.pipeline-config-pr.yaml` — **left untouched** for now
- The `deploy-release` stage `run-stage` script is updated to **echo only** that helm chart release is now handled by the operator release pipeline, and exits `0`. The actual promotion logic is removed
- The `versions/INSTANA_AGENT_CHART_VERSION` file is **deleted** from the repository — the version is driven by the operator git tag from this point forward
- `ci/build.sh` is updated with two backward-compatible changes to support being driven by the operator pipeline:
  - The `OPERATOR_YAML` env var: when set, the `curl` download of `instana-agent-operator.yaml` from the public GitHub release is skipped and the provided file path is used directly. When unset, existing download behaviour is preserved (used by the charts PR pipeline)
  - The `CHART_VERSION` env var: when set, it overrides the `cat versions/INSTANA_AGENT_CHART_VERSION` read. When unset, the file read is used as before. This is required because `versions/INSTANA_AGENT_CHART_VERSION` is deleted from the repo
- PR pipeline clusters remain as-is: `ocp-fyre-lowest` + `gke-operator-latest`

---

## File locations summary

| File | Purpose |
|---|---|
| `instana-agent-operator/ci/.pipeline-config-release-build.yaml` | New: `release-build` pipeline |
| `instana-agent-operator/ci/.pipeline-config-release-promotion.yaml` | New: `release-promotion` pipeline |
| `instana-agent-operator/ci/.pipeline-config.yaml` | Existing PR pipeline — unchanged |
| `instana-agent-operator/ci/pipeline.yaml` | Existing Concourse main pipeline — to be decommissioned |
| `instana-agent-operator/ci/sps-scripts/prepare.sh` | New: version computation + BOM init (runs in the `code-setup` pipeline task slot) |
| `instana-agent-operator/ci/sps-scripts/build-and-stage.sh` | New: multi-arch build + helm package + COS push |
| `instana-agent-operator/ci/sps-scripts/e2e-async.sh` | New: shared async E2E runner (parameterised by cluster) |
| `instana-agent-operator/ci/sps-scripts/validate-release.sh` | New: BOM + E2E status validation |
| `instana-agent-operator/ci/sps-scripts/promote-images.sh` | New: skopeo promotion logic |
| `instana-agent-operator/ci/sps-scripts/repackage-helm.sh` | New: image ref substitution + helm package |
| `instana-agent-operator/ci/sps-scripts/publish-helm.sh` | New: GCP + Artifactory + helm-charts repo push |
| `instana-agent-charts/.pipeline-config.yaml` | Modified: `deploy-release` run-stage replaced with echo stub |
| `instana-agent-charts/.pipeline-config-pr.yaml` | Unchanged |
| `instana-agent-charts/versions/INSTANA_AGENT_CHART_VERSION` | Deleted |

---

## Open items before implementation

1. **COS bucket name and credentials** — confirm which bucket and secret name to use (same as agent delivery pipeline, or a dedicated one for the operator release)
2. **Reslock token secret name** — confirm the secret name for the reslock GitHub token to use in the async E2E scripts (currently `ibm-ghe-tokens.concourse-other-teams` in the Concourse pipeline; verify the equivalent SPS secret name)
