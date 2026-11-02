# Unified operator + Helm release pipeline

Status: implementation hardened locally; **not yet validated by a new SPS run**.
This document records the agreed behavior after the 2026-11-02 review. The
[readiness review](RELEASE-PIPELINE-READINESS.md) retains the original log evidence
and historical findings. No production promotion has been executed for this work.

## Scope and decisions

- Release builds stay lean: SPS `simple-execute`, unit tests, artifact generation
  and four asynchronous E2E webhooks. No Slack and no new compliance tasks.
  A separate main-branch compliance pipeline can be added later.
- Promotion is manual and selects an **exact prebuilt COS BOM path**. There is
  no `latest-release-candidate` selection or automatic promotion.
- Operator and charts share a version. Image references remain readable tags.
- For each required E2E scenario, **one successful attempt for the exact candidate
  is sufficient**. Later intermittent failures do not revoke that pass.
- Finish `canonical/CHANGELOG.md` before the final release build; publication
  never rebuilds the chart from a newer charts HEAD.
- SPS promotion owns publication to all registries, Helm destinations, GitHub
  releases, and the three OLM catalog PRs. The old Concourse OLM publisher must
  be paused/reconfigured at cutover to avoid duplicate ownership.
- Keep the existing build-tool image for now:
  `icr.io/instana-agent-dev/agent-delivery/build-image:main`. Switching to its
  promoted location remains a production configuration decision; do not confuse
  tool images with the candidate operator images staged in dev ICR.
- This flow publishes `main` releases only; FedRAMP is unchanged/out of scope.

## Version discovery through APIs

`ci/sps-scripts/release_api.py` uses authenticated, paginated APIs:

| Repository | API | Secret |
|---|---|---|
| Public operator | `https://api.github.com/repos/instana/instana-agent-operator` | `github-token` |
| Charts | `https://github.ibm.com/api/v3/repos/instana/instana-agent-charts` | `ghe-token` |

Preparation lists every operator tag, accepts only canonical stable `vX.Y.Z`
versions, filters the requested `release-line`, and increments the highest patch.
No tags on the line means `.0`. Local tags, API ordering, prerelease suffixes,
malformed tags and tags from other lines cannot change this result.

Charts tags are queried through GHE, the highest stable charts tag is logged,
and a conflicting proposed charts tag fails preparation. Authentication and API
errors fail the build rather than silently falling back to local state.

The OLM predecessor is also frozen at preparation: the highest published stable
operator release below the proposed version that has its `olm-X.Y.Z.zip` asset.
This supports a new minor line upgrading from the previous line. With no eligible
predecessor the pipeline stops for an explicit initial-catalog policy decision.
This selects a published bundle, not proof that upstream catalogs have merged it;
verify catalog upgrade-chain readiness before release.

At the latest read-only check, the public API reports `v2.2.23` (published
2026-10-30), making the next `2.2` candidate `2.2.24`. Local GHE API credentials
returned HTTP 401 during development; the charts remote version remains to be
verified in SPS with `ghe-token`. No remote version is inferred from local tags.

## Release-build

Configuration: `ci/.pipeline-config-release-build.yaml`.

| Property | Default | Meaning |
|---|---|---|
| `release-line` | required | Canonical `MAJOR.MINOR`, e.g. `2.2` |
| `release-branch-operator` | `main` | Source branch frozen in the BOM |
| `release-branch-charts` | `main` | Charts/test source branch frozen in the BOM |
| `cos-bucket-name` | required | Candidate/evidence storage |
| `cos-region` | `us-south` | COS region |

Production trigger intent is manual; the development toolchain currently runs
on operator pushes. Trigger behavior is configured in SPS, not enforced by YAML
comments. Charts pushes currently do not run pipelines.

### Candidate identity and operator image

The identity is UTC **seconds** plus the actual run UUID extracted from the SPS
run URL. If the UUID is unavailable, generate a UUID, never use the generic
`simple-pipelinerun` name as identity:

