"""Offline release regression tests. No cloud credentials, network or clusters.

Run: PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s ci/sps-scripts -v
"""

import importlib.util
import io
import hashlib
import json
import os
from pathlib import Path
import subprocess
import shutil
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile

from release_api import GitHub, check_not_older_release, discover, ensure_release, ensure_tag, semver
from release_state import validate_bom, validate_success
from publish_olm import bundle_files, publish, verify_catalog_files


SCRIPTS = Path(__file__).resolve().parent
VERSION = "2.2.24"
CANDIDATE = "20261102123045-12345678-1234-1234-1234-123456789abc"
STATE = f"pipeline-state/release-{VERSION}/{CANDIDATE}"
DEV = "icr.io/instana-agent-dev/instana-agent-operator"
PUBLIC = "icr.io/instana/instana-agent-operator"


def bom():
    return {"version": VERSION, "release-tag": f"v{VERSION}", "previous-version": "2.2.23",
            "candidate-id": CANDIDATE, "cos-state-path": STATE, "build-complete": True,
            "source-branch": "main", "charts-branch": "main",
            "sources": {"operator-sha": "a" * 40, "charts-sha": "b" * 40},
            "images": {"operator-manifest": f"{DEV}:{VERSION}-{CANDIDATE}",
                       "operator-manifest-digest": "sha256:" + "c" * 64},
            "artifacts": {"helm-chart": f"{STATE}/helm/instana-agent-{VERSION}.tgz",
                          "olm-bundle": f"{STATE}/olm/olm-{VERSION}.zip",
                          "operator-yaml": f"{STATE}/olm/instana-agent-operator.yaml"},
            "checksums": {k: "d" * 64 for k in ("helm-chart", "olm-bundle", "operator-yaml")}}


class FakeAPI:
    def __init__(self, tags=(), releases=()):
        self.tags, self.releases = tags, releases

    def pages(self, path):
        if path.endswith("/tags"):
            return [{"name": t} for t in self.tags]
        return self.releases


def release(version, draft=False, prerelease=False):
    return {"tag_name": f"v{version}", "draft": draft, "prerelease": prerelease,
            "assets": [{"name": f"olm-{version}.zip"}]}


class VersionsTest(unittest.TestCase):
    def test_stable_semver_only(self):
        for value in ("2.2.24-rc1", "02.2.24", "2x2x24", "2.2.24.1", "2.2.24+build"):
            self.assertIsNone(semver(value))
        self.assertEqual(semver("v2.2.24"), (2, 2, 24))

    def test_api_semantic_maximum_and_published_predecessor(self):
        result = discover(FakeAPI(["v2.3.99", "v2.2.8", "v2.2.23", "v2.2.99-rc1", "v2.2.x"],
                                  [release("2.2.8"), release("2.2.23"), release("2.2.24", draft=True)]),
                          FakeAPI(["2.0.54", "2.0.9"]), "2.2")
        self.assertEqual(result, {"version": VERSION, "previous-version": "2.2.23",
                                  "latest-charts-tag": "2.0.54"})

    def test_new_minor_preserves_previous_line_upgrade_edge(self):
        result = discover(FakeAPI(["v2.2.23"], [release("2.2.23")]), FakeAPI(), "2.3")
        self.assertEqual(result["version"], "2.3.0")
        self.assertEqual(result["previous-version"], "2.2.23")

    def test_charts_tag_conflict(self):
        with self.assertRaises(ValueError):
            discover(FakeAPI(["v2.2.23"]), FakeAPI([VERSION]), "2.2")

    def test_invalid_release_line(self):
        for line in ("02.2", "2.02", "2.2.0", "2x2", ""):
            with self.assertRaises(ValueError):
                discover(FakeAPI(), FakeAPI(), line)

    def test_api_pagination(self):
        api = GitHub("https://fixture.invalid", "fixture")
        calls = []
        def request(path):
            calls.append(path)
            return list(range(100)) if path.endswith("&page=1") else [100]
        api.request = request
        self.assertEqual(len(list(api.pages("/tags"))), 101)
        self.assertEqual(len(calls), 2)

    def test_auth_failure_does_not_fall_back_to_local_tags(self):
        api = FakeAPI()
        api.pages = lambda path: (_ for _ in ()).throw(RuntimeError("HTTP 401"))
        with self.assertRaisesRegex(RuntimeError, "401"):
            discover(FakeAPI(), api, "2.2")

    def test_older_candidate_cannot_downgrade_public_release(self):
        check_not_older_release(FakeAPI(releases=[release(VERSION)]), VERSION)
        with self.assertRaises(ValueError):
            check_not_older_release(FakeAPI(releases=[release("2.3.0")]), VERSION)


