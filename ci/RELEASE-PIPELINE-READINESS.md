# Unified release pipeline: review and production readiness

> **Historical review snapshot.** The findings below describe the implementation
> before the follow-up hardening. The agreed retry policy is now “one successful
> attempt per exact candidate”, not latest-attempt success; Slack/compliance additions
> were declined, and SPS now owns OLM PR publication. See
> [RELEASE-PIPELINE-DESIGN.md](RELEASE-PIPELINE-DESIGN.md) for the current implemented
> contract, changes, tests and remaining external cutover checks.

Review date: 2026-11-02. Scope: the checked-out operator and sibling
`instana-agent-charts` repositories, the original release design, and all five
files in `../logs`. No promotion, registry writes, cluster tests, git pushes, or
toolchain configuration changes were performed during this review.

## 1. Decision

**The build happy path works. It is not yet a sufficiently isolated, fail-closed
release-candidate factory. Promotion is not ready to run.**

The downloaded logs show real successful unit tests, four-architecture image
construction, validated OLM generation, chart packaging from the local operator
manifest, four real E2E suites, COS writes, GitHub statuses, and lock releases.
They do not prove retry safety, cancellation handling, candidate immutability,
promotion task admission, production credentials, or public artifact correctness.

Keep the build available for development validation. Before relying on it for
release approval, address findings B1–B5 below and explicitly decide the coverage
and compliance policy. Do not use a green entrypoint run as the release gate.

## 2. Exact candidate and evidence

| Item | Observed value |
|---|---|
| Operator source | `29d252586dbc8cffadfebcf89568b5d68772ca83` |
| Charts source | `61022632af50935ff50b14a532bc213c6a3ca3c3` |
| Both source branches | `ci/unified-release-pipeline`, **not `main`** |
| Candidate version | `2.2.24`; operator tag would be `v2.2.24` |
| Bucket | `instana-agent-operator-temp-artifacts` |
| Candidate prefix | `pipeline-state/release-2.2.24/202610301402` |
| Operator index digest | `sha256:d43a65e339fac484d633367696f1aa50e1eaa6b36463c94dfd41d3a6db8afac7` |
| OLM predecessor selected | `2.2.23` |
| BOM run ID | `simple-pipelinerun` — not a useful unique run identity |

The local source HEADs match the SHAs in the logs. A read-only public GitHub API
lookup returned latest release `v2.2.23`, published 2026-10-30; `git ls-remote`
also showed `v2.2.23` as the highest remote `2.2` tag. Local operator tags stop at
`v2.2.22`, so local tag inspection alone would give the wrong next version.
Charts local tags reach `2.0.54`; no remote GHE release lookup was performed.
`2.2.24` is an observed candidate, not a permanent reservation of that version.

### Main entrypoint

Evidence in `../logs/execute-main.log`:

- Lines 14–18: release properties and test branches.
- Lines 48, 78, 80–116: frozen SHAs, tag discovery, computed version and BOM.
- Unit-test package results finish successfully by line 551 (search `ok` for the
  individual packages rather than relying on the stage exit code).
- Lines 1491–1527: image digest, controller YAML, OLM predecessor and successful
  `operator-sdk bundle validate`; missing API-example annotations are warnings.
- Lines 1571–1588: local `OPERATOR_YAML` consumed, chart version/appVersion both
  `2.2.24`, dev image selected, chart lint passed.
- Lines 1593–1628: three artifacts and both BOM copies uploaded successfully.
- Lines 1630–1664: all four dispatches attempted and entrypoint exit zero;
  the corresponding child logs independently demonstrate actual execution.

### Webhook E2E runs

Top-level test counts exclude subtests. No top-level FAIL or SKIP markers were
found in these four logs.

