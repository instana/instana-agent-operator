#!/usr/bin/env python3
"""Generate a Markdown table of the release timeline across Instana repos.

Reads the last LOOKBACK_MONTHS calendar months of releases from:
  - github.com        instana/instana-agent-operator  (operator image)
  - github.ibm.com    instana/instana-agent-docker    (agent image)
  - github.ibm.com    instana/k8sensor                (k8sensor image)
  - github.ibm.com    instana/instana-agent-charts    (helm chart)

Required env vars: GH_TOKEN, GHE_TOKEN
Output: release-timeline.md in the current working directory.
Only rows where a helm chart or operator image release occurred are included.

Helm chart fill-forward behaviour
----------------------------------
The helm chart column is NEVER fill-forwarded. Each row in the "helm chart"
column only ever contains the version of a chart that was actually released on
that date.  When no chart was released on a given date the cell is empty.

For every helm chart release the operator image version that the chart bundles
is extracted eagerly by running ``helm pull`` and grepping the operator
deployment template.  That extracted version is used to populate the operator
image column for dates where no GitHub operator release happened on the same
day (rendered in italic, as per the existing carry-forward convention).
"""

import calendar
import json
import logging
import os
import re
import shutil
import subprocess
import sys
import tempfile
import urllib.error
import urllib.request
from datetime import date, datetime, timezone
from urllib.parse import urlencode


# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

REPOS = [
    {
        "column": "operator image",
        "host": "github.com",
        "owner": "instana",
        "repo": "instana-agent-operator",
        "token_env": "GH_TOKEN",
        "docker_image": "icr.io/instana/instana-agent-operator",
    },
    {
        "column": "agent image",
        "host": "github.ibm.com",
        "owner": "instana",
        "repo": "instana-agent-docker",
        "token_env": "GHE_TOKEN",
        "docker_image": "containers.instana.io/instana/release/agent/static",
    },
    {
        "column": "k8sensor image",
        "host": "github.ibm.com",
        "owner": "instana",
        "repo": "k8sensor",
        "token_env": "GHE_TOKEN",
        "docker_image": "icr.io/instana/k8sensor",
    },
    {
        "column": "helm chart",
        "host": "github.ibm.com",
        "owner": "instana",
        "repo": "instana-agent-charts",
        "token_env": "GHE_TOKEN",
        "helm_chart": "instana-agent",
        "helm_repo": "https://agents.instana.io/helm",
    },
]

COLUMNS = ["date", "helm chart", "operator image", "agent image", "k8sensor image"]

# Columns that gate row inclusion: a row is only emitted when at least one of
# these columns has a release on that date.
ROW_TRIGGER_COLUMNS = {"helm chart", "operator image"}

# Columns that must never be fill-forwarded; their cells are either an actual
# release value or empty.
NO_FILL_FORWARD_COLS = {"helm chart"}

# Regex to extract the operator version from a rendered helm chart template.
# Matches lines like:
#   image: icr.io/instana/instana-agent-operator:2.2.15
_OPERATOR_IMAGE_RE = re.compile(
    r"image:\s+icr\.io/instana/instana-agent-operator:(\S+)"
)

OUTPUT_STEM = "release-timeline"

LOOKBACK_MONTHS = 6

TIMEOUT = 10  # seconds

log = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def cutoff_date(_today: date | None = None) -> date:
    """Return today's date minus approximately LOOKBACK_MONTHS calendar months.

    The day is clamped to the last valid day of the resulting month
    (e.g. Aug 31 -> Feb 28).

    *_today* is only for testing; leave it as *None* in production.
    """
    assert 0 < LOOKBACK_MONTHS < 12, "LOOKBACK_MONTHS must be between 1 and 11"
    today = _today if _today is not None else date.today()
    month = today.month - LOOKBACK_MONTHS
    year = today.year
    if month <= 0:
        month += 12
        year -= 1
    last_day = calendar.monthrange(year, month)[1]
    day = min(today.day, last_day)
    return date(year, month, day)


def api_url(host: str, owner: str, repo: str) -> str:
    if host == "github.com":
        return f"https://api.github.com/repos/{owner}/{repo}/releases"
    return f"https://{host}/api/v3/repos/{owner}/{repo}/releases"