class StateTest(unittest.TestCase):
    def test_exact_candidate(self):
        validate_bom(bom(), STATE + "/bom.json")

    def test_rejects_mutable_pointer_wrong_branch_and_incomplete_build(self):
        for change in ({"source-branch": "dev"}, {"charts-branch": "dev"}, {"build-complete": False},
                       {"release-tag": "v2.2.23"}, {"candidate-id": "minute-only"}):
            value = bom() | change
            with self.assertRaises(ValueError):
                validate_bom(value, STATE + "/bom.json")
        with self.assertRaises(ValueError):
            validate_bom(bom(), "pipeline-state/latest-release-candidate/bom.json")

    def test_rejects_artifact_outside_candidate(self):
        value = bom()
        value["artifacts"]["helm-chart"] = "another-candidate/chart.tgz"
        with self.assertRaises(ValueError):
            validate_bom(value, STATE + "/bom.json")

    def test_success_must_match_inputs(self):
        value = bom()
        result = {k: value[k] for k in ("candidate-id", "sources", "images", "checksums")}
        result |= {"status": "success", "cluster": "ocp-fyre-latest", "mode": "operator",
                   "log": f"{STATE}/e2e/ocp-fyre-latest/test-attempt-success.log", "log-sha256": "e" * 64}
        validate_success(value, result, "ocp-fyre-latest")
        for field, replacement in (("images", {}), ("sources", {}), ("status", "failure"), ("mode", "helm")):
            with self.assertRaises(ValueError):
                validate_success(value, result | {field: replacement}, "ocp-fyre-latest")


class TagReleaseTest(unittest.TestCase):
    def test_existing_tag_is_noop_but_conflict_rejected(self):
        api = FakeAPI()
        api.request = lambda *args, **kwargs: {"object": {"type": "commit", "sha": "a" * 40}}
        ensure_tag(api, "instana/operator", "v2.2.24", "a" * 40)
        with self.assertRaises(ValueError):
            ensure_tag(api, "instana/operator", "v2.2.24", "b" * 40)

    def test_annotated_tag_is_peeled(self):
        api = FakeAPI()
        replies = iter([{"object": {"type": "tag", "sha": "tag-object"}},
                        {"object": {"type": "commit", "sha": "a" * 40}}])
        api.request = lambda *args, **kwargs: next(replies)
        ensure_tag(api, "instana/operator", "v2.2.24", "a" * 40)

    def test_draft_release_resumes_and_rejects_conflicting_asset(self):
        with tempfile.TemporaryDirectory(dir="/tmp/opencode") as tmp:
            file = Path(tmp) / "operator.yaml"
            file.write_bytes(b"approved bytes")
            api = FakeAPI()
            calls = []
            def request(path, data=None, **kwargs):
                calls.append((path, data))
                if kwargs.get("raw"):
                    return b"approved bytes"
                return {"id": 1, "draft": True, "html_url": "https://fixture.invalid/release"}
            api.request = request
            api.pages = lambda path: [{"name": file.name, "url": "https://fixture.invalid/asset"}]
            ensure_release(api, bom(), [file])
            self.assertEqual(calls[-1][1], {"draft": False})
            self.assertEqual(sum(1 for _, data in calls if isinstance(data, bytes)), 0)
            file.write_bytes(b"conflict")
            with self.assertRaises(ValueError):
                ensure_release(api, bom(), [file])

    def test_draft_hidden_from_tag_endpoint_is_found_by_listing(self):
        draft = {"id": 1, "draft": True, "tag_name": f"v{VERSION}",
                 "html_url": "https://fixture.invalid/draft"}
        api, writes = FakeAPI(), []
        api.pages = lambda path: [draft] if path.endswith("/releases") else []
        def request(path, data=None, **kwargs):
            if data is not None:
                writes.append((path, data))
            return None
        api.request = request
        ensure_release(api, bom(), [])
        self.assertEqual(writes, [("/repos/instana/instana-agent-operator/releases/1", {"draft": False})])