```text
candidate-id: 20261102123045-12345678-1234-1234-1234-123456789abc
COS prefix: pipeline-state/release-2.2.24/<candidate-id>
image: icr.io/instana-agent-dev/instana-agent-operator:2.2.24-<candidate-id>
```

Multiple unpublished builds of the same version therefore have distinct readable
image tags and COS locations. Operator `VERSION` labels and chart/appVersion
remain `2.2.24`; the candidate suffix is only the image tag/identity.

The build prints this exact line during preparation **and after staging**:

```text
PROMOTION INPUT: release-state-path=pipeline-state/release-2.2.24/<candidate-id>/bom.json
```

Copy that value into the promotion trigger. Use the corresponding run URL and
COS prefix when handing the candidate to QA; no mutable latest pointer is needed.

### Build and BOM

`prepare.sh` freezes both source SHAs and the predecessor, authenticates to COS,
and writes the initial scoped BOM. `build-and-stage.sh` then:

1. Checks out both frozen SHAs; runs operator unit tests.
2. Builds amd64, arm64, s390x and ppc64le in one Buildx invocation.
3. Records the index digest from Buildx's metadata file, not a later shared-tag
   lookup; verifies the readable candidate tag resolves to that digest.
4. Builds operator YAML and validated OLM bundle with the candidate tag.
5. Builds the chart from that local YAML, preserving its candidate image tag and
   semantic operator-version label. Chart CRDs come from this operator build.
6. Uploads the chart, OLM zip and YAML. Adds SHA-256 checksums, image index digest
   and `build-complete: true` to the BOM.
7. Attempts all four webhook dispatches; if any fails, marks the entrypoint failed
   after attempting the others. Missing successful evidence still blocks promotion.

The BOM includes `candidate-id`, version/tag, previous-version, both source
branches/SHAs, COS prefix, run UUID/URL, image tag/digest, artifact keys/checksums
and build completion. It contains no mutable latest-attempt test status.

### E2E matrix and retry contract

| Installer | Platform/version role | Cluster secret |
|---|---|---|
| Operator | OCP latest | `ocp-fyre-latest` |
| Operator | GKE lowest | `gke-operator-lowest` |
| Helm | OCP lowest | `ocp-fyre-lowest` |
| Helm | GKE latest | `gke-operator-latest` |

This provides one OCP and one GKE per installer, with lowest/latest mixed for
each installer. The supplied logs show both GKE clusters on 1.35.8, so provision
the intended lower GKE version separately; changing a cluster name is not coverage.

Each child reads the scoped BOM and frozen tests, authenticates independently,
claims its reslock, verifies the tagged operator digest **before and after** the
suite, and verifies the downloaded chart checksum. Operator tests get explicit
`OPERATOR_IMAGE_NAME` and `OPERATOR_IMAGE_TAG`; Helm uses staged chart defaults.

Every completed attempt has a seconds+UUID log and JSON result under:

```text
<prefix>/e2e/<cluster>/test-<attempt>-success.log
<prefix>/e2e/<cluster>/test-<attempt>-failure.log
<prefix>/e2e/<cluster>/<attempt>.json
```

Only successful attempts update the durable pass pointer:

```text
<prefix>/e2e/success/<cluster>.json
```

Each result binds candidate ID, both SHAs, image digest/tag, artifact checksums,
installer mode, cluster, attempt and log/checksum. A failure never overwrites the
pass pointer. Concurrent successful attempts are interchangeable because their
inputs must match. A pass from another candidate is never sufficient.

