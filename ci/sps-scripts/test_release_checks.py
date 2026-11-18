"""Offline tests for production candidate gates and public artifact checks.

Run: PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s ci/sps-scripts -v
"""

import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch
import zipfile

from release_api import GitHub, check_not_older_release
from release_state import validate_bom, validate_success
from release_test_support import DEV, PUBLIC, STATE, SCRIPTS, VERSION, FakeAPI, ReleaseShellTest, bom


class VersionsTest(unittest.TestCase):
    def test_read_only_api_rejects_mutations_before_network_access(self):
        api = GitHub("https://fixture.invalid", "fixture", read_only=True)
        with patch("release_api.urllib.request.build_opener", side_effect=AssertionError("Unexpected network")):
            for kwargs in ({"data": {}}, {"data": b"upload"}, {"method": "POST"},
                           {"method": "PATCH"}, {"method": "DELETE"}, {"method": "PUT"}):
                with self.subTest(kwargs=kwargs), self.assertRaises(ValueError):
                    api.request("/fixture", **kwargs)

    def test_read_only_api_allows_get(self):
        api = GitHub("https://fixture.invalid", "fixture", read_only=True)
        with patch("release_api.urllib.request.build_opener") as opener:
            opener.return_value.open.return_value = io.BytesIO(b'{"read": true}')
            self.assertEqual(api.request("/fixture"), {"read": True})
            request = opener.return_value.open.call_args.args[0]
            self.assertEqual(request.get_method(), "GET")

    def test_older_candidate_cannot_downgrade_public_release(self):
        def release(version):
            return {"tag_name": f"v{version}", "draft": False, "prerelease": False}

        check_not_older_release(FakeAPI(releases=[release(VERSION)]), VERSION)
        with self.assertRaises(ValueError):
            check_not_older_release(FakeAPI(releases=[release("2.3.0")]), VERSION)