class CatalogTest(unittest.TestCase):
    def test_existing_branch_creates_pr_once_then_resumes(self):
        prs, writes = {}, []
        class API:
            def pages(self, path):
                return prs.get(path.split("/pulls")[0], [])

            def request(self, path, data=None, **kwargs):
                if data is not None:
                    writes.append(path)
                    pr = {"html_url": "https://fixture.invalid" + path, "state": "open"}
                    prs[path.split("/pulls")[0]] = [pr]
                    return pr
                if "/git/ref/" in path:
                    return {"object": {"sha": "existing-branch"}}
                return {"default_branch": "main"}
        with patch("publish_olm.bundle_files", return_value={"fixture": "approved"}), \
                patch("publish_olm.verify_catalog_files") as verify:
            first = publish(API(), bom(), "unused.zip", "instana")
            second = publish(API(), bom(), "unused.zip", "instana")
        self.assertEqual(first, second)
        self.assertEqual(len(writes), 3)
        self.assertEqual(verify.call_count, 6)

    def test_matching_merged_pr_does_not_recreate_deleted_branch(self):
        api = FakeAPI()
        api.pages = lambda path: [{"merged_at": "2026-11-02", "merge_commit_sha": "merged",
                                   "html_url": "https://fixture.invalid/merged"}]
        api.request = lambda *args, **kwargs: self.fail("Merged PR must not recreate a branch")
        with patch("publish_olm.bundle_files", return_value={"fixture": "approved"}), \
                patch("publish_olm.verify_catalog_files") as verify:
            self.assertEqual(len(publish(api, bom(), "unused.zip", "instana")), 3)
        self.assertEqual(verify.call_count, 3)

    def test_conflicting_catalog_bytes_are_not_overwritten(self):
        import base64
        api = FakeAPI()
        api.request = lambda *args, **kwargs: {"content": base64.b64encode(b"old bytes").decode()}
        with patch("publish_olm.catalog_files", return_value={"fixture": {"sha": "existing"}}):
            with self.assertRaises(ValueError):
                verify_catalog_files(api, "/repo", "commit", "", {"fixture": "different bytes"})


class ShellTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="release-test-", dir="/tmp/opencode")
        self.addCleanup(self.temp.cleanup)
        self.workspace = Path(self.temp.name)

    def bash(self, code, expected=0):
        guards = "\n".join(
            f'{cmd}() {{ echo "Unexpected service command: {cmd}" >&2; exit 99; }}'
            for cmd in ("ibmcloud", "curl", "git", "gsutil", "gcloud", "skopeo", "helm", "dnf", "docker", "get_secret")
        )
        result = subprocess.run(["bash", "--noprofile", "--norc", "-c", "set -euo pipefail\n" + guards + "\n" + code],
                                cwd=self.workspace, env={"PATH": os.defpath, "WORKSPACE": str(self.workspace),
                                                         "HOME": str(self.workspace), "PYTHONDONTWRITEBYTECODE": "1"},
                                capture_output=True, text=True)
        self.assertEqual(result.returncode, expected, result.stdout + result.stderr)
        return result

    def test_tag_digest_mismatch_fails(self):
        self.bash(f'''
source "{SCRIPTS}/release-common.sh"
timeout() {{ shift; "$@"; }}
skopeo() {{ echo sha256:actual; }}
cat() {{ echo fixture-key; }}
verify_tag {DEV}:human-readable sha256:expected
''', expected=1)

    def test_failure_attempt_never_overwrites_success_pointer(self):
        # Execute the actual EXIT finalizer with all service commands guarded.
        text = (SCRIPTS / "e2e-async.sh").read_text()
        finalizer = text.split("finalize_e2e() {", 1)[1].split("trap finalize_e2e EXIT", 1)[0]
        for status, rc in (("success", 0), ("failure", 1), ("success", 143)):
            result = self.bash(f'''
source "{SCRIPTS}/release-common.sh"
release_reslock() {{ return 0; }}
cos_put() {{ echo "PUT $1"; }}
CLUSTER_ID=ocp-fyre-latest E2E_MODE=operator E2E_TIMESTAMP=attempt
COS_STATE_PATH={STATE}
LOG_FILE="$WORKSPACE/log"; echo test > "$LOG_FILE"
BOM_JSON='{json.dumps(bom())}'
E2E_STATUS={status}
finalize_e2e() {{{finalizer}
trap finalize_e2e EXIT
exit {rc}
''', expected=rc)
            self.assertEqual("/e2e/success/" in result.stdout, status == "success" and rc == 0)

    def test_reslock_download_failure_never_claims(self):
        self.bash(f'''
get_secret() {{ echo fixture; }}
curl() {{ return 22; }}
APP_REPO_NAME=operator BRANCH=main PIPELINE_RUN_NAME=fixture BUILD_NUMBER=1
set -- claim cluster
source "{SCRIPTS}/reslock.sh"
''', expected=22)

    def test_image_copy_retains_digest_and_rejects_conflict(self):
        code = (SCRIPTS / "promote-images.sh").read_text().split("copy_version() {", 1)[1].split('case "${PROMOTION_PHASE', 1)[0]
        for remote, rc in (("sha256:expected", 0), ("sha256:conflict", 1)):
            self.bash(f'''
DIGEST=sha256:expected
copy_version() {{{code}
skopeo() {{
  case "$1" in
    list-tags) echo '{{"Tags":["2.2.24"]}}' ;;
    inspect) echo {remote} ;;
    *) echo 'Unexpected copy of existing image' >&2; exit 99 ;;
  esac
}}
copy_version icr.io/instana/operator:2.2.24 fixture
''', expected=rc)

    def test_frozen_checkout_fetches_sha_before_checkout(self):
        for script in ("build-and-stage.sh", "e2e-async.sh"):
            for folder, variable in (("operator-src", "OPERATOR_SHA"), ("charts-src", "CHARTS_SHA")):
                lines = [line.strip() for line in (SCRIPTS / script).read_text().splitlines()
                         if line.strip().startswith(f'git -C "${{WORKSPACE}}/{folder}"')]
                self.assertEqual(len(lines), 2)
                self.assertIn(f'fetch --depth 1 origin "${{{variable}}}"', lines[0])
                self.assertIn(f'checkout --detach "${{{variable}}}"', lines[1])

    def test_build_dispatch_attempts_all_then_fails(self):
        text = (SCRIPTS / "build-and-stage.sh").read_text().split("DISPATCH_FAILED=0", 1)[1]
        result = self.bash('''
E2E_STAGES=(one two three four)
trigger-task() { echo "DISPATCH $1"; [[ $1 != two ]]; }
DISPATCH_FAILED=0
''' + text, expected=1)
        self.assertEqual(result.stdout.count("DISPATCH "), 4)

    def test_promotion_requires_all_pass_pointers_unless_explicitly_forced(self):
        value = bom()
        store = self.workspace / "store"
        for key, path in value["artifacts"].items():
            file = store / path
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_bytes(b"fixture artifact")
            value["checksums"][key] = hashlib.sha256(file.read_bytes()).hexdigest()
        (store / STATE / "bom.json").write_text(json.dumps(value))
        harness = f'''
get_env() {{
  case "$1" in
    release-state-path) echo {STATE}/bom.json ;;
    skip-e2e-check) echo "$FORCE" ;;
    *) echo "${{2:-fixture}}" ;;
  esac
}}
get_secret() {{ echo fixture; }}
set_env() {{ :; }}
export_env() {{ :; }}
ibmcloud() {{
  [[ $1 == login ]] && return 0
  local key='' out="${{@: -1}}"
  while [[ $# -gt 0 ]]; do
    if [[ $1 == --key ]]; then key=$2; shift; fi
    shift
  done
  cp "$WORKSPACE/store/$key" "$out"
}}
source "{SCRIPTS}/validate-release.sh"
'''
        self.bash('FORCE=false\n' + harness, expected=1)
        result = self.bash('FORCE=true\n' + harness)
        self.assertIn("FORCED PROMOTION", result.stdout)
        for cluster, mode in (("ocp-fyre-latest", "operator"), ("gke-operator-lowest", "operator"),
                              ("ocp-fyre-lowest", "helm"), ("gke-operator-latest", "helm")):
            log = f"{STATE}/e2e/{cluster}/test-attempt-success.log"
            log_file = store / log
            log_file.parent.mkdir(parents=True)
            log_file.write_bytes(b"passed")
            result = {k: value[k] for k in ("candidate-id", "sources", "images", "checksums")}
            result |= {"status": "success", "cluster": cluster, "mode": mode, "log": log,
                       "log-sha256": hashlib.sha256(b"passed").hexdigest()}
            pointer = store / STATE / "e2e/success" / f"{cluster}.json"
            pointer.parent.mkdir(parents=True, exist_ok=True)
            pointer.write_text(json.dumps(result))
        self.bash('FORCE=false\n' + harness)