| Log in `../logs` | Mode / cluster | Passed tests | Go suite time | Result evidence |
|---|---|---:|---:|---|
| `run-stage-ocp-fyre-latest.log` | operator / OCP latest | 28 | 2858.276s | result 2409; COS 2422/2425; GitHub 2459 |
| `run-stage-gke-operator-lowest.log` | operator / GKE lowest | 28 | 2902.925s | result 2454; COS 2467/2470; GitHub 2504 |
| `run-stage-stage-e2e-helm-ocp-lowest.log` | Helm / OCP lowest | 32 | 1690.185s | result 3377; COS 3390/3393; GitHub 3427 |
| `run-stage-gke-operator-latest-helm.log` | Helm / GKE latest | 32 | 2038.812s | result 3566; COS 3579/3582; GitHub 3616 |

All four report the same SHAs and digest at lines 34–36. Both Helm runners
downloaded `instana-agent-2.2.24.tgz` from the candidate, rather than rebuilding
from the latest public release. Upgrade tests include old v1 and latest chart
installs followed by candidate upgrades; version-label assertions passed.
The latest charts changes correctly avoid accidentally retaining the previous
operator image through Helm's reuse-values behavior.

All four logs contain confirmed reslock release messages. Retry messages while
waiting for OpenTelemetry ports, missing resources during cleanup, and one GKE
Helm uninstall API-server error were non-fatal; subsequent tests and final cleanup
completed. The latter merits monitoring, not a claim that these suites failed.

### Actual coverage, not cluster names

- OCP lowest nodes: Kubernetes `v1.29.14+7b5d27f`, RHCOS 4.16 build.
- OCP latest nodes: Kubernetes `v1.35.6`, RHCOS / CRI-O 4.22 build.
- GKE lowest: `v1.35.8-gke.1439001`.
- GKE latest: `v1.35.8-gke.1796000`.

**Both GKE clusters run the same Kubernetes minor/patch.** This does not establish
lowest-versus-latest Kubernetes support. Define the intended minimum, provision
or select an appropriate cluster, and assert/report versions at test start.
Four platforms being built does not establish runtime E2E coverage on arm64,
s390x, or ppc64le. There is also no OLM install/upgrade test in this matrix, only
bundle validation; decide whether an OLM smoke/upgrade gate is required.

### Limits of this evidence

These are downloaded step logs, not a live audit of COS retention/IAM, registry
contents, toolchain trigger filters, or all Tekton task conditions. Successful
upload responses are evidence of upload at run time, not independent checksum
verification today. The generic log parser misses timestamp-prefixed E2E summary
lines, so the explicit Go results and status responses were checked separately.

## 3. Build findings and required changes

### B1 — High: E2E and staged artifacts are not digest-bound

Locations: `ci/sps-scripts/build-and-stage.sh:131–168,203–208` and
`ci/sps-scripts/e2e-async.sh:61–65,230–231,283–299`;
`e2e/config.go:93–119`; charts `ci/build.sh:83–91`.

The BOM records an immutable index digest, but YAML/OLM/chart defaults use the
mutable version tag `:2.2.24`. Operator E2E uses the SHA tag via `GIT_COMMIT`, not
the digest it logs. A later build before promotion computes the same version and
can overwrite the version tag. Rebuilding the same commit can also overwrite its
SHA tag with different version metadata, base layers, or build-date labels.
Thus Helm E2E may test another candidate while promotion copies the original BOM
digest. Capturing the digest by inspecting a shared tag after push is another
race. A frozen source SHA alone is not an immutable image identity.

Required: capture the pushed index digest with Buildx metadata; bind every staged
deployment and test to that digest. Support digest syntax in the operator test
harness and preserve chart digest defaults, or use an unmodifiable candidate tag
with explicit resolved-digest verification. Retain the intended semantic version
as labels and Chart/appVersion, independently of image selection. Record artifact
SHA-256 values and all relevant agent-image inputs in the BOM. Assert that the
actual running operator image IDs belong to the recorded multiarch index.

### B2 — High: retry failures can leave an old success eligible for promotion

Locations: `e2e-async.sh:20,39–59,107–108,168–174,243–261,326–347` and
`validate-release.sh:99–124`.