class StateTest(unittest.TestCase):
    def test_exact_candidate(self):
        validate_bom(bom(), STATE + "/bom.json")

    def test_cli_rejects_non_main_candidate(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bom.json"
            path.write_text(json.dumps(bom() | {"source-branch": "test-release"}))
            result = subprocess.run(
                ["python3", str(SCRIPTS / "release_state.py"), "bom", str(path), STATE + "/bom.json"],
                capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("branches must both be main", result.stderr)

    def test_rejects_mutable_pointer_wrong_branch_and_incomplete_build(self):
        for change in ({"source-branch": "dev"}, {"charts-branch": "dev"}, {"build-complete": False},
                       {"release-tag": "v2.2.23"}, {"candidate-id": "minute-only"}):
            with self.subTest(change=change), self.assertRaises(ValueError):
                validate_bom(bom() | change, STATE + "/bom.json")
        with self.assertRaises(ValueError):
            validate_bom(bom(), "pipeline-state/latest-release-candidate/bom.json")

    def test_rejects_artifact_outside_candidate(self):
        value = bom()
        value["artifacts"]["helm-chart"] = "another-candidate/chart.tgz"
        with self.assertRaises(ValueError):
            validate_bom(value, STATE + "/bom.json")

    def test_success_must_match_inputs(self):
        value = bom()
        result = {key: value[key] for key in ("candidate-id", "sources", "images", "checksums")}
        result |= {"status": "success", "cluster": "ocp-fyre-latest", "mode": "operator",
                   "log": f"{STATE}/e2e/ocp-fyre-latest/test-attempt-success.log", "log-sha256": "e" * 64}
        validate_success(value, result, "ocp-fyre-latest")
        for field, replacement in (("images", {}), ("sources", {}), ("status", "failure"), ("mode", "helm")):
            with self.subTest(field=field), self.assertRaises(ValueError):
                validate_success(value, result | {field: replacement}, "ocp-fyre-latest")


class ShellTest(ReleaseShellTest):
    def test_promotion_requires_all_pass_pointers_unless_explicitly_forced(self):
        value = bom()
        store = self.workspace / "store"
        for key, path in value["artifacts"].items():
            file = store / path
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_bytes(b"fixture artifact")
            value["checksums"][key] = hashlib.sha256(file.read_bytes()).hexdigest()
        (store / STATE / "bom.json").write_text(json.dumps(value))
        script = f'''
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
        missing = self.bash('FORCE=false\n' + script, expected=1)
        self.assertIn("PASSED [candidate-bom]", missing.stdout)
        for artifact in value["artifacts"]:
            self.assertIn(f"PASSED [artifact-{artifact}]", missing.stdout)
        self.assertIn("FAILED [e2e-ocp-fyre-latest-evidence]", missing.stderr)
        self.assertIn(f"{STATE}/e2e/success/ocp-fyre-latest.json", missing.stderr)
        self.assertNotIn("CHECK [e2e-gke-operator-lowest", missing.stdout)
        forced = self.bash('FORCE=true\n' + script)
        self.assertIn("FORCED PROMOTION", forced.stdout)
        for cluster, mode in (("ocp-fyre-latest", "operator"), ("gke-operator-lowest", "operator"),
                              ("ocp-fyre-lowest", "helm"), ("gke-operator-latest", "helm")):
            log = f"{STATE}/e2e/{cluster}/test-attempt-success.log"
            log_file = store / log
            log_file.parent.mkdir(parents=True)
            log_file.write_bytes(b"passed")
            result = {key: value[key] for key in ("candidate-id", "sources", "images", "checksums")}
            result |= {"status": "success", "cluster": cluster, "mode": mode, "log": log,
                       "log-sha256": hashlib.sha256(b"passed").hexdigest()}
            pointer = store / STATE / "e2e/success" / f"{cluster}.json"
            pointer.parent.mkdir(parents=True, exist_ok=True)
            pointer.write_text(json.dumps(result))
        passed = self.bash('FORCE=false\n' + script)
        self.assertIn("PASSED [e2e-gke-operator-latest-log]", passed.stdout)

        (store / value["artifacts"]["helm-chart"]).write_bytes(b"corrupt chart")
        corrupt = self.bash('FORCE=true\n' + script, expected=1)
        self.assertIn("FAILED [artifact-helm-chart]", corrupt.stderr)
        self.assertNotIn("FORCED PROMOTION", corrupt.stdout)


class ArchiveTest(unittest.TestCase):
    def test_normalized_archives_are_repeatable(self):
        spec = importlib.util.spec_from_file_location("normalize", SCRIPTS / "normalize-release-archives.py")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        with tempfile.TemporaryDirectory() as tmp:
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


@unittest.skipUnless(os.environ.get("RELEASE_TEST_TOOL_DIR"), "Set RELEASE_TEST_TOOL_DIR for real Helm/yq smoke")
class RealRepackageTest(unittest.TestCase):
    def test_real_chart_repackaging_render_and_deterministic_retry(self):
        import yaml

        root = SCRIPTS.parents[1]
        charts = root.parent / "instana-agent-charts"
        with tempfile.TemporaryDirectory() as tmp:
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
            value["checksums"] = {key: hashlib.sha256(path.read_bytes()).hexdigest() for key, path in staged.items()}
            (work / "bom.json").write_text(json.dumps(value))
            originals = {path: path.read_bytes() for path in staged.values()}
            script = f'''
set -euo pipefail
get_env() {{
  case "$1" in
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
            subprocess.run(["bash", "-c", script], env=env, check=True, capture_output=True, text=True)
            first = [path.read_bytes() for path in outputs]
            subprocess.run(["bash", "-c", script], env=env, check=True, capture_output=True, text=True)
            self.assertEqual(first, [path.read_bytes() for path in outputs])
            for path, original in originals.items():
                self.assertEqual(path.read_bytes(), original)

            public_yaml = work / "operator-public/instana-agent-operator.yaml"
            public_yaml.write_text(public_yaml.read_text().replace(PUBLIC, DEV))
            setup = script.split(f'source "{SCRIPTS}/repackage-helm.sh"', 1)[0]
            failed = subprocess.run(["bash", "-c", setup + f'\nsource "{SCRIPTS}/smoke-public-artifacts.sh"'],
                                    env=env, capture_output=True, text=True)
            self.assertNotEqual(failed.returncode, 0)
            self.assertIn("development reference", failed.stderr)


if __name__ == "__main__":
    unittest.main()
