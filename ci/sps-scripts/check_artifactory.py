"""Read-only check of the promotion Artifactory credentials and chart repository."""
# (c) Copyright IBM Corp. 2026

import base64
import json
import os
from pathlib import Path
import sys
import urllib.error
import urllib.request

from release_api import semver


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ValueError("Artifactory redirected the credential check; refusing to forward credentials")


def check(credentials, version):
    if not semver(version) or version.startswith("v"):
        raise ValueError("Invalid previous chart version")
    username, password = credentials.get("username"), credentials.get("password")
    if not isinstance(username, str) or not username or not isinstance(password, str) or not password:
        raise ValueError("Artifactory credentials require nonempty username and password")
    repo = "instana-team-release-agent-helm-local"
    path = f"/instana-agent-{version}.tgz"
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
                         "verify credentials, read access and previous chart availability") from None
    except urllib.error.URLError:
        raise ValueError("Artifactory read check failed: connection or TLS error") from None
    if metadata.get("repo") != repo or metadata.get("path") != path:
        raise ValueError("Artifactory returned unexpected chart metadata")
    print(f"Artifactory read check passed: {repo}{path} (upload permissions not tested)")


if __name__ == "__main__":
    check(json.loads(os.environ["TAAS_CREDS"]), json.loads(Path(sys.argv[1]).read_text())["previous-version"])