No pending/running result invalidates the previous success. Login, BOM/clone,
setup, or chart-download failure on a retry can leave the previous success JSON
untouched. Promotion only checks `.status == "success"`, so it can approve while
a newer attempt is running or has already failed before writing a result.
GitHub contexts also omit candidate/attempt identity and receive no pending state.

The runner's “intentionally no -e” comment is misleading: `set -uo pipefail`
does not disable an inherited `-e`; sourcing `installGolang.sh` enables `-euo`,
and both setup scripts enable `-e` and disable `-u`. Test failures are explicitly
captured with `||`, but other failures can exit before final evidence upload.

Required: register a unique attempt and invalidate success before work; record
candidate identity, both SHAs, digest, mode, attempt ID, start/end times, cluster
versions and log checksum. Centralize finalization for ordinary failures and
signals. Explicitly control shell options across sourced scripts. Make critical
COS writes fail the task, but attempt independent evidence writes even if one
fails. Validator must reject running/stale/mismatched attempts and verify evidence
exists. Use attempt ordering/conditional updates so concurrent retries cannot
overwrite a newer result with an older one. If COS is unavailable at attempt
registration, fail closed through the release coordinator, not an old success.

### B3 — High: candidate namespace is only minute-unique

Locations: `prepare.sh:143–154,195–209`, `build-and-stage.sh:287–299`.

Two builds started in the same minute for the same version overwrite the same
candidate BOM/artifacts/status namespace. The design says “precision to the
second” but its format and implementation have only minute precision. The BOM's
`simple-pipelinerun` ID does not disambiguate actual executions. Overlapping builds
can also finish out of order and move `latest-release-candidate` backwards; the
pointer is published once while the BOM is still incomplete.

Required: use UTC seconds plus an actual SPS run UUID (or a generated candidate
UUID), enforce candidate immutability after staging, and record the real run URL.
Expose a completed-candidate pointer only after all staged files validate; define
whether “latest” means latest started, latest built, or latest fully tested.
Serialize candidate selection/update or use conditional writes. QA/promotion
should select an explicit scoped BOM, not rely on a mutable convenience pointer.

### B4 — Medium: failed E2E dispatch is reported as build success

Location: `build-and-stage.sh:314–319`.

Each `trigger-task` failure is reduced to a warning. Missing status files normally
stop promotion, but the entrypoint can still look green without its promised
test coverage, and no completion coordinator/notification exists in these scripts.

Required: attempt all four dispatches, collect failures and child run identities,
then fail the dispatch stage if any required child could not start. Persist the
dispatch matrix; surface an aggregate tested/failed/pending candidate state.
Distinguish “artifacts staged / E2E dispatched” from “candidate ready for QA”.
Add the design's QA-ready and failure Slack notifications, without implying that
dispatch completion means E2E completion.

### B5 — Medium: OLM predecessor is a separate, weakly selected live input

Location: `build-and-stage.sh:182–194`.

The tags API's first page and first matching entry are not a stable release-line
predecessor contract. Dots in the regex are unescaped, no semantic maximum is
calculated, and tags need not correspond to a released OLM bundle. The result
happened to be correct (`2.2.23`) in this run. `prepare.sh:122–133` also admits
prerelease/malformed patch suffixes into shell arithmetic.

Required: filter exact stable SemVer tags, handle leading-zero/invalid inputs,
choose and freeze the intended **published OLM predecessor** during preparation,
and verify the catalog's upgrade chain at release. Handle the first release on
a line explicitly; crossing a minor line may need the previous line's last CSV.
Record the predecessor in the BOM and revalidate it before publication. Do not
derive it from whatever tag happens to be first in a later live API response.

### B6 — Coverage/compliance and lock policy need explicit sign-off

The actual `ci/.pipeline-config-release-build.yaml` uses `simple-execute`, not
the design's `code-setup`/`code-build`/`code-checks` graph. There is no release-time
detect-secrets, static scan, compliance task or signing in this configuration.
Relying on PR pipelines is a policy choice, not evidence that these frozen test
branch commits passed protected-main controls. Require approved/main source and
the relevant checks, and obtain agreement on SBOM, vulnerability, signing and
provenance requirements before replacing the compliance-oriented pipeline.

