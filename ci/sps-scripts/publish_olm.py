"""Create/resume the three OLM catalog PRs directly from the validated public zip.

No force pushes, fork-main resets, unrelated PR rebases, or release/latest lookup.
"""

import base64
import json
import os
from pathlib import Path
import sys
import zipfile

import yaml

from release_api import GitHub


CATALOGS = (
    ("k8s-operatorhub", "community-operators", "instana-agent-operator"),
    ("redhat-openshift-ecosystem", "certified-operators", "instana-agent-operator"),
    ("redhat-openshift-ecosystem", "redhat-marketplace-operators", "instana-agent-operator-rhmp"),
)


def bundle_files(archive, version, operator):
    files = {}
    with zipfile.ZipFile(archive) as source:
        for name in source.namelist():
            if name.endswith("/"):
                continue
            if name.startswith("/") or ".." in Path(name).parts:
                raise ValueError("Unsafe bundle path")
            content = source.read(name).decode()
            if operator.endswith("-rhmp"):
                if name == "manifests/instana-agent-operator.clusterserviceversion.yaml":
                    csv = yaml.safe_load(content)
                    annotations = csv["metadata"].setdefault("annotations", {})
                    annotations["marketplace.openshift.io/remote-workflow"] = (
                        "https://marketplace.redhat.com/en-us/operators/instana-agent-operator-rhmp/"
                        "pricing?utm_source=openshift_console")
                    annotations["marketplace.openshift.io/support-workflow"] = (
                        "https://marketplace.redhat.com/en-us/operators/instana-agent-operator-rhmp/"
                        "support?utm_source=openshift_console")
                    content = yaml.safe_dump(csv, sort_keys=False)
                    name = name.replace("instana-agent-operator.", "instana-agent-operator-rhmp.")
                elif name == "metadata/annotations.yaml":
                    data = yaml.safe_load(content)
                    data["annotations"]["operators.operatorframework.io.bundle.package.v1"] = operator
                    content = yaml.safe_dump(data, sort_keys=False)
            files[f"operators/{operator}/{version}/{name}"] = content
    return files


def catalog_files(api, repo, commit_sha, prefix):
    """Walk to the version subtree before recursion (catalog roots can truncate)."""
    tree_sha = api.request(f"{repo}/git/commits/{commit_sha}")["tree"]["sha"]
    for component in prefix.rstrip("/").split("/"):
        entries = api.request(f"{repo}/git/trees/{tree_sha}")["tree"]
        found = next((entry for entry in entries if entry["path"] == component), None)
        if found is None:
            return {}
        if found["type"] != "tree":
            raise ValueError("Catalog version path is not a directory")
        tree_sha = found["sha"]
    tree = api.request(f"{repo}/git/trees/{tree_sha}?recursive=1")
    if tree.get("truncated"):
        raise ValueError("Catalog version subtree truncated")
    return {prefix + entry["path"]: entry for entry in tree["tree"] if entry["type"] == "blob"}


def verify_catalog_files(api, repo, commit_sha, prefix, expected):
    actual = catalog_files(api, repo, commit_sha, prefix)
    if actual.keys() != expected.keys():
        raise ValueError(f"Conflicting catalog files {repo}:{prefix}")
    for path, content in expected.items():
        blob = api.request(f"{repo}/git/blobs/{actual[path]['sha']}")
        if base64.b64decode(blob["content"]).decode() != content:
            raise ValueError(f"Conflicting catalog bytes: {repo}:{path}")


def publish(api, bom, archive, fork_owner):
    # Preserve the identity used by the legacy publisher's git commit -s.
    identity = {"name": "instanacd", "email": "instanacd@instana.com"}
    signoff = f"Signed-off-by: {identity['name']} <{identity['email']}>"
    version = bom["version"]
    urls = []
    for owner, repo, operator in CATALOGS:
        upstream, fork = f"/repos/{owner}/{repo}", f"/repos/{fork_owner}/{repo}"
        branch = f"{operator}-{version}"
        prefix = f"operators/{operator}/{version}/"
        expected = bundle_files(archive, version, operator)
        prs = list(api.pages(f"{upstream}/pulls?state=all&head={fork_owner}:{branch}"))
        merged = next((pr for pr in prs if pr.get("merged_at")), None)
        if merged:
            verify_catalog_files(api, upstream, merged["merge_commit_sha"], prefix, expected)
            print(merged["html_url"], flush=True)
            urls.append(merged["html_url"])
            continue
        existing = api.request(f"{fork}/git/ref/heads/{branch}", missing=True)
        if existing:
            verify_catalog_files(api, fork, existing["object"]["sha"], prefix, expected)
        else:
            base_branch = api.request(upstream)["default_branch"]
            base = api.request(f"{upstream}/git/ref/heads/{base_branch}")["object"]["sha"]
            base_commit = api.request(f"{upstream}/git/commits/{base}")
            if catalog_files(api, upstream, base, prefix):
                raise ValueError(f"Catalog version already exists upstream: {repo}:{version}; inspect before resuming")
            tree = api.request(f"{fork}/git/trees", {
                "base_tree": base_commit["tree"]["sha"],
                "tree": [{"path": path, "mode": "100644", "type": "blob", "content": content}
                         for path, content in sorted(expected.items())],
            })
            commit = api.request(f"{fork}/git/commits", {
                "message": f"operator {operator} ({version})\n\n{signoff}",
                "author": identity, "committer": identity,
                "tree": tree["sha"], "parents": [base],
            })
            api.request(f"{fork}/git/refs", {"ref": f"refs/heads/{branch}", "sha": commit["sha"]})
        if prs:
            pr = prs[0]
            if pr["state"] == "closed" and not pr.get("merged_at"):
                raise ValueError(f"Catalog PR closed without merge: {pr['html_url']}")
        else:
            base_branch = api.request(upstream)["default_branch"]
            pr = api.request(f"{upstream}/pulls", {
                "title": f"operator {operator} ({version})", "head": f"{fork_owner}:{branch}", "base": base_branch,
                "body": f"Candidate: `{bom['cos-state-path']}`\n\nOperator digest: "
                        f"`{bom['images']['operator-manifest-digest']}`\n\nPublished by SPS release-promotion.",
            })
        print(pr["html_url"], flush=True)
        urls.append(pr["html_url"])
    return urls


if __name__ == "__main__":
    result = publish(GitHub("https://api.github.com", os.environ["GH_TOKEN"]),
                     json.loads(Path(sys.argv[1]).read_text()), sys.argv[2], sys.argv[3])
    Path(sys.argv[4]).write_text(json.dumps(result, indent=2) + "\n")
