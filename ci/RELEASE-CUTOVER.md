# Coordinated release cutover — blocked until explicitly approved

This is an operational checklist, **not authorization to execute promotion**.
The cleanup PR changes checked-in configuration only. It does not pause deployed
Concourse jobs, remove SPS triggers, or prove that production publication works.

## Before disabling the existing release path

- [ ] Operator release-build and charts support PRs are merged and both source
  branches can be frozen from `main`; charts changelog is ready.
- [ ] Promotion implementation is reviewed; credentials, host access, build tools,
  required registry platforms and actual lowest/latest clusters are verified.
- [ ] Fresh build, all four candidate-bound E2E scenarios and promotion-validation
  are qualified on the intended source SHAs. Historical combined-branch runs do
  not qualify rewritten commits automatically.
- [ ] Isolated production publication/recovery tests cover conflicts and retries.
  Validation-only mode cannot prove production write permissions or certification.
- [ ] An owner explicitly approves the production window and rollback/recovery
  plan. SPS production promotion concurrency is limited to one.
- [ ] Record deployed legacy job/trigger configuration so it can be restored if
  cutover is aborted **before** any new publication occurs.

## Production window — never run both publisher families

1. Pause the deployed Concourse OLM publisher and disable the charts main/manual
   release triggers. Confirm no legacy publication is already in flight. Keep PR
   compliance and E2E validation pipelines enabled.
2. Select the exact final `release-state-path`; verify frozen `main` SHAs, digest,
   artifacts, predecessor and four E2E passes. No rebuilding or latest pointer.
3. Only with explicit approval, run production promotion for that candidate.
4. Verify required destinations and the promotion receipt: versioned/latest image
   digests, public Helm repositories, GitHub attachments, Red Hat preflight and
   all three catalog PR URLs. Track catalog acceptance separately from submission.
5. If publication partially fails, leave legacy publishers disabled and retry
   **the same candidate**. Do not restore the old publisher or overwrite conflicting
   content under the same version. Escalate conflicts for manual recovery.

## Cleanup merge gate

- [ ] New production promotion has succeeded and live publications were checked.
- [ ] The ETCD CA fix is included in the latest public operator; otherwise charts
  standalone CI must retain its old-public-operator compatibility assertion.
- [ ] Owners agree rollback through the legacy publisher is no longer needed.
- [ ] Merge both `release-cleanup-unified` PRs in the agreed window. Apply the
  checked-in Concourse configuration and confirm obsolete deployed triggers stay
  disabled. Source changes alone are not deployment changes.
- [ ] Confirm standalone charts PR/local builds use the development default and
  release builds still supply an explicit version. Preserve original combined
  branches and the recorded candidate evidence for investigation.

The operator cleanup deliberately retains the rest of
`ci/public-pr-creation-and-e2e-image-build.yaml`: it contains unrelated PR/image
jobs. Only the legacy OLM auto-trigger is retired. Do not delete shared scripts or
configurations merely because their names look old.