`reslock.sh` grants a 70-minute lease while Helm permits 75-minute tests; setup
and cleanup can add more time. Current runs fit the lease, but a slow run can
outlive exclusivity. Renew the lease or cap the entire locked section below its
TTL with margin. The EXIT trap is useful but is not a guarantee for SIGKILL or
pod deletion. Add TERM/INT handling, test graceful cancellation, retain lease
expiry as the fallback, and alert on release failure. The release helper currently
clears `RESLOCK_CLAIMED` before it knows release succeeded, preventing a trap retry.
`reslock.sh:23–25` also downloads and executes a helper without `curl --fail` or
an integrity check: a failed/empty download must not be accepted as lock ownership.

### B7 — Charts standalone compatibility has regressed

Charts `ci/build.sh:12–16` requires `CHART_VERSION`; the version file is gone.
Its PR E2E wrapper now injects a dev version, which is correct. However,
`ci/sps/build.sh:12–13` still calls `make build` without supplying a version,
and its later legacy publish code still reads/bumps the deleted file.
Removing Go installation from charts setup also leaves the old PR E2E route
depending on whatever Go is already in its base image; only the unified runner
explicitly installs it.

Required: either retire the old charts main build and its unreachable publish
implementation completely, or supply dev versions and a supported Go toolchain
to every remaining caller. Keep local `make build` usage documented with
`CHART_VERSION`. A deploy-release echo stub alone is not a complete migration.
The actual charts PR OCP cluster is **latest**, not lowest as the old design says.
No charts pipeline currently runs on push in the user's configuration; that
does not make these paths correct if PR/manual triggers remain enabled later.

## 4. Promotion blockers and implementation plan

### P1 — First make the orchestration executable, without publishing

`ci/.pipeline-config-release-promotion.yaml:16–92` still declares custom task
names with `runAfter`, the exact pattern the design says SPS rejects. The build
already escaped this constraint with `simple-execute`. There are no supplied
promotion logs proving that this graph is admitted.

Prefer one supported `simple-execute` sequence for validate → registry staging →
preflight → repackage → publish, or map to supported SPS slots and prove the
rendered graph. First test admission with inert scripts and mocked destinations,
not the real promotion scripts. For each fresh container explicitly authenticate
to IBM COS and install/check required tools. Unlike build preparation/E2E,
promotion scripts contain no explicit IBM login; do not assume credentials or
CLI sessions persist across tasks. Preflight's inline script also lacks strict
error handling. Export/import selected state explicitly between supported steps.

### P2 — Fail closed before any public write

Extend `validate-release.sh` from nonempty-field checks to a schema and policy gate:

1. Require an explicitly approved candidate BOM path and display its identity,
   SHAs, digest, version and evidence to the release owner. The current default
   chooses latest at trigger time, which may not be the candidate QA approved.
2. Enforce `main` for both source branches under the stated scope; verify actual
   ancestry/approved checks, not just the BOM's branch strings. Currently a clean
   `X.Y.Z` version from the development branch can still reach public registries;
   only the `latest` copy has a main-branch condition.
3. Validate version/tag consistency, exact SHA/digest formats, supported platforms,
   artifact keys beneath the selected prefix, existence, size and checksums.
4. Require all four latest completed attempts to match candidate inputs, with no
   newer running attempt. Retest or explicitly expire old evidence as policy
   requires. Disable/strongly authorize and audit `skip-e2e-check` for production.
5. Ensure neither repo has a conflicting release tag, and that no destination
   already contains different bytes under this version. Recheck at publication
   under a release lock, since validation alone is not a reservation.
6. Freeze the selected BOM/evidence by hash or immutable snapshot for the entire
   promotion. Current scoped-path pinning usefully avoids the latest-pointer
   race, but does not prevent mutation at that scoped key.

Serialize releases globally (or at least per release line plus shared publication
destinations). Prevent an older candidate from downgrading `latest`, losing Helm
index updates, or racing tag creation. Disable conflicting legacy publishers.