class ArchiveTest(unittest.TestCase):
    def test_normalized_archives_are_repeatable(self):
        spec = importlib.util.spec_from_file_location("normalize", SCRIPTS / "normalize-release-archives.py")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory(dir="/tmp/opencode") as tmp:
            chart, bundle = Path(tmp) / "chart.tgz", Path(tmp) / "bundle.zip"
            with tarfile.open(chart, "w:gz") as archive:
                info = tarfile.TarInfo("instana-agent/Chart.yaml")
                data = b"version: 2.2.24\n"
                info.size, info.mtime = len(data), 123456
                archive.addfile(info, io.BytesIO(data))
            with zipfile.ZipFile(bundle, "w") as archive:
                archive.writestr("metadata/annotations.yaml", "annotations: {}")
            module.normalize(chart, bundle)
            first = chart.read_bytes(), bundle.read_bytes()
            module.normalize(chart, bundle)
            self.assertEqual(first, (chart.read_bytes(), bundle.read_bytes()))

    def test_marketplace_transform_preserves_catalog_contract(self):
        with tempfile.TemporaryDirectory(dir="/tmp/opencode") as tmp:
            archive = Path(tmp) / "bundle.zip"
            with zipfile.ZipFile(archive, "w") as source:
                source.writestr("manifests/instana-agent-operator.clusterserviceversion.yaml",
                                "metadata:\n  annotations: {}\nspec:\n  version: 2.2.24\n")
                source.writestr("metadata/annotations.yaml", "annotations: {}\n")
            files = bundle_files(archive, VERSION, "instana-agent-operator-rhmp")
            csv = next(value for name, value in files.items() if name.endswith(".clusterserviceversion.yaml"))
            self.assertIn("marketplace.openshift.io/remote-workflow", csv)
            self.assertTrue(all("/instana-agent-operator-rhmp/2.2.24/" in name for name in files))


