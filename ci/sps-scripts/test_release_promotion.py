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

from release_api import GitHub, check_not_older_release, discover, ensure_release, ensure_tag, semver
from release_state import validate_bom, validate_success
from check_artifactory import check as check_artifactory, latest_chart_version, NoRedirect
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


    def test_promotion_names_failed_stage_and_stops(self):
        (self.workspace / "selected-bom.json").write_text(json.dumps(bom()))
        result = self.bash(f'''
dnf() {{ :; }}
yq() {{ :; }}
preflight() {{ :; }}
get_secret() {{ echo secret-must-not-appear; }}
python3() {{ :; }}
bash() {{
  echo "Script: $1"
  [[ $1 != */publish-helm.sh ]] || return 23
}}
source "{SCRIPTS}/run-promotion.sh"
''', expected=23)
        self.assertIn("PASSED [candidate-and-e2e]", result.stdout)
        self.assertIn("PASSED [versioned-images]", result.stdout)
        self.assertIn("FAILED [publish-helm]", result.stderr)
        self.assertIn("exit code 23", result.stderr)
        self.assertNotIn("publish-olm.sh", result.stdout)
        self.assertNotIn("secret-must-not-appear", result.stdout + result.stderr)


    def test_publishers_use_plain_github_token_and_nonsecret_commit_identity(self):
        for name in ("run-promotion.sh", "publish-helm.sh", "publish-olm.sh"):
            script = (SCRIPTS / name).read_text()
            with self.subTest(script=name):
                self.assertIn("GH_TOKEN=$(get_secret github-token)", script)
                self.assertNotIn("github-instana-agent-build", script)
                self.assertNotIn(".token'", script)
        helm = (SCRIPTS / "publish-helm.sh").read_text()
        self.assertIn('config user.email "instanacd@instana.com"', helm)
        self.assertIn('config user.name "Instana CD"', helm)


    def test_redhat_registry_secret_used_for_copy_and_preflight_login(self):
        for name in ("promote-images.sh", "preflight-release.sh"):
            script = (SCRIPTS / name).read_text()
            with self.subTest(script=name):
                self.assertIn("get_secret redhat-registry", script)
                self.assertNotIn("quay-redhat-isv-push-credentials", script)
                self.assertIn(".username", script)
                self.assertIn(".password", script)
        preflight = (SCRIPTS / "preflight-release.sh").read_text()
        self.assertIn(".token | strings | select(length > 0)", preflight)
        self.assertNotIn("get_secret redhat-pyxis-api-token", preflight)


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


    def test_release_icr_uses_json_username_and_password(self):
        script = (SCRIPTS / "promote-images.sh").read_text()
        phase = script.split("  versioned)\n", 1)[1].split("    ;;", 1)[0]
        for creds, rc in (({"username": "release-user", "password": "release-password"}, 0),
                          ({"username": "release-user"}, 4),
                          ({"username": "", "password": "release-password"}, 4)):
            with self.subTest(creds=creds):
                self.bash(f'''
VERSION=2.2.24 ICR_KEY=fixture
get_secret() {{ [[ $1 == icr-rel-api-key ]]; echo '{json.dumps(creds)}'; }}
copy_version() {{
  if [[ $1 == icr.io/instana-rel/* ]]; then
    [[ $2 == release-user:release-password ]] || exit 99
  fi
}}
{phase}
''', expected=rc)


class ArchiveTest(unittest.TestCase):
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



if __name__ == "__main__":
    unittest.main()