### P3 — Separate image staging from public activation

`promote-images.sh` correctly uses the BOM digest as source, `skopeo copy --all`,
and separate source/destination credentials. Retain that. Verify each resulting
multiarch index and platform image, and detect manifest conversion: public YAML
cannot assume a digest survives a registry copy unless preservation is enforced
or the destination digest is recorded and substituted.

Current order publishes public ICR version **and `latest`** before Red Hat
preflight. A later failure is already customer-visible. Safer order:

1. Copy the candidate to the Red Hat validation destination/version required by
   preflight and any approved non-discoverable staging destinations.
2. Run preflight with authenticated pull access (`--docker-config` or equivalent,
   as in the legacy pipeline), Pyxis token and pinned/available binary. Persist
   scan artifacts and submission IDs. Verify architecture policy explicitly.
3. After scan success, prepare/validate public release files; copy verified
   versioned images to required public/internal destinations.
4. Update `latest` only at the chosen final commit point after all required gates,
   with monotonic version checks and recovery rules.

No multi-service transaction exists. Explicitly define which partial publication
states are acceptable and how an operator resumes each. A failure after an image
copy must not invite rebuilding or overwriting the same released version.

### P4 — Restore promoted build-tool images, not staged candidate images

The unified YAMLs currently use
`icr.io/instana-agent-dev/agent-delivery/build-image:main`. Before production,
replace these with the **confirmed promoted build-image repository and immutable
digest** wherever used, including async entrypoints and promotion steps. The
promoted location is an environment decision; do not guess a namespace.

This is separate from the operator under test: release-build/E2E must continue
using private dev candidate images. Only public copies of the chart/operator YAML/
OLM should reference the promoted operator repository. Do not globally replace
all dev references in the build pipeline.

Verify the promoted tool image has bash, jq, git, Python/YAML, Docker/Buildx,
skopeo, IBM CLI/COS plugin, yq v4, compatible Helm, zip/unzip, gcloud/gsutil and
preflight. Pin tool versions and record tool-image/config-repo revisions in the
BOM. Test the exact promoted image in a **nonpublishing** build/E2E run; a dev
image passing is not proof the promoted image has the same tools/behavior.

### P5 — Validate transformed artifacts, including fallback paths

`repackage-helm.sh` already preserves staged COS files, patches the chart image
defaults and rewrites operator references in YAML/OLM. These are good boundaries,
but are not a completed public-artifact test.

- Render the packaged chart using real Helm/yq with defaults, partial/empty image
  maps, supported overrides, and upgrade behavior. The chart template's fallback
  currently remains `icr.io/instana/instana-agent-operator:2.2.24` during dev tests,
  because charts `ci/build.sh` only changes its tag. The normal values path is dev,
  but empty-map fallback would pull a not-yet-published public version. Test and
  explicitly define both candidate and public fallback behavior.
- Unpack all outputs; reject dev namespaces, placeholders, malformed image refs,
  unexpected private pull-secret requirements and wrong version labels. Preserve
  legitimate private-registry override support for customers.
- Validate public OLM again, inspect CSV `replaces`, related images and CRDs, and
  perform the agreed OLM install/upgrade smoke test. Simple namespace rewriting
  alone does not certify catalog compatibility.
- Compare transformation diffs: only approved image/provenance changes should
  differ from the tested chart/manifests. Record staged/public checksums and a
  promotion receipt. Smoke-test public pull access without CI credentials when
  versioned copies become available.

### P6 — Resolve the changelog/frozen-source contradiction

The design asks QA to edit `canonical/CHANGELOG.md` **after build**. That file is
already inside the staged chart, while publication tags the old `charts-sha`.
A later charts push neither changes the staged archive nor triggers a build.
The release would therefore omit the new changelog and tag the old content.

Preferred policy: finish release notes/changelog before the final candidate build;
explicitly trigger that final build after the charts update, then QA approves the
new candidate. Alternatively freeze a separate approved documentation revision,
allow only a documented changelog-only transform during promotion, record both
revisions/checksums, and decide which SHA the charts release tag represents.
Do not silently rebuild from charts HEAD during promotion. GitHub auto-generated
notes only cover operator history and are not a substitute for chart changes.

