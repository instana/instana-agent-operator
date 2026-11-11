"""Offline release regression tests. No cloud credentials, network or clusters.

Run: PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s ci/sps-scripts -v
"""

import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile

from release_api import ensure_release, ensure_tag
from release_test_support import FakeAPI, ReleaseShellTest, SCRIPTS, VERSION, bom
from publish_olm import bundle_files, publish, verify_catalog_files


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
        with tempfile.TemporaryDirectory() as tmp:
            file = Path(tmp) / "operator.yaml"
            file.write_bytes(b"approved bytes")
            api = FakeAPI()
            calls = []
            def request(path, data=None, **kwargs):
                self.assertNotEqual(kwargs.get("method"), "DELETE")
                calls.append((path, data))
                if kwargs.get("raw"):
                    return b"approved bytes"
                return {"id": 1, "draft": True, "html_url": "https://fixture.invalid/release"}
            api.request = request
            api.pages = lambda path: [{"name": file.name, "url": "https://fixture.invalid/asset",
                                       "state": "uploaded"}]
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


    def test_missing_draft_asset_is_uploaded_before_publication(self):
        api = FakeAPI()
        draft = {
            "id": 1,
            "draft": True,
            "html_url": "https://fixture.invalid/release",
            "upload_url": "https://fixture.invalid/assets{?name,label}",
        }
        writes = []

        def request(path, data=None, **kwargs):
            if data is not None:
                writes.append((path, data, kwargs))
            return draft

        api.request = request
        with tempfile.TemporaryDirectory() as tmp:
            asset = Path(tmp) / "operator manifest.yaml"
            asset.write_bytes(b"approved bytes")
            ensure_release(api, bom(), [asset])

        self.assertEqual(writes, [
            ("https://fixture.invalid/assets?name=operator%20manifest.yaml", b"approved bytes",
             {"content_type": "application/octet-stream"}),
            ("/repos/instana/instana-agent-operator/releases/1", {"draft": False}, {"method": "PATCH"}),
        ])


    def test_published_release_with_missing_asset_requires_manual_recovery(self):
        api = FakeAPI()
        api.request = lambda *args, **kwargs: {"id": 1, "draft": False}
        with self.assertRaisesRegex(ValueError, "manual recovery required"):
            ensure_release(api, bom(), ["operator.yaml"])


    def test_incomplete_draft_asset_is_replaced_and_rerun_is_read_only(self):
        release = {
            "id": 1,
            "draft": True,
            "html_url": "https://fixture.invalid/release",
            "upload_url": "https://fixture.invalid/uploads{?name,label}",
        }
        asset_url = "https://fixture.invalid/asset"
        assets = [{"name": "operator.yaml", "url": asset_url, "state": "starter"}]
        writes = []
        api = FakeAPI()
        api.pages = lambda path: assets

        def request(path, data=None, **kwargs):
            method = kwargs.get("method")
            if method == "DELETE":
                self.assertEqual(path, asset_url)
                writes.append("delete")
                assets.clear()
            elif isinstance(data, bytes):
                self.assertEqual(path, "https://fixture.invalid/uploads?name=operator.yaml")
                self.assertEqual(data, b"approved bytes")
                self.assertFalse(assets)
                writes.append("upload")
                assets.append({"name": "operator.yaml", "url": asset_url, "state": "uploaded"})
            elif method == "PATCH":
                self.assertEqual(assets[0]["state"], "uploaded")
                writes.append("publish")
                release.update(data)
            elif kwargs.get("raw"):
                return b"approved bytes"
            return release

        api.request = request
        with tempfile.TemporaryDirectory() as tmp:
            file = Path(tmp) / "operator.yaml"
            file.write_bytes(b"approved bytes")
            ensure_release(api, bom(), [file])
            self.assertEqual(writes, ["delete", "upload", "publish"])
            writes.clear()
            ensure_release(api, bom(), [file])
            self.assertEqual(writes, [])


    def test_incomplete_asset_recovery_failure_never_publishes_release(self):
        for failure in ("delete", "upload"):
            with self.subTest(failure=failure), tempfile.TemporaryDirectory() as tmp:
                file = Path(tmp) / "operator.yaml"
                file.write_bytes(b"approved bytes")
                release = {"id": 1, "draft": True, "upload_url": "https://fixture.invalid/uploads{?name}"}
                asset = {"name": file.name, "url": "https://fixture.invalid/asset", "state": "starter"}
                api = FakeAPI()
                api.pages = lambda path: [asset]
                writes = []

                def request(path, data=None, **kwargs):
                    if kwargs.get("method") == "DELETE":
                        writes.append("delete")
                        if failure == "delete":
                            raise RuntimeError("delete failed")
                    elif isinstance(data, bytes):
                        writes.append("upload")
                        raise RuntimeError("upload failed")
                    elif kwargs.get("method") == "PATCH":
                        self.fail("Failed asset recovery must not publish the release")
                    return release

                api.request = request
                with self.assertRaisesRegex(RuntimeError, f"{failure} failed"):
                    ensure_release(api, bom(), [file])
                self.assertEqual(writes, ["delete"] if failure == "delete" else ["delete", "upload"])


    def test_incomplete_published_asset_is_never_deleted(self):
        api = FakeAPI()
        api.pages = lambda path: [{"name": "operator.yaml", "state": "starter"}]

        def request(path, data=None, **kwargs):
            self.assertIsNone(data)
            self.assertNotIn("method", kwargs)
            self.assertFalse(kwargs.get("raw"))
            return {"id": 1, "draft": False}

        api.request = request
        with self.assertRaisesRegex(ValueError, "manual recovery required"):
            ensure_release(api, bom(), ["operator.yaml"])


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


class ShellTest(ReleaseShellTest):
    def test_artifact_checks_stop_before_any_publication(self):
        (self.workspace / "selected-bom.json").write_text(json.dumps(bom()))
        for failed_script in ("validate-release.sh", "repackage-helm.sh", "smoke-public-artifacts.sh"):
            with self.subTest(failed_script=failed_script):
                result = self.bash(f'''
dnf() {{ :; }}
yq() {{ :; }}
preflight() {{ :; }}
get_secret() {{ echo fixture; }}
python3() {{ :; }}
bash() {{
  echo "Script: $1"
  [[ $1 != */{failed_script} ]] || return 23
}}
source "{SCRIPTS}/run-promotion.sh"
''', expected=23)
                for publisher in ("promote-images.sh", "preflight-release.sh", "publish-helm.sh", "publish-olm.sh"):
                    self.assertNotIn(publisher, result.stdout)
                self.assertNotIn("promotion-receipt", result.stdout)


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
        self.assertLess(result.stdout.index("PASSED [public-artifact-smoke]"),
                        result.stdout.index("CHECK [redhat-images]"))


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
        with tempfile.TemporaryDirectory() as tmp:
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
