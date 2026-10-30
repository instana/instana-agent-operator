"""Offline regression tests: python3 -m unittest discover -s ci/sps-scripts -v.

No production credentials or services are used. External service commands are
blocked by default, and individual tests replace them with local fixture mocks.
"""

import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest
import zipfile


SCRIPTS = Path(__file__).resolve().parent
ROOT = SCRIPTS.parents[1]
DEV_REPO = "icr.io/instana-agent-dev/instana-agent-operator"
PUBLIC_REPO = "icr.io/instana/instana-agent-operator"
VERSION = "2.2.22"
STATE = f"pipeline-state/release-{VERSION}/candidate-a"


class ReleaseScriptsTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="release-test-", dir="/tmp/opencode")
        self.addCleanup(self.temp.cleanup)
        self.workspace = Path(self.temp.name)

    def run_bash(self, code):
        # Block service commands even if a test unexpectedly reaches new code.
        guards = "\n".join(
            f'{cmd}() {{ echo "Unexpected service command: {cmd}" >&2; exit 99; }}'
            for cmd in (
                "ibmcloud", "curl", "git", "gsutil", "gcloud", "skopeo",
                "helm", "dnf", "docker", "trigger-task", "get_secret",
            )
        )
        result = subprocess.run(
            ["bash", "--noprofile", "--norc", "-c", "set -euo pipefail\n" + guards + "\n" + code],
            cwd=self.workspace,
            env={"PATH": os.defpath, "WORKSPACE": str(self.workspace), "HOME": str(self.workspace)},
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def text(self, name):
        return (SCRIPTS / name).read_text()

    def bom(self):
        return {
            "version": VERSION,
            "release-tag": f"v{VERSION}",
            "cos-state-path": STATE,
            "source-branch": "main",
            "sources": {"operator-sha": "operator-a", "charts-sha": "charts-a"},
            "images": {
                "operator-manifest": f"{DEV_REPO}:operator-a",
                "operator-manifest-digest": "sha256:abc123",
            },
            "artifacts": {
                "helm-chart": "staged.tgz",
                "olm-bundle": "staged.zip",
                "operator-yaml": "staged.yaml",
            },
        }

    def test_all_image_copies_have_separate_credentials_source_and_all_platforms(self):
        code = self.text("promote-images.sh").split("skopeo_copy() {", 1)[1]
        code = "skopeo_copy() {" + code
        for branch, count in (("main", 4), ("release-2.2", 3)):
            with self.subTest(branch=branch):
                self.run_bash(f'''
VERSION={VERSION}
SOURCE_BRANCH={branch}
ICR_API_KEY='dev test key'
ICRREL_API_KEY='rel test key'
QUAY_USERNAME=quay-user
QUAY_PASSWORD='quay test key'
SRC_IMAGE={DEV_REPO}@sha256:abc123
count=0
skopeo() {{
  [[ $# == 8 && $1 == copy && $2 == --all && $3 == --src-creds ]]
  [[ $4 == 'iamapikey:dev test key' && $5 == --dest-creds ]]
  [[ $7 == "docker://${{SRC_IMAGE}}" ]]
  case "$8" in
    docker://icr.io/instana/*) [[ $6 == "iamapikey:${{ICR_API_KEY}}" ]] ;;
    docker://icr.io/instana-rel/*) [[ $6 == "iamapikey:${{ICRREL_API_KEY}}" ]] ;;
    docker://quay.io/*) [[ $6 == "${{QUAY_USERNAME}}:${{QUAY_PASSWORD}}" ]] ;;
    *) exit 1 ;;
  esac
  count=$((count + 1))
}}
{code}
[[ $count == {count} ]]
''')

    def test_promotion_keeps_the_candidate_selected_by_validation(self):
        (self.workspace / "candidate.json").write_text(json.dumps(self.bom()))
        (self.workspace / "latest.json").write_text(json.dumps(self.bom()))
        mock = '''
get_env() {
  case "$1" in
    cos-state-path) cat "$WORKSPACE/selected-state" ;;
    release-state-path) echo pipeline-state/latest-release-candidate/bom.json ;;
    skip-e2e-check) echo false ;;
    *) echo "${2:-fixture}" ;;
  esac
}
set_env() { [[ $1 == cos-state-path ]]; echo "$2" > "$WORKSPACE/selected-state"; }
export_env() { [[ $1 == cos-state-path ]]; }
ibmcloud() {
  local key='' output="${@: -1}"
  while [[ $# -gt 0 ]]; do
    if [[ $1 == --key ]]; then key=$2; shift; fi
    shift
  done
  case "$key" in
    pipeline-state/latest-release-candidate/bom.json) cp "$WORKSPACE/latest.json" "$output" ;;
    */e2e/*.json) echo '{"status":"success"}' > "$output" ;;
    "$(cat "$WORKSPACE/selected-state")/bom.json") cp "$WORKSPACE/candidate.json" "$output" ;;
    *) echo "Unexpected COS key: $key" >&2; exit 1 ;;
  esac
}
'''
        readers = []
        for name in ("promote-images.sh", "repackage-helm.sh", "publish-helm.sh"):
            text = self.text(name)
            begin = text.index("COS_BUCKET=$(get_env")
            end = text.index('rm -f "${BOM_FILE}"', begin) + len('rm -f "${BOM_FILE}"')
            readers.append(text[begin:end] + f'\n[[ $VERSION == {VERSION} ]]')
        pipeline = (ROOT / "ci/.pipeline-config-release-promotion.yaml").read_text()
        preflight = pipeline.split('          COS_BUCKET=$(get_env', 1)[1]
        preflight = 'COS_BUCKET=$(get_env' + preflight.split('          PREFLIGHT_TOKEN=', 1)[0]
        self.run_bash(
            mock + self.text("validate-release.sh")
            + '\necho \'{"version":"another-candidate"}\' > "$WORKSPACE/latest.json"\n'
            + "\n".join(readers) + "\n" + preflight + f'\n[[ $VERSION == {VERSION} ]]'
        )

    def test_build_and_e2e_read_scoped_bom(self):
        for name in ("build-and-stage.sh", "e2e-async.sh"):
            with self.subTest(script=name):
                text = self.text(name)
                read = text.split("ibmcloud cos get-object", 1)[1].split("VERSION=", 1)[0]
                self.assertIn('--key "${COS_STATE_PATH}/bom.json"', read)
                self.assertNotIn("latest-release-candidate", read)

    def test_public_artifacts_are_repackaged_without_changing_staged_copies(self):
        (self.workspace / "bom.json").write_text(json.dumps(self.bom()))
        yaml = f"image: {DEV_REPO}:{VERSION}\n"
        (self.workspace / "staged.yaml").write_text(yaml)
        with zipfile.ZipFile(self.workspace / "staged.zip", "w") as archive:
            archive.writestr("manifests/operator.yaml", yaml + f"containerImage: {DEV_REPO}@sha256:abc123\n")
            archive.writestr("metadata/annotations.yaml", "annotations: {}\n")
        chart = self.workspace / "fixture/instana-agent"
        (chart / "templates").mkdir(parents=True)
        (chart / "Chart.yaml").write_text(f"version: {VERSION}\nappVersion: {VERSION}\n")
        (chart / "values.yaml").write_text("controllerManager: {}\n")
        (chart / "templates/operator_deployment_instana-agent-controller-manager.yml").write_text(
            f'image: "{DEV_REPO}:{VERSION}"\n'
        )
        with tarfile.open(self.workspace / "staged.tgz", "w:gz") as archive:
            archive.add(chart, arcname="instana-agent")
        staged = {name: (self.workspace / name).read_bytes() for name in ("staged.yaml", "staged.zip", "staged.tgz")}
        self.run_bash(f'''
get_env() {{
  case "$1" in
    cos-state-path) echo {STATE} ;;
    *) echo "${{2:-fixture}}" ;;
  esac
}}
set_env() {{ printf '%s\n' "$2" > "$WORKSPACE/$1"; }}
ibmcloud() {{
  local key='' output="${{@: -1}}"
  while [[ $# -gt 0 ]]; do
    if [[ $1 == --key ]]; then key=$2; shift; fi
    shift
  done
  [[ $key == {STATE}/bom.json ]] && key=bom.json
  cp "$WORKSPACE/$key" "$output"
}}
# Helm/yq mocks keep the test offline; real sed/zip/unzip process the fixtures.
yq() {{
  if [[ $2 == -i ]]; then
    printf 'controllerManager:\n  image:\n    name: {PUBLIC_REPO}\n    tag: {VERSION}\n' > "$4"
  else
    echo {VERSION}
  fi
}}
helm() {{
  if [[ $1 == package ]]; then
    tar -czf "${{@: -1}}/instana-agent-{VERSION}.tgz" -C "$2" .
  else
    [[ $1 == lint ]]
  fi
}}
{self.text('repackage-helm.sh')}
''')
        public_yaml = Path((self.workspace / "repackaged-operator-yaml").read_text().strip())
        self.assertEqual(public_yaml.read_text(), yaml.replace(DEV_REPO, PUBLIC_REPO))
        public_zip = Path((self.workspace / "repackaged-olm-zip").read_text().strip())
        with zipfile.ZipFile(public_zip) as archive:
            manifest = archive.read("manifests/operator.yaml").decode()
            self.assertNotIn(DEV_REPO, manifest)
            self.assertIn(f"{PUBLIC_REPO}:{VERSION}", manifest)
            self.assertIn(f"{PUBLIC_REPO}@sha256:abc123", manifest)
            self.assertIn("metadata/annotations.yaml", archive.namelist())
        template = self.workspace / "helm-unpack/instana-agent/templates"
        self.assertEqual(
            (template / "operator_deployment_instana-agent-controller-manager.yml").read_text(),
            f'image: "{PUBLIC_REPO}:{VERSION}"\n',
        )
        for name, original in staged.items():
            self.assertEqual((self.workspace / name).read_bytes(), original)

    def test_tags_share_version_but_use_correct_hosts_and_credentials(self):
        text = self.text("publish-helm.sh")
        tags = text.split('echo "=== Pushing git tags ==="', 1)[1].split("# 2a. Publish", 1)[0]
        self.run_bash(f'''
VERSION={VERSION}
RELEASE_TAG=v{VERSION}
OPERATOR_SHA=operator-a
CHARTS_SHA=charts-a
PUBLIC_GITHUB_TOKEN=public-token
PUBLIC_GITHUB_EMAIL=fixture@example.invalid
PUBLIC_GITHUB_USERNAME=fixture
GHE_TOKEN=enterprise-token
git() {{ printf '%s\n' "$*" >> "$WORKSPACE/git-calls"; }}
{tags}
''')
        calls = (self.workspace / "git-calls").read_text()
        self.assertIn("public-token@github.com/instana/instana-agent-operator.git", calls)
        self.assertIn("enterprise-token@github.ibm.com/instana/instana-agent-charts.git", calls)
        self.assertIn(f"tag v{VERSION} operator-a", calls)
        self.assertIn(f"tag {VERSION} charts-a", calls)

    def test_frozen_checkout_fetches_sha_before_checkout(self):
        # Execute the actual changed checkout lines against a local-only repo.
        repo = self.workspace / "origin"
        def git(*args):
            return subprocess.check_output(
                ["git", *args], cwd=self.workspace, text=True,
                env={"PATH": os.defpath, "HOME": str(self.workspace), "GIT_CONFIG_NOSYSTEM": "1"},
                stderr=subprocess.DEVNULL,
            ).strip()
        git("init", str(repo))
        git("-C", str(repo), "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid",
            "commit", "-s", "--allow-empty", "-m", "frozen")
        frozen = git("-C", str(repo), "rev-parse", "HEAD")
        git("-C", str(repo), "-c", "user.name=fixture", "-c", "user.email=fixture@example.invalid",
            "commit", "-s", "--allow-empty", "-m", "new tip")
        for script in ("build-and-stage.sh", "e2e-async.sh"):
            for folder, variable in (("operator-src", "OPERATOR_SHA"), ("charts-src", "CHARTS_SHA")):
                with self.subTest(script=script, repo=folder):
                    dest = self.workspace / f"{script}-{folder}"
                    git("clone", "--depth", "1", repo.as_uri(), str(dest))
                    lines = [line.strip() for line in self.text(script).splitlines()
                             if line.strip().startswith(f'git -C "${{WORKSPACE}}/{folder}"')]
                    self.assertEqual(len(lines), 2)
                    code = "\n".join(lines).replace(f"${{WORKSPACE}}/{folder}", str(dest))
                    self.run_bash(f'unset -f git\n{variable}={frozen}\n{code}')
                    self.assertEqual(git("-C", str(dest), "rev-parse", "HEAD"), frozen)


if __name__ == "__main__":
    unittest.main()