### P7 — Make publishing resumable and avoid incomplete public releases

Locations: `publish-helm.sh:73–92,107–117,128–134,166–191`.

Current tag creation/push fails on retry when a tag already exists; fixed clone
directories can also collide on a restart. GitHub release POST fails if the
release already exists. Partial GCP/Artifactory/public-repo writes have no receipt.

Implement idempotent operations: an existing tag is acceptable only when it
resolves to the frozen SHA; an existing artifact only when checksums match.
Use isolated workdirs and persist per-destination completion records. Stage git
changes before checking `git diff --cached`, or inspect porcelain status:
`git diff --quiet` alone misses newly added/untracked chart files.

Upload the GCP chart first and publish the merged index last under serialization
or generation preconditions; glob-copying both has no explicit safe ordering.
Preserve existing versions and verify URL/digest entries after publication.
Verify Artifactory indexing and public-repo contents rather than just HTTP 2xx.
Use signed-off commits (`git commit -s`) for generated repository commits too.

Create/resume a **draft** GitHub release, upload and verify all three assets, then
publish it. The current `draft:false` creates a visible release before uploading
OLM, YAML and chart. A failure or downstream watcher can observe incomplete
contents. Draft-first is a recommended change to the original design, not current
behavior. Ensure retries do not duplicate downstream publication or replace
already released bytes. Add final success/failure Slack notifications and links.

### P8 — Keep and verify the downstream OLM publisher

`ci/public-pr-creation-and-e2e-image-build.yaml:75–80,170–177` confirms the existing
Concourse job watches public GitHub releases and downloads `olm-*.zip`. Retaining
`olm-<version>.zip` and its current archive layout is compatible in principle.
The job also rejects versions older than GitHub's latest release, so “re-publish
an older candidate” does not imply its catalog PRs will be created.

Do not decommission this pipeline with `ci/pipeline.yaml`. Verify it is deployed,
unpaused, authenticated, and still able to create all three catalog PRs. Observe
the first real release end to end; the current build logs cannot establish this.
Keep the release-tag source consistent with the attached artifact and check
bundle-specific registry/metadata rewrites performed by the existing PR scripts.

## 5. Configuration and secret checklist

Configure properties in SPS; YAML comments do not enforce trigger behavior.

| Area | Required confirmation |
|---|---|
| Triggers | Build currently responds to operator pushes per user setup; disable that for production manual-only operation; retain four authenticated internal webhook triggers. Keep promotion manual, no timer initially. Check tag-push filters too. |
| Source/config | Production operator/charts branches `main`; record pipeline-config revision independently of application SHAs; protected source and approval/check policy. |
| Version | Required `release-line=2.2` or explicitly approved line; no hard-coded next patch; check both repo tags and public destinations. |
| Candidate selection | Explicit scoped BOM for QA and promotion, not a moving latest pointer; actual run UUID/URL. |
| COS | Bucket/region, explicit authentication in fresh containers, read/write separation, encryption and lifecycle retention long enough for QA/retries/audit; protect approved BOMs/evidence from mutation. |
| Image credentials | `/config/api-key` dev read/write and public ICR authorization; `icr-rel-api-key` JSON (`username`, `password`); `redhat-registry` JSON (`username`/`password` for registry login, `token` for Pyxis submission); no assumption that one account grants every namespace. |
| Source/status credentials | `github-token`, `ghe-token`; promotion requires protected tag/repo/release rights on these tokens. |
| E2E credentials | Four cluster secrets, `gcp-service-account`, `instana-e2e-backend-details`, `reslock-github-token`, private ICR pull access recreated after namespace cleanup. |
| Publication | `gcp-bucket-write-agents-io-service-key`, `taas-artifactory-read-writer-creds`, `redhat-registry` (including Pyxis token), Red Hat pull authentication, Slack secret/notification implementation. |
| Tool images | Replace dev build-tool image with confirmed promoted digest and verify tooling before cutover. |

