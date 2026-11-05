"""Read-only check of the promotion Artifactory credentials and chart repository."""
# (c) Copyright IBM Corp. 2026

import base64
import json
import os
import urllib.error
import urllib.request

from release_api import GitHub, semver


def latest_chart_version(charts):
    versions = [semver(tag["name"]) for tag in charts.pages("/repos/instana/instana-agent-charts/tags")]
    stable = [version for version in versions if version is not None]
    if not stable:
        raise ValueError("No stable Helm chart release tag found")
    return ".".join(map(str, max(stable)))

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ValueError("Artifactory redirected the credential check; refusing to forward credentials")


def check(credentials, charts):
    username, password = credentials.get("username"), credentials.get("password")
    if not isinstance(username, str) or not username or not isinstance(password, str) or not password:
        raise ValueError("Artifactory credentials require nonempty username and password")
    repo = "instana-team-release-agent-helm-local"
    version = latest_chart_version(charts)
    path = f"/instana-agent-{version}.tgz"
    print(f"Artifactory read check: latest stable chart tag resolves to {version}")
    url = f"https://na.artifactory.swg-devops.com/artifactory/api/storage/{repo}{path}"
    auth = base64.b64encode(f"{username}:{password}".encode()).decode()
    request = urllib.request.Request(url, headers={"Authorization": f"Basic {auth}"}, method="GET")
    try:
        with urllib.request.build_opener(NoRedirect()).open(request, timeout=60) as response:
            if response.status != 200:
                raise ValueError("Artifactory metadata check requires HTTP 200")
            metadata = json.load(response)
    except urllib.error.HTTPError as error:
        raise ValueError(f"Artifactory read check failed: HTTP {error.code}; "
                         f"verify credentials, metadata read access and chart {version} availability") from None
    except urllib.error.URLError:
        raise ValueError("Artifactory read check failed: connection or TLS error") from None
    if metadata.get("repo") != repo or metadata.get("path") != path:
        raise ValueError("Artifactory returned unexpected chart metadata")
    print(f"Artifactory read check passed: {repo}{path} (upload permissions not tested)")


if __name__ == "__main__":
    check(json.loads(os.environ["TAAS_CREDS"]),
          GitHub("https://github.ibm.com/api/v3", os.environ["GHE_TOKEN"], read_only=True))
