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
import urllib.error
import zipfile

from release_api import GitHub, discover, semver

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


class ShellTest(unittest.TestCase):
    def test_chart_candidate_opts_into_fixed_etcd_ca_precedence(self):
        script = (SCRIPTS / "e2e-async.sh").read_text()
        self.assertIn("export EXPECT_CUSTOM_ETCD_CA=true", script)

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="release-test-")
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


    def test_cos_failure_names_object_and_preserves_exit_code(self):
        result = self.bash(f'''
source "{SCRIPTS}/release-common.sh"
COS_BUCKET=fixture-bucket
COS_REGION=us-south
ibmcloud() {{ echo 'The specified key does not exist.' >&2; return 23; }}
cos_get candidate/e2e/success/ocp-fyre-latest.json "$WORKSPACE/result.json"
''', expected=23)
        self.assertIn("cos://fixture-bucket/candidate/e2e/success/ocp-fyre-latest.json", result.stderr)
        self.assertIn("exit code 23", result.stderr)


    def test_cos_upload_failure_is_not_hidden_in_a_conditional(self):
        result = self.bash(f'''
source "{SCRIPTS}/release-common.sh"
COS_BUCKET=fixture COS_REGION=fixture
ibmcloud() {{ return 23; }}
if cos_put pipeline-validation/fixture/report.json fixture.json; then
  exit 99
else
  [[ $? == 23 ]]
fi
''')
        self.assertNotIn("COS upload:", result.stdout)


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


    def test_frozen_checkout_fetches_sha_before_checkout(self):
        for script in ("build-and-stage.sh", "e2e-async.sh"):
            for folder, variable in (("operator-src", "OPERATOR_SHA"), ("charts-src", "CHARTS_SHA")):
                lines = [line.strip() for line in (SCRIPTS / script).read_text().splitlines()
                         if line.strip().startswith(f'git -C "${{WORKSPACE}}/{folder}"')]
                self.assertEqual(len(lines), 2)
                self.assertIn(f'fetch --depth 1 origin "${{{variable}}}"', lines[0])
                self.assertIn(f'checkout --detach "${{{variable}}}"', lines[1])


    def test_go_setup_uses_frozen_source_without_pr_pipeline_variables(self):
        source = self.workspace / "operator-src"
        source.mkdir()
        installer = source / "installGolang.sh"
        for script in ("build-and-stage.sh", "e2e-async.sh"):
            text = (SCRIPTS / script).read_text()
            install = 'bash "${WORKSPACE}/operator-src/installGolang.sh" amd64'
            self.assertGreater(text.index(install), text.index('checkout --detach "${OPERATOR_SHA}"'))
            self.assertNotIn("./installGolang.sh", text)
            lines = text.splitlines()
            index = lines.index(install)
            setup = "\n".join(lines[index:index + 2])
            for rc in (0, 23):
                with self.subTest(script=script, installer_exit=rc):
                    installer.write_text('printf "%s" "$1" > "$WORKSPACE/installer-arch"\n'
                                         'echo "installer diagnostic" >&2\n'
                                         f'exit {rc}\n')
                    result = self.bash('unset APP_REPO_FOLDER PIPELINE_CONFIG_REPO_PATH\n' + setup
                                       + '\n[[ $PATH == /usr/local/go/bin:* ]]\necho setup-complete\n', expected=rc)
                    self.assertEqual((self.workspace / "installer-arch").read_text(), "amd64")
                    self.assertIn("installer diagnostic", result.stderr)
                    self.assertEqual("setup-complete" in result.stdout, rc == 0)


    def test_build_dispatch_attempts_all_then_fails(self):
        text = (SCRIPTS / "build-and-stage.sh").read_text().split("DISPATCH_FAILED=0", 1)[1]
        result = self.bash('''
E2E_STAGES=(one two three four)
trigger-task() { echo "DISPATCH $1"; [[ $1 != two ]]; }
DISPATCH_FAILED=0
''' + text, expected=1)
        self.assertEqual(result.stdout.count("DISPATCH "), 4)


@unittest.skipUnless(os.environ.get("RELEASE_TEST_TOOL_DIR"), "Set RELEASE_TEST_TOOL_DIR for real Helm/yq smoke")
class RealRepackageTest(unittest.TestCase):
    def test_chart_build_preserves_legacy_version_and_candidate_image_tag(self):
        import yaml
        root = SCRIPTS.parents[1]
        charts = root.parent / "instana-agent-charts"
        with tempfile.TemporaryDirectory() as tmp:
            work = Path(tmp)
            shutil.copytree(charts / "canonical", work / "canonical")
            (work / "versions").mkdir()
            (work / "versions/INSTANA_AGENT_CHART_VERSION").write_text("2.0.56\n")
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
            for chart_version in ("2.0.56", VERSION):
                if chart_version == VERSION:
                    env["CHART_VERSION"] = VERSION
                result = subprocess.run(["bash", str(charts / "ci/build.sh")], cwd=work, env=env,
                                        capture_output=True, text=True)
                self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
                with tarfile.open(work / f"artefacts/instana-agent-{chart_version}.tgz") as archive:
                    values = yaml.safe_load(archive.extractfile("instana-agent/values.yaml"))
                    metadata = yaml.safe_load(archive.extractfile("instana-agent/Chart.yaml"))
                    template = archive.extractfile(
                        "instana-agent/templates/operator_deployment_instana-agent-controller-manager.yml").read().decode()
                self.assertEqual(values["controllerManager"]["image"]["tag"], dev_ref.rsplit(":", 1)[1])
                self.assertEqual(metadata["appVersion"], VERSION)
                self.assertEqual(metadata["version"], chart_version)
                self.assertIn(dev_ref, template)



if __name__ == "__main__":
    unittest.main()