The EXIT finalizer attempts lock release and evidence writes on setup/test errors;
TERM/INT exit nonzero through the same finalizer. No pod process can guarantee
uploads or cleanup after SIGKILL: lease expiry is the fallback. Reslock helper
downloads now fail on HTTP errors, reject empty files and check shell syntax.
Release failures are not silently treated as successful release. Suites are
limited to 55 minutes (+60 seconds termination grace), below the 70-minute lease
with headroom for chart download/digest checks and cleanup. Do not lengthen that
timeout beyond the lease without changing the lease/renewal policy. Digest
lookups are bounded to two minutes and the locked chart download to five minutes,
so these network operations cannot consume the entire lease before a suite starts.

The entrypoint being green means build and dispatch completed, **not** that the
four independently restartable child runs have all passed. QA and promotion use
the candidate's successful-evidence pointers instead.

## Release-promotion

Configuration: `ci/.pipeline-config-release-promotion.yaml`. One supported SPS
`simple-execute` task runs `run-promotion.sh`; no unsupported custom task graph.

| Property | Default | Meaning |
|---|---|---|
| `release-state-path` | **required, no default** | Exact candidate `<prefix>/bom.json` printed by build |
| `skip-e2e-check` | `false` | Explicit emergency bypass of E2E checks only |
| `olm-fork-owner` | `instana` | Organization owning the three catalog forks |

**Set promotion concurrency to one in SPS before enabling the trigger.** Shared
registry tags, chart repos and catalogs require serialized publication. GCP also
uses generation conditions to detect competing index writers. Existing-version
bytes and tags are never deliberately replaced by the versioned publication steps.

### Ordered steps

1. **Validate:** explicitly log in to IBM Cloud. Validate candidate schema/path,
   both main-branch declarations, stable version/tag, SHAs/digest and completed
   build. Download all three artifacts and verify checksums. Unless forced,
   require all four durable pass pointers to match the BOM and verify their logs.
   Freeze the selected BOM bytes locally for every remaining step.
2. **Read-only tag/release preflight:** reject existing tags that resolve to other
   SHAs and any newer public release (no downgrading publication).
3. **Stage Red Hat image:** copy from the BOM digest with `skopeo --all
   --preserve-digests`; accept an existing version only when its digest matches.
4. **Red Hat preflight:** use explicit Quay pull authentication and the Pyxis token;
   failure stops before public ICR version/latest or Helm/release publication.
5. **Prepare public artifacts:** download the checksummed staged inputs; change
   operator image references to `icr.io/instana/instana-agent-operator:<version>`
   in chart defaults/fallback, operator YAML and OLM. Keep staged artifacts intact.
   Normalize tar/gzip/zip metadata so retries produce byte-identical outputs.
6. **Offline public smoke test:** real `helm lint` and `helm template` with defaults
   and null-image fallback; inspect actual deployment images, both CRDs,
   chart/appVersion and CSV version/replaces/container references. Reject dev
   namespaces and placeholders anywhere in the final archives/manifests.
7. **Promote versioned images:** public ICR and internal release ICR, from the
   frozen digest, all platforms, preserving and verifying the digest. An existing
   different versioned image is an error, not permission to overwrite.
8. **Publish tags and chart:** idempotent GitHub/GHE tag API operations at the
   frozen SHAs. GCP tarball first, merged index last with generation precondition;
   Artifactory existing-byte comparison; update `instana/helm-charts` from the
   public archive with staged-diff detection and a signed-off commit.
9. **Publish GitHub release:** create/resume a draft. Compare existing asset bytes,
   upload missing assets and publish only after all three are complete. An already
   published incomplete release requires manual recovery rather than silently
   mutating a public release. Release notes are auto-generated for the operator;
   the chart changelog was already frozen before the build.
10. **Publish OLM PRs in SPS:** create/resume one version-specific branch and PR in
    each of `k8s-operatorhub/community-operators`,
    `redhat-openshift-ecosystem/certified-operators`, and
    `redhat-openshift-ecosystem/redhat-marketplace-operators`. Preserve the legacy
    marketplace CSV filename, workflow annotations and package-name adaptations.
    Verify existing branch bytes; recognize matching merged PRs. Never force-push,
    reset fork main, or rebase unrelated open PRs. Closed-unmerged conflicts stop
    for human intervention. PR submission is not upstream merge/certification.