def _gh_request(url: str, token: str) -> list:
    """Perform a single GitHub API GET and return the parsed JSON list."""
    req = urllib.request.Request(
        url,
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            return json.load(resp)
    except urllib.error.HTTPError as exc:
        if exc.code in (401, 403):
            log.error("Auth failed (HTTP %s): %s", exc.code, url)
            sys.exit(1)
        if exc.code == 404:
            log.error("Not found: %s", url)
            sys.exit(1)
        if exc.code == 429:
            log.error("Rate limited. Check your token or wait before retrying: %s", url)
            sys.exit(1)
        log.error("HTTP %s fetching %s", exc.code, url)
        sys.exit(1)
    except urllib.error.URLError as exc:
        log.exception("Network error fetching %s: %s", url, exc.reason)
        sys.exit(1)


def fetch_releases(
    host: str,
    token: str,
    owner: str,
    repo: str,
    cutoff: date,
    *,
    extra_tag_filter=None,
) -> list[tuple[date, str]]:
    """Fetch all non-draft, non-prerelease releases published on or after *cutoff*.

    *extra_tag_filter*: optional callable that receives a tag (without leading
    ``v``) and returns ``True`` when the release should be excluded silently.
    """
    base = api_url(host, owner, repo)
    results: list[tuple[date, str]] = []
    page = 1

    log.debug("Fetching releases for %s/%s from %s (cutoff: %s)", owner, repo, host, cutoff)

    while True:
        url = f"{base}?{urlencode({'per_page': 100, 'page': page})}"
        log.debug("GET %s", url)
        releases = _gh_request(url, token)

        log.debug("Page %d: received %d release(s)", page, len(releases))

        if not releases:
            break

        stop = False
        for rel in releases:
            if rel.get("draft") or rel.get("prerelease"):
                log.debug("Skipping draft/prerelease: %s", rel.get("tag_name"))
                continue
            published_at = rel.get("published_at", "")
            if not published_at:
                log.debug("Skipping release with no published_at: %s", rel.get("tag_name"))
                continue
            rel_date = datetime.fromisoformat(published_at.replace("Z", "+00:00")).date()
            if rel_date < cutoff:
                stop = True
                continue  # finish the page; releases may not be strictly ordered
            tag = rel["tag_name"].lstrip("v")
            if _is_fedramp(tag):
                log.debug("Skipping FedRAMP release: %s", tag)
                continue
            if extra_tag_filter is not None and extra_tag_filter(tag):
                log.debug("Skipping excluded release: %s", tag)
                continue
            log.debug("Found release %s on %s", tag, rel_date)
            results.append((rel_date, tag))

        if stop or len(releases) < 100:
            break
        page += 1

    log.info("Fetched %d release(s) from %s/%s", len(results), owner, repo)
    return results


def fetch_latest_release(
    host: str,
    token: str,
    owner: str,
    repo: str,
    *,
    extra_tag_filter=None,
) -> str | None:
    """Return the tag of the single most recent non-draft, non-prerelease release.

    Used to seed the fill-forward baseline so that even the earliest rows in the
    table carry a value for every column, even when that release predates the
    lookback window.

    Returns *None* when no qualifying release is found.
    """
    base = api_url(host, owner, repo)
    # Fetch up to two pages (200 releases) to find the first qualifying tag.
    # In practice the latest release is almost always on page 1.
    for page in range(1, 3):
        url = f"{base}?{urlencode({'per_page': 100, 'page': page})}"
        log.debug("fetch_latest_release GET %s", url)
        releases = _gh_request(url, token)
        if not releases:
            break
        for rel in releases:
            if rel.get("draft") or rel.get("prerelease"):
                continue
            published_at = rel.get("published_at", "")
            if not published_at:
                continue
            tag = rel["tag_name"].lstrip("v")
            if _is_fedramp(tag):
                continue
            if extra_tag_filter is not None and extra_tag_filter(tag):
                continue
            log.debug("Latest release for %s/%s: %s", owner, repo, tag)
            return tag
    log.debug("No qualifying release found for %s/%s", owner, repo)
    return None


# ---------------------------------------------------------------------------
# Helm chart → operator version extraction
# ---------------------------------------------------------------------------

_HELM_OPERATOR_TEMPLATE = "instana-agent/templates/operator_deployment_instana-agent-controller-manager.yml"


def extract_operator_from_helm_chart(
    chart: str,
    repo_url: str,
    version: str,
    *,
    _helm_cmd: list[str] | None = None,
) -> str | None:
    """Pull *version* of *chart* from *repo_url* and return the operator image tag it pins.

    Runs ``helm pull`` in a temporary directory, extracts the tarball, reads the
    operator deployment template and returns the version string from the first
    matching ``image: icr.io/instana/instana-agent-operator:<version>`` line.

    Returns *None* when:
    - the ``helm`` binary is not on PATH
    - ``helm pull`` fails (non-zero exit code)
    - the deployment template is absent from the chart
    - no matching image line is found

    *_helm_cmd* overrides the helm command list; used in tests.
    """
    if _helm_cmd is None and shutil.which("helm") is None:
        log.warning("helm binary not found; skipping operator version extraction for chart %s %s", chart, version)
        return None

    cmd = (_helm_cmd or ["helm"]) + [
        "pull",
        "--repo", repo_url,
        chart,
        "--version", version,
        "--untar",
    ]

    with tempfile.TemporaryDirectory() as tmpdir:
        log.debug("helm pull %s %s into %s", chart, version, tmpdir)
        try:
            result = subprocess.run(
                cmd,
                cwd=tmpdir,
                capture_output=True,
                text=True,
                timeout=60,
            )
        except subprocess.TimeoutExpired:
            log.warning("helm pull timed out for %s %s", chart, version)
            return None
        except FileNotFoundError:
            log.warning("helm binary not found while pulling %s %s", chart, version)
            return None

        if result.returncode != 0:
            log.warning(
                "helm pull failed for %s %s (rc=%d): %s",
                chart, version, result.returncode, result.stderr.strip(),
            )
            return None

        template_path = os.path.join(tmpdir, _HELM_OPERATOR_TEMPLATE)
        if not os.path.exists(template_path):
            log.warning("Operator deployment template not found in chart %s %s", chart, version)
            return None

        with open(template_path, encoding="utf-8") as fh:
            for line in fh:
                m = _OPERATOR_IMAGE_RE.search(line)
                if m:
                    op_version = m.group(1)
                    log.debug("Chart %s %s pins operator %s", chart, version, op_version)
                    return op_version

    log.warning("No operator image line found in chart %s %s", chart, version)
    return None


def fetch_helm_operator_pins(
    helm_releases: list[tuple[date, str]],
    chart: str,
    repo_url: str,
    *,
    _helm_cmd: list[str] | None = None,
) -> dict[str, str]:
    """Return a mapping of ``{chart_version: operator_version}`` for all *helm_releases*.

    Calls :func:`extract_operator_from_helm_chart` for each release.  Chart
    versions for which extraction fails are omitted from the result.
    """
    pins: dict[str, str] = {}
    for _rel_date, chart_version in helm_releases:
        op_version = extract_operator_from_helm_chart(chart, repo_url, chart_version, _helm_cmd=_helm_cmd)
        if op_version:
            pins[chart_version] = op_version
    return pins


# ---------------------------------------------------------------------------
# Table building / rendering (pure -- no I/O, easy to test)
# ---------------------------------------------------------------------------

def _is_fedramp(tag: str) -> bool:
    """Return True if *tag* is a FedRAMP variant (contains 'fedramp')."""
    return "fedramp" in tag.lower()


def _is_operator_21x(tag: str) -> bool:
    """Return True if *tag* is a 2.1.x operator release.

    These are FedRAMP-targeted builds not intended for the general audience
    and are silently excluded from the release table.
    """
    parts = tag.split(".")
    return len(parts) >= 2 and parts[0] == "2" and parts[1] == "1"


def latest_releases(
    table: dict[date, dict[str, list[str]]],
    repos: list[dict],
) -> dict[str, str]:
    """Return a mapping of release command -> latest tag/version for command generation.

    For each repo entry that has a ``docker_image`` key the most recent tag is
    used to build a ``docker pull`` command.
    For repo entries that have a ``helm_chart`` key the most recent tag is used
    to build a ``helm pull`` command (value is prefixed with ``helm:`` so the
    caller can distinguish it).

    Returns ``{command_key: tag}`` in display order.
    """
    # Collect all tags per column, sorted newest date first
    col_tags: dict[str, list[str]] = {}
    for d in sorted(table.keys(), reverse=True):
        for col, tags in table[d].items():
            col_tags.setdefault(col, []).extend(tags)

    result: dict[str, str] = {}
    for repo in repos:
        col = repo["column"]
        tags = col_tags.get(col, [])
        latest = next(iter(tags), None)
        if not latest:
            continue

        if "docker_image" in repo:
            result[repo["docker_image"]] = latest

        if "helm_chart" in repo:
            # Encode as "helm:<chart>|<repo>" so render knows the command type
            key = f"helm:{repo['helm_chart']}|{repo['helm_repo']}"
            result[key] = latest

    return result


def render_release_commands(pull_map: dict[str, str]) -> str:
    """Return a Markdown code-block with docker pull / helm pull commands."""
    if not pull_map:
        return ""
    lines = []
    for key, tag in pull_map.items():
        if key.startswith("helm:"):
            _, rest = key.split(":", 1)
            chart, repo_url = rest.split("|", 1)
            lines.append(f"helm pull --repo {repo_url} {chart} --version {tag}")
        else:
            lines.append(f"docker pull {key}:{tag}")
    return "```\n" + "\n".join(lines) + "\n```\n"


def build_table(
    all_releases: list[tuple[str, date, str]],
) -> dict[date, dict[str, list[str]]]:
    """Aggregate *(column, date, tag)* triples into a nested dict.

    Returns ``{date: {column: [tag, ...]}}`` with entries in insertion order.
    """
    table: dict[date, dict[str, list[str]]] = {}
    for col, rel_date, tag in all_releases:
        row = table.setdefault(rel_date, {})
        row.setdefault(col, []).append(tag)
    return table


def filter_table(
    table: dict[date, dict[str, list[str]]],
    trigger_columns: set[str],
) -> dict[date, dict[str, list[str]]]:
    """Return a copy of *table* keeping only dates that have at least one
    release in any of the *trigger_columns*."""
    return {
        d: row
        for d, row in table.items()
        if any(row.get(col) for col in trigger_columns)
    }


def fill_forward(
    table: dict[date, dict[str, list[str]]],
    columns: list[str],
    baseline: dict[str, str],
    no_fill_forward_cols: set[str] | None = None,
    helm_injected_operator: set[tuple[date, str]] | None = None,
) -> dict[date, dict[str, tuple[list[str], bool]]]:
    """Return a new table with carry-forward values for blank artifact columns.

    For each row and each artifact column (all columns except "date"), if the
    column has no release on that date, the most recent tag seen so far
    (scanning from oldest to newest) is used instead.  The boolean in the tuple
    indicates whether the value was carried forward (``True``) or was an actual
    release (``False``).

    *baseline* maps column name -> tag for the pre-window most-recent release;
    it seeds the carry-forward so even the oldest rows in the table have a value.

    *no_fill_forward_cols* names columns that must never be fill-forwarded.
    For those columns an empty cell stays empty regardless of prior releases.

    *helm_injected_operator* is a set of ``(date, operator_version)`` pairs for
    operator versions that were extracted from helm chart tarballs rather than
    from a GitHub operator release.  These are rendered in italic even though
    they are present as actual table entries, because no operator GitHub release
    happened on those dates.

    Returns ``{date: {column: ([tag, ...], is_carried_forward)}}``.
    """
    artifact_cols = [c for c in columns if c != "date"]
    _no_ff = no_fill_forward_cols or set()
    _helm_injected = helm_injected_operator or set()

    # latest[col] holds the single most-recent tag seen so far (oldest→newest)
    latest: dict[str, str | None] = {col: baseline.get(col) for col in artifact_cols}

    result: dict[date, dict[str, tuple[list[str], bool]]] = {}
    for d in sorted(table.keys()):
        row = table[d]
        filled: dict[str, tuple[list[str], bool]] = {}
        for col in artifact_cols:
            tags = row.get(col, [])
            if tags:
                # Update the latest seen tag for carry-forward in later rows.
                latest[col] = tags[0]  # newest tag on this date (first = most recent)
                # Check if this entry was injected from a helm chart tarball.
                # If so, mark it as carried (italic) so the table is honest.
                if col == "operator image" and any((d, t) in _helm_injected for t in tags):
                    filled[col] = (tags, True)
                else:
                    filled[col] = (tags, False)
            elif col in _no_ff:
                # Never fill-forward: leave empty
                filled[col] = ([], False)
            elif latest[col] is not None:
                filled[col] = ([latest[col]], True)
            else:
                filled[col] = ([], False)
        result[d] = filled

    return result


def render_markdown(
    table: dict[date, dict[str, list[str]]],
    columns: list[str],
    lookback_months: int,
    generated_date: datetime,
    repos: list[dict] | None = None,
    baseline: dict[str, str] | None = None,
    no_fill_forward_cols: set[str] | None = None,
    helm_injected_operator: set[tuple[date, str]] | None = None,
    trigger_columns: set[str] | None = None,
) -> str:
    """Render *table* as a Markdown document string.

    *trigger_columns*: when provided, only rows that have at least one actual
    release in one of these columns are emitted.  Fill-forward and
    ``latest_releases`` always operate on the full *table* first so that
    agent/k8sensor releases on non-trigger dates are still carried forward
    correctly into the visible rows.
    """
    # latest_releases and fill_forward must see ALL dates so that releases on
    # non-trigger dates (e.g. agent-only release days) are not silently dropped
    # from the carry-forward chain.
    pull_section = ""
    if repos:
        pull_map = latest_releases(table, repos)
        if pull_map:
            pull_section = (
                "\n## Latest Release Artifacts\n\n"
                + render_release_commands(pull_map)
            )

    filled = fill_forward(table, columns, baseline or {}, no_fill_forward_cols, helm_injected_operator)

    # Now restrict which dates appear in the rendered table.
    if trigger_columns:
        visible_dates = sorted(
            (d for d in table if any(table[d].get(col) for col in trigger_columns)),
            reverse=True,
        )
    else:
        visible_dates = sorted(table.keys(), reverse=True)

    header = "| " + " | ".join(columns) + " |"
    separator = "|" + "|".join("---" for _ in columns) + "|"

    lines = [
        "# Release Timeline",
        "",
        f"Generated: {generated_date.strftime('%Y-%m-%dT%H:%M:%S UTC')}",
        "",
        "> **Note:** There is no strict mapping between versions across projects.",
        "> This list provides an overview of what projects were released at which point in time.",
        "",
    ]

    if pull_section:
        lines.append(pull_section)

    lines += [
        "## Release Table",
        "",
        f"Releases from the last {lookback_months} months. Each row represents a date on which at least one",
        "artifact published a release. Italic values mean no release was published on that exact date but",
        "the most recent available version is shown instead.",
        "Rows only include artifacts when a helm chart or operator image release occurred, intermediate agent image",
        "and k8sensor image releases that were superseded before a new helm chart or operator image shipped",
        "are not listed as separate rows.",
        "",
        header,
        separator,
    ]

    for d in visible_dates:
        row_filled = filled[d]
        cells = [d.isoformat()]
        for col in columns[1:]:
            tags, carried = row_filled.get(col, ([], False))
            if not tags:
                cells.append("")
            elif carried:
                cells.append(f"_{tags[0]}_")
            else:
                cells.append(", ".join(tags))
        lines.append("| " + " | ".join(cells) + " |")

    return "\n".join(lines) + "\n"


# ---------------------------------------------------------------------------
# Token resolution
# ---------------------------------------------------------------------------

# Maps the token env-var name to the GitHub hostname used for gh CLI fallback.
_TOKEN_HOST: dict[str, str] = {
    "GH_TOKEN": "github.com",
    "GHE_TOKEN": "github.ibm.com",
}


def gh_auth_token(hostname: str) -> str | None:
    """Return the token from ``gh auth token --hostname <hostname>``, or *None*.

    Returns *None* when the ``gh`` binary is not found, the user is not
    authenticated, or the command fails for any other reason.
    """
    try:
        result = subprocess.run(
            ["gh", "auth", "token", "--hostname", hostname],
            capture_output=True,
            text=True,
            timeout=10,
        )
        if result.returncode == 0:
            token = result.stdout.strip()
            if token:
                return token
        log.debug("gh auth token --hostname %s failed (rc=%d): %s", hostname, result.returncode, result.stderr.strip())
    except FileNotFoundError:
        log.debug("gh CLI not found; skipping fallback for %s", hostname)
    except subprocess.TimeoutExpired:
        log.debug("gh auth token timed out for %s", hostname)
    return None


def resolve_tokens() -> dict[str, str]:
    """Return a mapping of ``{token_env_var: token}`` for all required vars.

    Env vars take precedence; if absent, falls back to ``gh auth token``.
    Exits with an error when a token cannot be resolved by either method.
    """
    tokens: dict[str, str] = {}
    missing: list[str] = []

    for var, hostname in _TOKEN_HOST.items():
        val = os.environ.get(var, "").strip()
        if val:
            log.debug("Using %s from environment", var)
            tokens[var] = val
            continue
        log.debug("%s not set; trying gh auth token --hostname %s", var, hostname)
        val = gh_auth_token(hostname)
        if val:
            log.info("Resolved %s via gh CLI for %s", var, hostname)
            tokens[var] = val
        else:
            missing.append(var)

    if missing:
        log.error(
            "Could not resolve token(s) for: %s. "
            "Set the environment variable(s) or run: gh auth login --hostname <host>",
            ", ".join(missing),
        )
        sys.exit(1)

    return tokens


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    logging.basicConfig(
        level=os.environ.get("LOG_LEVEL", "INFO").upper(),
        format="%(asctime)s %(levelname)s %(message)s",
        datefmt="%Y-%m-%dT%H:%M:%S",
    )

    tokens = resolve_tokens()

    cutoff = cutoff_date()
    log.info("Collecting releases since %s (%d months)", cutoff, LOOKBACK_MONTHS)

    # Collect (column, date, tag) triples and the pre-window baseline per column.
    # Also track helm chart releases separately so we can extract their operator pins.
    all_releases: list[tuple[str, date, str]] = []
    baseline: dict[str, str] = {}
    helm_releases_by_cfg: dict[int, list[tuple[date, str]]] = {}  # repo_cfg index → releases

    for idx, repo_cfg in enumerate(REPOS):
        token = tokens[repo_cfg["token_env"]]
        extra_filter = _is_operator_21x if repo_cfg["repo"] == "instana-agent-operator" else None
        releases = fetch_releases(
            repo_cfg["host"], token, repo_cfg["owner"], repo_cfg["repo"], cutoff,
            extra_tag_filter=extra_filter,
        )
        col = repo_cfg["column"]
        for rel_date, tag in releases:
            all_releases.append((col, rel_date, tag))

        if "helm_chart" in repo_cfg:
            helm_releases_by_cfg[idx] = releases

        # Fetch the unconditional most-recent release to seed fill-forward.
        # Helm chart column is never fill-forwarded, so skip seeding its baseline.
        if col not in NO_FILL_FORWARD_COLS:
            latest_tag = fetch_latest_release(
                repo_cfg["host"], token, repo_cfg["owner"], repo_cfg["repo"],
                extra_tag_filter=extra_filter,
            )
            if latest_tag:
                baseline[col] = latest_tag
                log.debug("Baseline for %s: %s", col, latest_tag)

    # For each helm chart release, eagerly extract the operator version it pins.
    # Inject synthetic operator image entries for dates that have no GitHub
    # operator release, so the operator column is populated on helm-chart-only rows.
    # Track which (date, operator_version) pairs were injected so fill_forward can
    # render them in italic (no GitHub operator release on those dates).
    github_operator_dates: set[date] = {
        rel_date for col, rel_date, _ in all_releases if col == "operator image"
    }
    helm_injected_operator: set[tuple[date, str]] = set()
    for idx, helm_releases in helm_releases_by_cfg.items():
        repo_cfg = REPOS[idx]
        pins = fetch_helm_operator_pins(
            helm_releases,
            repo_cfg["helm_chart"],
            repo_cfg["helm_repo"],
        )
        for rel_date, chart_version in helm_releases:
            op_version = pins.get(chart_version)
            if op_version and rel_date not in github_operator_dates:
                log.debug(
                    "Injecting operator %s for helm chart %s on %s (no GitHub release that day)",
                    op_version, chart_version, rel_date,
                )
                all_releases.append(("operator image", rel_date, op_version))
                helm_injected_operator.add((rel_date, op_version))

    table = build_table(all_releases)
    log.info("Building table with %d row(s) before trigger filter", len(table))

    now = datetime.now(timezone.utc)
    filename = f"{OUTPUT_STEM}.md"
    content = render_markdown(
        table, COLUMNS, LOOKBACK_MONTHS, now, REPOS, baseline,
        NO_FILL_FORWARD_COLS, helm_injected_operator,
        trigger_columns=ROW_TRIGGER_COLUMNS,
    )
    with open(filename, "w", encoding="utf-8") as fh:
        fh.write(content)
    log.info("Written %s (%d rows)", filename, len(table))


if __name__ == "__main__":
    main()
