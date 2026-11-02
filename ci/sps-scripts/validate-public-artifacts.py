"""Validate real rendered YAML, version labels, CRDs and OLM upgrade metadata."""

from pathlib import Path
import sys
import yaml


def documents(path):
    return [d for d in yaml.safe_load_all(Path(path).read_text()) if isinstance(d, dict)]


def deployment_image(docs, expected, version):
    deployments = [d for d in docs if d.get("kind") == "Deployment"
                   and d["metadata"]["name"] == "instana-agent-controller-manager"]
    if len(deployments) != 1:
        raise ValueError("Expected exactly one operator deployment")
    deployment = deployments[0]
    images = [c["image"] for c in deployment["spec"]["template"]["spec"]["containers"]]
    if expected not in images:
        raise ValueError(f"Wrong production image: {images}")
    if deployment["metadata"].get("labels", {}).get("app.kubernetes.io/operator-version") != version:
        raise ValueError("Operator version label mismatch")


def validate(version, manifest, output, previous):
    output = Path(output)
    expected = f"icr.io/instana/instana-agent-operator:{version}"
    for path in (Path(manifest), output / "rendered.yaml", output / "fallback.yaml"):
        deployment_image(documents(path), expected, version)
    chart = yaml.safe_load((output / "instana-agent/Chart.yaml").read_text())
    if str(chart["version"]) != version or str(chart["appVersion"]) != version:
        raise ValueError("Chart/appVersion mismatch")
    csvs = list((output / "olm/manifests").glob("*.clusterserviceversion.yaml"))
    if len(csvs) != 1:
        raise ValueError("Expected one CSV")
    csv = documents(csvs[0])[0]
    if str(csv["spec"]["version"]) != version or not csv["spec"].get("replaces"):
        raise ValueError("Invalid CSV version or missing replaces")
    if csv["spec"]["replaces"] != f"instana-agent-operator.v{previous}":
        raise ValueError("CSV replaces does not match the frozen predecessor")
    deployments = csv["spec"]["install"]["spec"]["deployments"]
    images = [c["image"] for d in deployments for c in d["spec"]["template"]["spec"]["containers"]]
    if expected not in images or csv["metadata"]["annotations"]["containerImage"] != expected:
        raise ValueError("Wrong public CSV image")
    crds = [d for p in (output / "instana-agent/crds").glob("*") for d in documents(p)]
    if not {"agents.instana.io", "agentsremote.instana.io"} <= {d["metadata"]["name"] for d in crds}:
        raise ValueError("Missing chart CRDs")


if __name__ == "__main__":
    validate(*sys.argv[1:])
