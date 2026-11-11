"""Offline validation of candidate identity and successful E2E evidence."""

import json
from pathlib import Path
import re
import sys

from release_api import semver


def validate_bom(bom, path):
    version = bom["version"]
    if not semver(version) or version.startswith("v") or bom["release-tag"] != f"v{version}":
        raise ValueError("Invalid release version/tag")
    if bom["cos-state-path"] + "/bom.json" != path or "latest" in path or ".." in path.split("/"):
        raise ValueError("Select an explicit candidate BOM path, not a latest pointer")
    if not re.fullmatch(r"[0-9]{14}-[a-f0-9-]{36}", bom["candidate-id"]):
        raise ValueError("Invalid candidate ID")
    if bom["cos-state-path"] != f"pipeline-state/release-{version}/{bom['candidate-id']}":
        raise ValueError("Candidate path does not match identity")
    if bom.get("build-complete") is not True:
        raise ValueError("Build not complete")
    if bom["source-branch"] != "main" or bom["charts-branch"] != "main":
        raise ValueError("Candidate operator and charts branches must both be main")
    for key in ("operator-sha", "charts-sha"):
        sha = bom["sources"][key]
        if not re.fullmatch(r"[a-f0-9]{40}", sha):
            raise ValueError("Invalid source SHA")
    if not re.fullmatch(r"sha256:[a-f0-9]{64}", bom["images"]["operator-manifest-digest"]):
        raise ValueError("Invalid image digest")
    expected_image = f"icr.io/instana-agent-dev/instana-agent-operator:{version}-{bom['candidate-id']}"
    if bom["images"]["operator-manifest"] != expected_image:
        raise ValueError("Candidate image tag does not match candidate ID")
    if not semver(bom["previous-version"]) or semver(bom["previous-version"]) >= semver(version):
        raise ValueError("Invalid OLM predecessor")
    expected = {"helm-chart": f"helm/instana-agent-{version}.tgz", "olm-bundle": f"olm/olm-{version}.zip",
                "operator-yaml": "olm/instana-agent-operator.yaml"}
    for key, suffix in expected.items():
        if bom["artifacts"][key] != f"{bom['cos-state-path']}/{suffix}":
            raise ValueError("Artifact outside selected candidate")
        if not re.fullmatch(r"[a-f0-9]{64}", bom["checksums"][key]):
            raise ValueError("Missing artifact checksum")


def validate_success(bom, result, cluster):
    if result["status"] != "success" or result["cluster"] != cluster:
        raise ValueError(f"No successful E2E for {cluster}")
    for field in ("candidate-id", "sources", "images", "checksums"):
        if result[field] != bom[field]:
            raise ValueError(f"E2E {field} mismatch for {cluster}")
    mode = "helm" if cluster in ("ocp-fyre-lowest", "gke-operator-latest") else "operator"
    if result["mode"] != mode:
        raise ValueError("Wrong E2E installer mode")
    if not result["log"].startswith(f"{bom['cos-state-path']}/e2e/{cluster}/test-"):
        raise ValueError("E2E log outside candidate")
    if not re.fullmatch(r"[a-f0-9]{64}", result["log-sha256"]):
        raise ValueError("Missing E2E log checksum")


if __name__ == "__main__":
    bom = json.loads(Path(sys.argv[2]).read_text())
    if sys.argv[1] == "bom":
        validate_bom(bom, sys.argv[3])
    elif sys.argv[1] == "success":
        validate_success(bom, json.loads(Path(sys.argv[3]).read_text()), sys.argv[4])
    else:
        raise ValueError("Unknown validation command")