11. **Activate latest:** only after required publications/PR submissions; refuse
    to downgrade the current image version. Store a COS promotion receipt with
    candidate, run URL, bypass setting and catalog PR links.

Rerunning the same promotion resumes through matching existing tags, images,
archives, assets and PRs. Different bytes under an existing version fail closed.
Red Hat scan/submission may repeat on retry. There is no transaction spanning all
services: partial progress remains visible, but recovery must reuse the same
candidate rather than rebuild/retag a different candidate under that version.

## Charts development and PR pipeline

- Retire the old charts main trigger at merge. Its YAML and legacy build-script
  entrypoint no longer publish; the operator release pipeline owns releases.
- Keep `.pipeline-config-pr.yaml` and its E2E tests enabled. Standalone PR setup
  installs the Go toolchain declared by the charts `go.mod` and verifies the
  downloaded archive checksum. The unified runner reuses operator-installed Go.
- Local `make build` uses `CHART_VERSION=0.0.0-dev` if unset. It still downloads
  the latest public operator YAML unless `OPERATOR_YAML` is provided. Local
  development does not need to edit a version file.
- The release caller supplies `CHART_VERSION` and `OPERATOR_YAML`; its semantic
  appVersion is separate from the candidate-specific image tag.

## Cutover checklist and remaining external validation

1. Confirm `github-token`, `ghe-token`, `github-instana-agent-build`, dev/public ICR
   key permissions, `icr-rel-api-key`, Quay/Pyxis credentials, GCP write account,
   Artifactory credentials, E2E/reslock/cluster secrets and COS IAM/retention.
2. Ensure the existing build image contains required tools (especially preflight,
   compatible yq/Helm, skopeo, gcloud/gsutil and IBM COS plugin). Startup checks
   stop promotion if tooling is absent; no previous container login is assumed.
3. Review/apply toolchain properties, manual promotion permissions, one concurrent
   promotion maximum, webhook access, source/config branch protection and trigger
   filters. API version discovery must succeed against GHE using SPS credentials.
4. Provision the real lowest GKE version. Keep the four installer/platform roles.
   Four-architecture builds still do not imply non-amd64 runtime E2E or OLM install
   coverage; decide additional qualification separately.
5. Finish charts changelog; merge both repos. Charts push alone does not test the
   unified candidate. Operator pushes currently trigger a new build: coordinate
   the merge/push and expect that run. No pushes were performed during this work.
6. Pause the legacy charts main pipeline and the old Concourse OLM PR job. Its
   checked-in release trigger is now disabled, but running deployments must be
   updated/paused explicitly. Retain any unrelated Concourse image-build jobs
   and the out-of-scope FedRAMP path.
7. Run a new nonpublishing release build; retry one child independently and verify
   durable pass-pointer behavior, digest checks, evidence identity and reslock
   cleanup. Old log candidates lack the new schema and require a new build.
8. Validate SPS promotion admission with inert/no-write scripts and test destination
   credentials without publication. Local fixture tests cannot prove real cloud
   permissions, Pyxis acceptance or GitHub fork/PR rights.
9. Approve an exact final main-branch candidate. Production promotion remains a
   separately authorized action; **do not run it as a validation experiment**.

## Local regression checks

```bash
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s ci/sps-scripts -v
# Additionally run the real Helm/yq integration fixture with both tools on PATH:
PYTHONDONTWRITEBYTECODE=1 RELEASE_TEST_TOOL_DIR=/path/to/tools \
  python3 -m unittest discover -s ci/sps-scripts -v
```

Tests are offline/mocked for external services. The real-tool fixture uses the
checked-out chart templates and CRDs, checks public/fallback renders, compares
retry archive bytes and verifies that staged inputs remain unchanged. It does
not install anything in a cluster or promote any image.