@unittest.skipUnless(os.environ.get("RELEASE_TEST_TOOL_DIR"), "Set RELEASE_TEST_TOOL_DIR for real Helm/yq smoke")
class RealRepackageTest(unittest.TestCase):
    def test_chart_build_defaults_to_dev_and_preserves_candidate_image_tag(self):
        import yaml
        root = SCRIPTS.parents[1]
        charts = root.parent / "instana-agent-charts"
        with tempfile.TemporaryDirectory(dir="/tmp/opencode") as tmp:
            work = Path(tmp)
            shutil.copytree(charts / "canonical", work / "canonical")
            subprocess.run(["git", "init", str(work)], check=True, capture_output=True)
            dev_ref = bom()["images"]["operator-manifest"]
            docs = [yaml.safe_load(p.read_text()) for p in (root / "config/crd/bases").glob("*.yaml")]
            docs.append({"kind": "Deployment", "metadata": {"name": "instana-agent-controller-manager",
                         "labels": {"app.kubernetes.io/operator-version": VERSION}},
                         "spec": {"template": {"spec": {"containers": [{"image": dev_ref}]}}}})
            manifest = work / "operator.yaml"
            manifest.write_text(yaml.safe_dump_all(docs))
            env = dict(os.environ, PATH=os.environ["RELEASE_TEST_TOOL_DIR"] + ":" + os.environ["PATH"],
                       OPERATOR_YAML=str(manifest))
            env.pop("CHART_VERSION", None)
            for chart_version in ("0.0.0-dev", VERSION):
                if chart_version == VERSION:
                    env["CHART_VERSION"] = VERSION
                subprocess.run(["bash", str(charts / "ci/build.sh")], cwd=work, env=env,
                               check=True, capture_output=True, text=True)
                with tarfile.open(work / f"artefacts/instana-agent-{chart_version}.tgz") as archive:
                    values = yaml.safe_load(archive.extractfile("instana-agent/values.yaml"))
                    metadata = yaml.safe_load(archive.extractfile("instana-agent/Chart.yaml"))
                    template = archive.extractfile(
                        "instana-agent/templates/operator_deployment_instana-agent-controller-manager.yml").read().decode()
                self.assertEqual(values["controllerManager"]["image"]["tag"], dev_ref.rsplit(":", 1)[1])
                self.assertEqual(metadata["appVersion"], VERSION)
                self.assertEqual(metadata["version"], chart_version)
                self.assertIn(dev_ref, template)

    def test_real_chart_repackaging_render_and_deterministic_retry(self):
        import yaml
        root = SCRIPTS.parents[1]
        charts = root.parent / "instana-agent-charts"
        with tempfile.TemporaryDirectory(dir="/tmp/opencode") as tmp:
            work = Path(tmp)
            chart = work / "fixture/instana-agent"
            shutil.copytree(charts / "canonical", chart)
            value = bom()
            dev_ref = value["images"]["operator-manifest"]
            values = yaml.safe_load((chart / "values.yaml").read_text())
            values["controllerManager"]["image"] = {"name": DEV, "tag": dev_ref.rsplit(":", 1)[1]}
            (chart / "values.yaml").write_text(yaml.safe_dump(values))
            metadata = yaml.safe_load((chart / "Chart.yaml").read_text())
            metadata.update(version=VERSION, appVersion=VERSION)
            (chart / "Chart.yaml").write_text(yaml.safe_dump(metadata))
            template = chart / "templates/operator_deployment_instana-agent-controller-manager.yml"
            template.write_text(template.read_text().replace("OPERATOR_VERSION_PLACEHOLDER", VERSION)
                                .replace(f"{PUBLIC}:latest", dev_ref))
            (chart / "crds").mkdir(exist_ok=True)
            for file in (root / "config/crd/bases").glob("*.yaml"):
                shutil.copy(file, chart / "crds" / file.name)
            manifest = "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: instana-agent-controller-manager\n"
            manifest += f'  labels:\n    app.kubernetes.io/operator-version: "{VERSION}"\n'
            manifest += f"spec:\n  template:\n    spec:\n      containers:\n        - name: manager\n          image: {dev_ref}\n"
            staged_yaml = work / "staged.yaml"
            staged_yaml.write_text(manifest)
            staged_zip = work / "staged.zip"
            csv = {"kind": "ClusterServiceVersion", "metadata": {"annotations": {"containerImage": dev_ref}},
                   "spec": {"version": VERSION, "replaces": "instana-agent-operator.v2.2.23", "install": {"spec": {
                       "deployments": [{"spec": {"template": {"spec": {"containers": [{"image": dev_ref}]}}}}]}}}}
            with zipfile.ZipFile(staged_zip, "w") as archive:
                archive.writestr("manifests/instana-agent-operator.clusterserviceversion.yaml", yaml.safe_dump(csv))
            env = dict(os.environ, PATH=os.environ["RELEASE_TEST_TOOL_DIR"] + ":" + os.environ["PATH"],
                       WORKSPACE=tmp, SELECTED_BOM_FILE=str(work / "bom.json"))
            subprocess.run(["helm", "package", str(chart), "--destination", tmp], env=env, check=True,
                           capture_output=True)
            staged_tgz = work / f"instana-agent-{VERSION}.tgz"
            staged = {"helm-chart": staged_tgz, "olm-bundle": staged_zip, "operator-yaml": staged_yaml}
            value["checksums"] = {k: hashlib.sha256(p.read_bytes()).hexdigest() for k, p in staged.items()}
            (work / "bom.json").write_text(json.dumps(value))
            originals = {p: p.read_bytes() for p in staged.values()}
            shell = f'''
set -euo pipefail
get_env() {{
  case "$1" in
    cos-state-path) echo {STATE} ;;
    repackaged-*) cat "$WORKSPACE/$1" ;;
    *) echo "${{2:-fixture}}" ;;
  esac
}}
set_env() {{ printf '%s\n' "$2" > "$WORKSPACE/$1"; }}
ibmcloud() {{
  local key='' out="${{@: -1}}"
  while [[ $# -gt 0 ]]; do
    if [[ $1 == --key ]]; then key=$2; shift; fi
    shift
  done
  case "$key" in
    */helm/*) cp "$WORKSPACE/instana-agent-{VERSION}.tgz" "$out" ;;
    */olm/olm-*) cp "$WORKSPACE/staged.zip" "$out" ;;
    */olm/instana-agent-operator.yaml) cp "$WORKSPACE/staged.yaml" "$out" ;;
    *) exit 99 ;;
  esac
}}
source "{SCRIPTS}/repackage-helm.sh"
source "{SCRIPTS}/smoke-public-artifacts.sh"
'''
            outputs = [work / f"helm-public/instana-agent-{VERSION}.tgz", work / f"operator-public/olm-{VERSION}.zip"]
            subprocess.run(["bash", "-c", shell], env=env, check=True, capture_output=True, text=True)
            first = [p.read_bytes() for p in outputs]
            subprocess.run(["bash", "-c", shell], env=env, check=True, capture_output=True, text=True)
            self.assertEqual(first, [p.read_bytes() for p in outputs])
            for path, original in originals.items():
                self.assertEqual(path.read_bytes(), original)
            # A public artifact accidentally left on dev ICR must fail the smoke gate.
            public_yaml = work / "operator-public/instana-agent-operator.yaml"
            public_yaml.write_text(public_yaml.read_text().replace(PUBLIC, DEV))
            harness = shell.split(f'source "{SCRIPTS}/repackage-helm.sh"', 1)[0]
            failed = subprocess.run(["bash", "-c", harness + f'\nsource "{SCRIPTS}/smoke-public-artifacts.sh"'],
                                    env=env, capture_output=True, text=True)
            self.assertNotEqual(failed.returncode, 0)
            self.assertIn("development reference", failed.stderr)


if __name__ == "__main__":
    unittest.main()