Disable `PIPELINE_DEBUG` in credentialed production runs: `set -x` and `trap env
EXIT` appear throughout the scripts and can expose tokens/passwords stored in
variables or command arguments. Replace environment dumps with an allowlisted
diagnostic view; avoid credentials embedded in clone URLs and scrub temporary
credential files/configs. COS log paths are authenticated object identifiers, not
public download links; provide safe authenticated access for QA.

## 6. Acceptance tests before enabling production promotion

These tests are requirements, **not actions performed during this review**.

1. Two same-minute builds, different source SHAs: different prefixes and immutable
   image identities; old candidate still deploys its own image. Repeat same SHA
   with different version/base image and verify no candidate is reinterpreted.
2. Retry E2E after success: promotion rejects pending, setup/login/download errors,
   test failure, SIGTERM, missing log/status upload and stale/out-of-order attempts.
3. Fail reslock download/claim/release; verify no unowned test work, alerting,
   cancellation cleanup and lease renewal/expiry. Exercise a >70-minute run.
4. Fail one webhook dispatch: other three dispatch attempts still happen, overall
   dispatch fails, candidate cannot become QA-ready. Restart only one child and
   confirm the same scoped BOM/test harness revisions are retained.
5. Version fixtures: no tags, multiple lines, prerelease tags, invalid patch text,
   chart tag conflicts, missing predecessor and a new minor's OLM upgrade chain.
6. Inert SPS promotion graph admission test; then local/mocked service tests for
   all gates, no public writes. Validate fresh-container authentication and env
   handoff with the exact promoted tool image.
7. Real Helm/yq public repackaging tests, schema/CRD/CSV validation and checksum
   comparisons; confirm no dev operator refs in any final default/fallback output.
8. Failure injection after each publication boundary; retry succeeds only for
   identical existing bytes/SHAs, never force-tags/overwrites an existing release.
9. Concurrent promotion / Helm index writers / latest-tag downgrade: blocked or
   safely serialized. GitHub release stays draft until all assets are verified.
10. Final main-branch candidate with completed chart changelog, required support
    matrix and compliance evidence. Approve its exact scoped identity. Only then
    schedule the first manual production promotion and downstream observation.

### Checks actually run in this review

- Existing six offline release-script regression tests:
  `python3 -m unittest discover -s ci/sps-scripts -v` — **6 passed**.
- All five logs parsed with `ci/sps-scripts/parse-tekton-logs.sh`, followed by
  targeted source/log inspection and top-level test/result counting.
- Read-only public release/tag lookups for current operator version.

The regression tests cover scoped BOM reads, mocked registry-copy arguments,
frozen checkout, tag credential/host selection and fixture repackaging. They do
not test real SPS admission, real Helm/yq behavior (mocked in repackaging), E2E
failure handling, cloud authentication or publication idempotency.

## 7. Suggested cutover order

1. Agree release policy, minimum cluster versions, promoted tool image, and the
   changelog rule; implement B1–B5 and the retained charts entrypoint fixes.
2. Finish promotion orchestration/gates, safe publication ordering and retry
   receipts; pass the nonpublishing acceptance tests above.
3. Merge charts and operator changes in a coordinated window. A charts push does
   not trigger testing; an operator push does. Do not push operator review changes
   merely to update documentation without expecting another build. No pushes
   were made for this review.
4. Disable duplicate legacy release writers (including charts legacy build-script
   publishing), but retain the downstream OLM PR pipeline and FedRAMP handling.
   Change production triggers/properties explicitly; keep schedules off.
5. Build one final candidate from both `main` branches using promoted build tools;
   collect all E2E results and QA approval against its exact immutable identity.
6. At a separately approved release window, run promotion once, monitor every
   destination and downstream catalog PR, and retain the promotion receipt.

Until those steps are complete, the supplied `2.2.24` run is strong evidence for
development build functionality, **not authorization or proof of production
release readiness**.
