"""Shared candidate fixtures and isolated shell helpers for release tests."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest


SCRIPTS = Path(__file__).resolve().parent
VERSION = "2.2.24"
CANDIDATE = "20261102123045-12345678-1234-1234-1234-123456789abc"
STATE = f"pipeline-state/release-{VERSION}/{CANDIDATE}"
DEV = "icr.io/instana-agent-dev/instana-agent-operator"
PUBLIC = "icr.io/instana/instana-agent-operator"


def bom():
    """Return a fresh candidate so tests can safely mutate nested fields."""
    return {
        "version": VERSION,
        "release-tag": f"v{VERSION}",
        "previous-version": "2.2.23",
        "candidate-id": CANDIDATE,
        "cos-state-path": STATE,
        "build-complete": True,
        "source-branch": "main",
        "charts-branch": "main",
        "sources": {"operator-sha": "a" * 40, "charts-sha": "b" * 40},
        "images": {
            "operator-manifest": f"{DEV}:{VERSION}-{CANDIDATE}",
            "operator-manifest-digest": "sha256:" + "c" * 64,
        },
        "artifacts": {
            "helm-chart": f"{STATE}/helm/instana-agent-{VERSION}.tgz",
            "olm-bundle": f"{STATE}/olm/olm-{VERSION}.zip",
            "operator-yaml": f"{STATE}/olm/instana-agent-operator.yaml",
        },
        "checksums": {key: "d" * 64 for key in ("helm-chart", "olm-bundle", "operator-yaml")},
    }


class FakeAPI:
    def __init__(self, tags=(), releases=()):
        self.tags = tags
        self.releases = releases

    def pages(self, path):
        if path.endswith("/tags"):
            return [{"name": tag} for tag in self.tags]
        return self.releases


class ReleaseShellTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="release-test-")
        self.addCleanup(self.temp.cleanup)
        self.workspace = Path(self.temp.name)

    def bash(self, code, expected=0):
        commands = (
            "ibmcloud", "curl", "git", "gsutil", "gcloud", "skopeo", "helm", "dnf", "docker", "get_secret",
        )
        guards = "\n".join(
            f'{command}() {{ echo "Unexpected service command: {command}" >&2; exit 99; }}'
            for command in commands
        )
        result = subprocess.run(
            ["bash", "--noprofile", "--norc", "-c", "set -euo pipefail\n" + guards + "\n" + code],
            cwd=self.workspace,
            env={
                "PATH": os.defpath,
                "WORKSPACE": str(self.workspace),
                "HOME": str(self.workspace),
                "PYTHONDONTWRITEBYTECODE": "1",
            },
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, expected, result.stdout + result.stderr)
        return result
