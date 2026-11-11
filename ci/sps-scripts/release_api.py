"""GitHub API helpers for release discovery and resumable publication (stdlib only)."""

import hashlib
import json
import os
from pathlib import Path
import re
import sys
import urllib.error
import urllib.parse
import urllib.request


STABLE = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")


class SafeRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        redirected = super().redirect_request(req, fp, code, msg, headers, newurl)
        if redirected and urllib.parse.urlparse(req.full_url).hostname != urllib.parse.urlparse(newurl).hostname:
            redirected.remove_header("Authorization")
        return redirected


def semver(tag):
    value = tag.removeprefix("v")
    return tuple(map(int, value.split("."))) if STABLE.fullmatch(value) else None


class GitHub:
    def __init__(self, base, token, read_only=False):
        self.base, self.token, self.read_only = base, token, read_only

    def request(self, path, data=None, method=None, missing=False, raw=False, content_type=None):
        if self.read_only and (data is not None or method not in (None, "GET", "HEAD")):
            raise ValueError("GitHub client is read-only; refusing a write request")
        url = path if path.startswith("https://") else self.base + path
        headers = {"Authorization": f"Bearer {self.token}",
                   "Accept": "application/octet-stream" if raw else "application/vnd.github+json"}
        if data is not None:
            if not isinstance(data, bytes):
                data = json.dumps(data).encode()
            headers["Content-Type"] = content_type or "application/json"
        req = urllib.request.Request(url, data=data, headers=headers, method=method)
        try:
            with urllib.request.build_opener(SafeRedirect()).open(req, timeout=120) as response:
                body = response.read()
                return body if raw else (json.loads(body) if body else None)
        except urllib.error.HTTPError as exc:
            if missing and exc.code == 404:
                return None
            # Do not print authenticated URLs, headers, or credentials.
            raise RuntimeError(f"GitHub API {method or 'GET'} failed: HTTP {exc.code}") from None

    def pages(self, path):
        page = 1
        while True:
            values = self.request(f"{path}{'&' if '?' in path else '?'}per_page=100&page={page}")
            if not isinstance(values, list):
                raise ValueError("Expected a paginated API list")
            yield from values
            if len(values) < 100:
                return
            page += 1


def discover(operator, charts, line):
    if not re.fullmatch(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)", line):
        raise ValueError("release-line must be canonical MAJOR.MINOR")
    major, minor = map(int, line.split("."))
    tags = [semver(t["name"]) for t in operator.pages("/repos/instana/instana-agent-operator/tags")
            if t["name"].startswith("v")]
    patches = [v[2] for v in tags if v and v[:2] == (major, minor)]
    version = (major, minor, max(patches) + 1 if patches else 0)
    # A published bundle, not simply a tag, must be present for an upgrade edge.
    released = []
    for release in operator.pages("/repos/instana/instana-agent-operator/releases"):
        v = semver(release["tag_name"])
        if (v and v < version and not release["draft"] and not release["prerelease"]
                and any(a["name"] == f"olm-{'.'.join(map(str, v))}.zip" for a in release["assets"])):
            released.append(v)
    chart_tags = {t["name"] for t in charts.pages("/repos/instana/instana-agent-charts/tags")}
    bare = ".".join(map(str, version))
    if bare in chart_tags:
        raise ValueError(f"Charts tag {bare} already exists; resolve version alignment before building")
    return {"version": bare, "previous-version": ".".join(map(str, max(released))) if released else "",
            "latest-charts-tag": max((t for t in chart_tags if semver(t)), key=semver, default="")}


def ensure_tag(api, repo, tag, sha, create=True):
    path = f"/repos/{repo}/git/ref/tags/{urllib.parse.quote(tag, safe='')}"
    existing = api.request(path, missing=True)
    if existing:
        obj = existing["object"]
        while obj["type"] == "tag":
            obj = api.request(f"/repos/{repo}/git/tags/{obj['sha']}")["object"]
        if obj["type"] != "commit" or obj["sha"] != sha:
            raise ValueError(f"Conflicting tag {repo}:{tag}")
    elif create:
        api.request(f"/repos/{repo}/git/refs", {"ref": f"refs/tags/{tag}", "sha": sha})


def check_not_older_release(api, version):
    releases = api.pages("/repos/instana/instana-agent-operator/releases")
    versions = (semver(release["tag_name"]) for release in releases
                if not release["draft"] and not release["prerelease"])
    published = [parsed for parsed in versions if parsed is not None]
    if published and max(published) > semver(version):
        raise ValueError("A newer version is already public; refusing a downgrading promotion")


def ensure_release(api, bom, files):
    repo = "/repos/instana/instana-agent-operator"
    release = api.request(f"{repo}/releases/tags/{bom['release-tag']}", missing=True)
    if release is None:
        # The tag endpoint may expose published releases only. Authenticated
        # listing is necessary to resume a partially uploaded draft safely.
        release = next((r for r in api.pages(f"{repo}/releases")
                        if r["tag_name"] == bom["release-tag"]), None)
    if release is None:
        release = api.request(f"{repo}/releases", {
            "tag_name": bom["release-tag"], "target_commitish": bom["sources"]["operator-sha"],
            "name": bom["release-tag"], "draft": True, "prerelease": False, "generate_release_notes": True,
        })
    assets = {a["name"]: a for a in api.pages(f"{repo}/releases/{release['id']}/assets")}
    for file in files:
        file = Path(file)
        asset = assets.get(file.name)
        if asset and asset.get("state") == "starter":
            if not release["draft"]:
                raise ValueError(f"Published release has an incomplete {file.name}; manual recovery required")
            # Failed uploads can leave an empty starter asset reserving the name.
            # Only these unfinished draft assets are safe to delete and retry.
            api.request(asset["url"], method="DELETE")
            asset = None
        if asset:
            remote = api.request(asset["url"], raw=True)
            if hashlib.sha256(remote).digest() != hashlib.sha256(file.read_bytes()).digest():
                raise ValueError(f"Conflicting release asset {file.name}; never overwrite")
        else:
            if not release["draft"]:
                raise ValueError(f"Published release is missing {file.name}; manual recovery required")
            api.request(release["upload_url"].split("{")[0] + "?name=" + urllib.parse.quote(file.name),
                        file.read_bytes(), content_type="application/octet-stream")
    if release["draft"]:
        api.request(f"{repo}/releases/{release['id']}", {"draft": False}, method="PATCH")
    print(release["html_url"])


def main():
    command = sys.argv[1]
    api = GitHub("https://api.github.com", os.environ["GH_TOKEN"], read_only=command == "check-tags")
    charts = GitHub("https://github.ibm.com/api/v3", os.environ.get("GHE_TOKEN", ""),
                    read_only=command == "check-tags")
    if command == "versions":
        print(json.dumps(discover(api, charts, sys.argv[2])))
    elif command in ("tags", "check-tags"):
        bom = json.loads(Path(sys.argv[2]).read_text())
        check_not_older_release(api, bom["version"])
        ensure_tag(api, "instana/instana-agent-operator", bom["release-tag"],
                   bom["sources"]["operator-sha"], create=command == "tags")
        ensure_tag(charts, "instana/instana-agent-charts", bom["version"],
                   bom["sources"]["charts-sha"], create=command == "tags")
    elif command == "release":
        ensure_release(api, json.loads(Path(sys.argv[2]).read_text()), sys.argv[3:])
    else:
        raise ValueError(f"Unknown command {command}")


if __name__ == "__main__":
    main()
