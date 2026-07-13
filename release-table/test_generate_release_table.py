"""Tests for generate-release-table.py

Run with:  python3 -m unittest test_generate_release_table -v
"""

import importlib
import io
import json
import sys
import unittest
import urllib.error
from datetime import date, datetime, timezone
from unittest.mock import MagicMock, patch

# ---------------------------------------------------------------------------
# Import the module under test.
# The filename contains a hyphen, so standard import won't work.
# ---------------------------------------------------------------------------
import importlib.util, pathlib

_spec = importlib.util.spec_from_file_location(
    "generate_release_table",
    pathlib.Path(__file__).parent / "generate-release-table.py",
)
_mod = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(_mod)

cutoff_date = _mod.cutoff_date
api_url = _mod.api_url
fetch_releases = _mod.fetch_releases
build_table = _mod.build_table
filter_table = _mod.filter_table
fill_forward = _mod.fill_forward
render_markdown = _mod.render_markdown
_is_fedramp = _mod._is_fedramp
_is_operator_21x = _mod._is_operator_21x
latest_releases = _mod.latest_releases
render_release_commands = _mod.render_release_commands
gh_auth_token = _mod.gh_auth_token
resolve_tokens = _mod.resolve_tokens
extract_operator_from_helm_chart = _mod.extract_operator_from_helm_chart
fetch_helm_operator_pins = _mod.fetch_helm_operator_pins
NO_FILL_FORWARD_COLS = _mod.NO_FILL_FORWARD_COLS


# ---------------------------------------------------------------------------
# gh_auth_token / resolve_tokens
# ---------------------------------------------------------------------------

class TestGhAuthToken(unittest.TestCase):

    def _run(self, returncode, stdout="", stderr=""):
        mock_result = MagicMock()
        mock_result.returncode = returncode
        mock_result.stdout = stdout
        mock_result.stderr = stderr
        return mock_result

    def test_returns_token_on_success(self):
        with patch("subprocess.run", return_value=self._run(0, "mytoken\n")):
            self.assertEqual(gh_auth_token("github.com"), "mytoken")

    def test_returns_none_on_nonzero_exit(self):
        with patch("subprocess.run", return_value=self._run(1, "", "not logged in")):
            self.assertIsNone(gh_auth_token("github.com"))

    def test_returns_none_when_stdout_empty(self):
        with patch("subprocess.run", return_value=self._run(0, "")):
            self.assertIsNone(gh_auth_token("github.com"))

    def test_returns_none_when_gh_not_found(self):
        with patch("subprocess.run", side_effect=FileNotFoundError):
            self.assertIsNone(gh_auth_token("github.com"))

    def test_returns_none_on_timeout(self):
        import subprocess as _sp
        with patch("subprocess.run", side_effect=_sp.TimeoutExpired(cmd="gh", timeout=10)):
            self.assertIsNone(gh_auth_token("github.com"))


class TestResolveTokens(unittest.TestCase):

    def test_env_takes_precedence_over_gh_cli(self):
        env = {"GH_TOKEN": "env-gh", "GHE_TOKEN": "env-ghe"}
        with patch.dict("os.environ", env, clear=False):
            with patch("subprocess.run") as mock_run:
                tokens = resolve_tokens()
        mock_run.assert_not_called()
        self.assertEqual(tokens["GH_TOKEN"], "env-gh")
        self.assertEqual(tokens["GHE_TOKEN"], "env-ghe")

    def test_falls_back_to_gh_cli_when_env_missing(self):
        mock_result = MagicMock()
        mock_result.returncode = 0
        mock_result.stdout = "cli-token\n"
        mock_result.stderr = ""
        env = {}
        with patch.dict("os.environ", env):
            with patch("os.environ.get", side_effect=lambda k, d="": ""):
                with patch("subprocess.run", return_value=mock_result):
                    with patch("subprocess.run", return_value=mock_result):
                        tokens = resolve_tokens()
        self.assertEqual(tokens["GH_TOKEN"], "cli-token")
        self.assertEqual(tokens["GHE_TOKEN"], "cli-token")

    def test_exits_when_both_env_and_gh_cli_missing(self):
        with patch("os.environ.get", return_value=""):
            with patch("subprocess.run", side_effect=FileNotFoundError):
                with self.assertRaises(SystemExit):
                    resolve_tokens()

    def test_exits_when_only_one_token_missing(self):
        def _fake_get(key, default=""):
            return "good-token" if key == "GH_TOKEN" else ""

        with patch("os.environ.get", side_effect=_fake_get):
            with patch("subprocess.run", side_effect=FileNotFoundError):
                with self.assertRaises(SystemExit):
                    resolve_tokens()


# ---------------------------------------------------------------------------
# cutoff_date
# ---------------------------------------------------------------------------

class TestCutoffDate(unittest.TestCase):

    def _cutoff(self, today: date) -> date:
        """Call cutoff_date() with a fixed *today* via the test-only parameter."""
        return cutoff_date(_today=today)

    def test_simple_subtraction(self):
        result = self._cutoff(date(2025, 8, 15))
        self.assertEqual(result, date(2025, 2, 15))

    def test_year_rollover(self):
        # January minus 6 months → July of the previous year
        result = self._cutoff(date(2025, 1, 20))
        self.assertEqual(result, date(2024, 7, 20))

    def test_day_clamped_to_month_end(self):
        # Aug 31 − 6 months → Feb; Feb only has 28 days in 2025
        result = self._cutoff(date(2025, 8, 31))
        self.assertEqual(result, date(2025, 2, 28))

    def test_leap_year_feb(self):
        # Aug 31, 2024 − 6 months → Feb 29 (2024 is a leap year)
        result = self._cutoff(date(2024, 8, 31))
        self.assertEqual(result, date(2024, 2, 29))

    def test_exactly_six_months_from_july(self):
        result = self._cutoff(date(2025, 7, 1))
        self.assertEqual(result, date(2025, 1, 1))


# ---------------------------------------------------------------------------
# api_url
# ---------------------------------------------------------------------------

class TestApiUrl(unittest.TestCase):

    def test_github_com(self):
        url = api_url("github.com", "instana", "instana-agent-operator")
        self.assertEqual(
            url,
            "https://api.github.com/repos/instana/instana-agent-operator/releases",
        )

    def test_github_enterprise(self):
        url = api_url("github.ibm.com", "instana", "k8sensor")
        self.assertEqual(
            url,
            "https://github.ibm.com/api/v3/repos/instana/k8sensor/releases",
        )


# ---------------------------------------------------------------------------
# fetch_releases
# ---------------------------------------------------------------------------

def _make_response(payload: list) -> MagicMock:
    """Return a mock that behaves like urllib.request.urlopen()'s context manager."""
    body = json.dumps(payload).encode()
    resp = MagicMock()
    resp.__enter__ = MagicMock(return_value=resp)
    resp.__exit__ = MagicMock(return_value=False)
    resp.read = MagicMock(return_value=body)
    # json.load uses resp.read() internally via the file-like interface
    resp.readall = MagicMock(return_value=body)
    # Make json.load work: provide a real file-like read()
    resp.read.side_effect = None
    resp.read.return_value = body
    # Simplest: patch json.load to use our payload directly
    return resp, payload


class TestFetchReleases(unittest.TestCase):

    def _urlopen_side_effect(self, pages: list[list]):
        """Return a side_effect callable that yields successive page payloads."""
        call_count = [0]

        def _side_effect(req, timeout=None):
            idx = call_count[0]
            call_count[0] += 1
            payload = pages[idx] if idx < len(pages) else []
            body = json.dumps(payload).encode()
            resp = MagicMock()
            resp.__enter__ = MagicMock(return_value=resp)
            resp.__exit__ = MagicMock(return_value=False)
            # Provide a real BytesIO so json.load works
            resp.read = io.BytesIO(body).read
            return resp

        return _side_effect

    def test_basic_filtering(self):
        """Draft and prerelease entries are excluded; entries before cutoff are excluded.
        Tag names are returned without a leading 'v'."""
        pages = [
            [
                {"tag_name": "v1.0", "published_at": "2025-05-01T00:00:00Z", "draft": False, "prerelease": False},
                {"tag_name": "v0.9", "published_at": "2024-01-01T00:00:00Z", "draft": False, "prerelease": False},  # before cutoff
                {"tag_name": "v1.1-rc", "published_at": "2025-05-10T00:00:00Z", "draft": False, "prerelease": True},
                {"tag_name": "v1.1-draft", "published_at": "2025-05-10T00:00:00Z", "draft": True, "prerelease": False},
            ]
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        # 'v' prefix is stripped
        self.assertEqual(results, [(date(2025, 5, 1), "1.0")])

    def test_v_prefix_stripped(self):
        """Leading 'v' is stripped from tag names returned by fetch_releases."""
        pages = [
            [{"tag_name": "v2.3.4", "published_at": "2025-05-01T00:00:00Z", "draft": False, "prerelease": False}]
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        self.assertEqual(results[0][1], "2.3.4")

    def test_tag_without_v_unchanged(self):
        """Tags without a leading 'v' (e.g. helm chart versions) are returned as-is."""
        pages = [
            [{"tag_name": "2.0.47", "published_at": "2025-05-01T00:00:00Z", "draft": False, "prerelease": False}]
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        self.assertEqual(results[0][1], "2.0.47")

    def test_pagination(self):
        """A second page is fetched when the first page has exactly 100 items."""
        page1 = [
            {"tag_name": f"v{i}", "published_at": "2025-05-01T00:00:00Z", "draft": False, "prerelease": False}
            for i in range(100)
        ]
        page2 = [
            {"tag_name": "v100", "published_at": "2025-05-02T00:00:00Z", "draft": False, "prerelease": False},
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect([page1, page2])):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        self.assertEqual(len(results), 101)

    def test_empty_response_stops_pagination(self):
        pages = [[]]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        self.assertEqual(results, [])

    def test_missing_published_at_skipped(self):
        pages = [
            [{"tag_name": "v1.0", "published_at": "", "draft": False, "prerelease": False}]
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        self.assertEqual(results, [])

    def test_http_401_exits(self):
        err = urllib.error.HTTPError(url="", code=401, msg="Unauthorized", hdrs=None, fp=None)
        with patch("urllib.request.urlopen", side_effect=err):
            with self.assertRaises(SystemExit):
                fetch_releases("github.com", "bad-tok", "org", "repo", date(2025, 1, 1))

    def test_http_404_exits(self):
        err = urllib.error.HTTPError(url="", code=404, msg="Not Found", hdrs=None, fp=None)
        with patch("urllib.request.urlopen", side_effect=err):
            with self.assertRaises(SystemExit):
                fetch_releases("github.com", "tok", "org", "missing-repo", date(2025, 1, 1))

    def test_extra_tag_filter_excludes_matching_tags(self):
        """extra_tag_filter callable can exclude additional tags."""
        pages = [
            [
                {"tag_name": "v2.1.44", "published_at": "2025-05-01T00:00:00Z", "draft": False, "prerelease": False},
                {"tag_name": "v2.2.0", "published_at": "2025-05-02T00:00:00Z", "draft": False, "prerelease": False},
            ]
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases(
                "github.com", "tok", "org", "repo", cutoff,
                extra_tag_filter=_is_operator_21x,
            )
        tags = [tag for _, tag in results]
        self.assertNotIn("2.1.44", tags)
        self.assertIn("2.2.0", tags)

    def test_extra_tag_filter_none_applies_no_extra_exclusion(self):
        """Without extra_tag_filter, 2.1.x tags are not excluded."""
        pages = [
            [
                {"tag_name": "v2.1.44", "published_at": "2025-05-01T00:00:00Z", "draft": False, "prerelease": False},
            ]
        ]
        cutoff = date(2025, 1, 1)
        with patch("urllib.request.urlopen", side_effect=self._urlopen_side_effect(pages)):
            results = fetch_releases("github.com", "tok", "org", "repo", cutoff)
        self.assertEqual(results, [(date(2025, 5, 1), "2.1.44")])


# ---------------------------------------------------------------------------
# build_table
# ---------------------------------------------------------------------------

class TestBuildTable(unittest.TestCase):

    def test_empty_input(self):
        self.assertEqual(build_table([]), {})

    def test_groups_by_date(self):
        d1 = date(2025, 5, 1)
        d2 = date(2025, 5, 3)
        releases = [
            ("operator image", d1, "1.0"),
            ("agent image", d1, "2.0"),
            ("operator image", d2, "1.1"),
        ]
        table = build_table(releases)
        self.assertEqual(set(table.keys()), {d1, d2})
        self.assertEqual(table[d1]["operator image"], ["1.0"])
        self.assertEqual(table[d1]["agent image"], ["2.0"])
        self.assertEqual(table[d2]["operator image"], ["1.1"])

    def test_multiple_tags_same_date_and_column(self):
        d = date(2025, 5, 1)
        releases = [
            ("helm chart", d, "3.0"),
            ("helm chart", d, "3.1"),
        ]
        table = build_table(releases)
        self.assertEqual(table[d]["helm chart"], ["3.0", "3.1"])


# ---------------------------------------------------------------------------
# filter_table
# ---------------------------------------------------------------------------

class TestFilterTable(unittest.TestCase):

    TRIGGERS = {"helm chart", "operator image"}

    def test_row_with_helm_chart_kept(self):
        d = date(2025, 5, 1)
        table = {d: {"helm chart": ["2.0.47"]}}
        result = filter_table(table, self.TRIGGERS)
        self.assertIn(d, result)

    def test_row_with_operator_kept(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["2.1.44"]}}
        result = filter_table(table, self.TRIGGERS)
        self.assertIn(d, result)

    def test_row_with_only_agent_dropped(self):
        d = date(2025, 5, 2)
        table = {d: {"agent image": ["1.320.5"]}}
        result = filter_table(table, self.TRIGGERS)
        self.assertNotIn(d, result)

    def test_row_with_only_k8sensor_dropped(self):
        d = date(2025, 5, 3)
        table = {d: {"k8sensor image": ["1.4.19"]}}
        result = filter_table(table, self.TRIGGERS)
        self.assertNotIn(d, result)

    def test_mixed_dates_filtered_correctly(self):
        d_helm = date(2025, 5, 1)
        d_agent = date(2025, 5, 2)
        table = {
            d_helm: {"helm chart": ["2.0.47"], "agent image": ["1.320.5"]},
            d_agent: {"agent image": ["1.320.6"]},
        }
        result = filter_table(table, self.TRIGGERS)
        self.assertIn(d_helm, result)
        self.assertNotIn(d_agent, result)

    def test_empty_table_returns_empty(self):
        self.assertEqual(filter_table({}, self.TRIGGERS), {})

    def test_row_with_empty_trigger_column_dropped(self):
        d = date(2025, 5, 1)
        # helm chart key present but empty list
        table = {d: {"helm chart": [], "agent image": ["1.320.5"]}}
        result = filter_table(table, self.TRIGGERS)
        self.assertNotIn(d, result)


# ---------------------------------------------------------------------------
# render_markdown
# ---------------------------------------------------------------------------

class TestRenderMarkdown(unittest.TestCase):

    COLS = ["date", "helm chart", "operator image", "agent image"]

    def test_empty_table_has_no_data_rows(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        lines = output.splitlines()
        # Header + separator = 2 table lines, no data rows
        table_lines = [ln for ln in lines if ln.startswith("|")]
        self.assertEqual(len(table_lines), 2)

    def test_release_table_heading_present(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertIn("## Release Table", output)

    def test_rows_sorted_descending(self):
        d1 = date(2025, 3, 1)
        d2 = date(2025, 5, 1)
        table = {d1: {"operator image": ["1.0"]}, d2: {"helm chart": ["2.0"]}}
        output = render_markdown(table, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        lines = [ln for ln in output.splitlines() if ln.startswith("| 202")]
        self.assertTrue(lines[0].startswith("| 2025-05-01"))
        self.assertTrue(lines[1].startswith("| 2025-03-01"))

    def test_missing_column_renders_empty_cell_when_no_baseline(self):
        d = date(2025, 5, 1)
        # Only operator image present; no baseline → helm chart and agent image empty
        table = {d: {"operator image": ["1.0"]}}
        output = render_markdown(table, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        data_row = [ln for ln in output.splitlines() if ln.startswith("| 2025")][0]
        # helm chart empty, then operator image 1.0, then agent image empty
        self.assertIn("|  | 1.0 |  |", data_row)

    def test_missing_column_renders_italic_when_baseline_present(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["1.0"]}}
        baseline = {"helm chart": "2.0.40", "agent image": "1.300.0"}
        output = render_markdown(
            table, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc), baseline=baseline
        )
        data_row = [ln for ln in output.splitlines() if ln.startswith("| 2025")][0]
        # helm chart carry-forward italic, operator actual, agent carry-forward italic
        self.assertIn("| _2.0.40_ | 1.0 | _1.300.0_ |", data_row)

    def test_multiple_tags_joined_with_comma(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["1.0", "1.1"], "agent image": []}}
        output = render_markdown(table, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        data_row = [l for l in output.splitlines() if l.startswith("| 2025")][0]
        self.assertIn("1.0, 1.1", data_row)

    def test_intro_text_mentions_italic_convention(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertIn("Italic values mean no release was published on that date", output)

    def test_generated_date_in_header(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertIn("Generated: 2025-06-01T12:00:00 UTC", output)

    def test_lookback_months_in_preamble(self):
        output = render_markdown({}, self.COLS, 3, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertIn("last 3 months", output)

    def test_output_ends_with_newline(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertTrue(output.endswith("\n"))

    def test_no_commands_section_without_repos(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertNotIn("docker pull", output)
        self.assertNotIn("helm pull", output)

    def test_commands_section_with_docker_repos(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["2.1.44"], "agent image": ["1.320.5"]}}
        repos = [
            {"column": "operator image", "docker_image": "icr.io/instana/instana-agent-operator"},
            {"column": "agent image", "docker_image": "containers.instana.io/instana/release/agent/static"},
        ]
        output = render_markdown(table, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc), repos)
        self.assertIn("## Latest Release Artifacts", output)
        self.assertIn("docker pull icr.io/instana/instana-agent-operator:2.1.44", output)
        self.assertIn("docker pull containers.instana.io/instana/release/agent/static:1.320.5", output)
        # commands section must appear before the table
        self.assertLess(output.index("docker pull"), output.index("| date |"))

    def test_commands_section_with_helm(self):
        COLS = ["date", "helm chart", "operator image", "agent image", "k8sensor image"]
        d = date(2025, 5, 1)
        table = {d: {"helm chart": ["2.0.47"]}}
        repos = [{"column": "helm chart", "helm_chart": "instana-agent", "helm_repo": "https://agents.instana.io/helm"}]
        output = render_markdown(table, COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc), repos)
        self.assertIn("## Latest Release Artifacts", output)
        self.assertIn(
            "helm pull --repo https://agents.instana.io/helm instana-agent --version 2.0.47",
            output,
        )
        # must appear before the table
        self.assertLess(output.index("helm pull"), output.index("| date |"))

    def test_no_fedramp_note(self):
        output = render_markdown({}, self.COLS, 6, datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc))
        self.assertNotIn("FedRAMP", output)


# ---------------------------------------------------------------------------
# _is_fedramp
# ---------------------------------------------------------------------------

class TestIsFedramp(unittest.TestCase):

    def test_fedramp_tag_detected(self):
        self.assertTrue(_is_fedramp("1.320.4.fedramp-1.0.26"))

    def test_normal_tag_not_fedramp(self):
        self.assertFalse(_is_fedramp("1.320.5"))

    def test_case_insensitive(self):
        self.assertTrue(_is_fedramp("1.320.4.FedRAMP-1.0.26"))


# ---------------------------------------------------------------------------
# _is_operator_21x
# ---------------------------------------------------------------------------

class TestIsOperator21x(unittest.TestCase):

    def test_exact_21x_excluded(self):
        self.assertTrue(_is_operator_21x("2.1.0"))

    def test_21_patch_variant_excluded(self):
        self.assertTrue(_is_operator_21x("2.1.44"))

    def test_21_extra_segments_excluded(self):
        self.assertTrue(_is_operator_21x("2.1.44.1"))

    def test_20x_not_excluded(self):
        self.assertFalse(_is_operator_21x("2.0.9"))

    def test_22x_not_excluded(self):
        self.assertFalse(_is_operator_21x("2.2.0"))

    def test_30x_not_excluded(self):
        self.assertFalse(_is_operator_21x("3.1.0"))

    def test_1x_not_excluded(self):
        self.assertFalse(_is_operator_21x("1.1.5"))

    def test_short_tag_not_excluded(self):
        # Single-segment tag like "2" should not match
        self.assertFalse(_is_operator_21x("2"))


# ---------------------------------------------------------------------------
# latest_releases
# ---------------------------------------------------------------------------

class TestLatestReleases(unittest.TestCase):

    REPOS = [
        {"column": "operator image", "docker_image": "icr.io/instana/instana-agent-operator"},
        {"column": "agent image", "docker_image": "containers.instana.io/instana/release/agent/static"},
        {"column": "k8sensor image", "docker_image": "icr.io/instana/k8sensor"},
        {
            "column": "helm chart",
            "helm_chart": "instana-agent",
            "helm_repo": "https://agents.instana.io/helm",
        },
        {"column": "no-key col"},  # no docker_image or helm_chart → must be skipped
    ]

    def test_picks_latest_docker_tag(self):
        d1 = date(2025, 7, 8)
        d2 = date(2025, 7, 1)
        table = {
            d1: {"operator image": ["2.1.44"]},
            d2: {"operator image": ["2.1.43"]},
        }
        result = latest_releases(table, self.REPOS)
        self.assertEqual(result["icr.io/instana/instana-agent-operator"], "2.1.44")

    def test_picks_latest_helm_version(self):
        d = date(2025, 7, 1)
        table = {d: {"helm chart": ["2.0.47"]}}
        result = latest_releases(table, self.REPOS)
        self.assertIn("helm:instana-agent|https://agents.instana.io/helm", result)
        self.assertEqual(result["helm:instana-agent|https://agents.instana.io/helm"], "2.0.47")

    def test_column_without_any_key_excluded(self):
        d = date(2025, 7, 1)
        table = {d: {"no-key col": ["ignored"]}}
        result = latest_releases(table, self.REPOS)
        self.assertNotIn("no-key col", result)

    def test_empty_table_returns_empty(self):
        result = latest_releases({}, self.REPOS)
        self.assertEqual(result, {})

    def test_empty_column_excluded(self):
        d = date(2025, 7, 1)
        table = {d: {"operator image": []}}
        result = latest_releases(table, self.REPOS)
        self.assertNotIn("icr.io/instana/instana-agent-operator", result)


# ---------------------------------------------------------------------------
# render_release_commands
# ---------------------------------------------------------------------------

class TestRenderReleaseCommands(unittest.TestCase):

    def test_empty_map_returns_empty_string(self):
        self.assertEqual(render_release_commands({}), "")

    def test_docker_pull_command(self):
        pull_map = {"icr.io/instana/k8sensor": "1.4.19"}
        output = render_release_commands(pull_map)
        self.assertIn("docker pull icr.io/instana/k8sensor:1.4.19", output)

    def test_helm_pull_command(self):
        pull_map = {"helm:instana-agent|https://agents.instana.io/helm": "2.0.47"}
        output = render_release_commands(pull_map)
        self.assertIn(
            "helm pull --repo https://agents.instana.io/helm instana-agent --version 2.0.47",
            output,
        )

    def test_mixed_docker_and_helm(self):
        pull_map = {
            "icr.io/instana/k8sensor": "1.4.19",
            "helm:instana-agent|https://agents.instana.io/helm": "2.0.47",
        }
        output = render_release_commands(pull_map)
        self.assertIn("docker pull icr.io/instana/k8sensor:1.4.19", output)
        self.assertIn(
            "helm pull --repo https://agents.instana.io/helm instana-agent --version 2.0.47",
            output,
        )

    def test_wrapped_in_code_block(self):
        pull_map = {"icr.io/instana/k8sensor": "1.4.19"}
        output = render_release_commands(pull_map)
        self.assertTrue(output.startswith("```"))
        self.assertIn("```\n", output)


# ---------------------------------------------------------------------------
# fill_forward
# ---------------------------------------------------------------------------

class TestFillForward(unittest.TestCase):

    COLS = ["date", "helm chart", "operator image", "agent image"]

    def test_baseline_used_for_all_rows_when_no_window_releases(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["1.0"]}}
        baseline = {"helm chart": "2.0.40", "agent image": "1.300.0"}
        result = fill_forward(table, self.COLS, baseline)
        hc_tags, hc_carried = result[d]["helm chart"]
        self.assertEqual(hc_tags, ["2.0.40"])
        self.assertTrue(hc_carried)
        ai_tags, ai_carried = result[d]["agent image"]
        self.assertEqual(ai_tags, ["1.300.0"])
        self.assertTrue(ai_carried)

    def test_actual_release_not_marked_as_carried(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["1.0"]}}
        result = fill_forward(table, self.COLS, {})
        op_tags, op_carried = result[d]["operator image"]
        self.assertEqual(op_tags, ["1.0"])
        self.assertFalse(op_carried)

    def test_carry_forward_propagates_across_rows(self):
        d1 = date(2025, 5, 1)
        d2 = date(2025, 5, 5)
        table = {
            d1: {"operator image": ["1.0"], "agent image": ["9.0"]},
            d2: {"operator image": ["1.1"]},
        }
        result = fill_forward(table, self.COLS, {})
        ai_tags, ai_carried = result[d2]["agent image"]
        self.assertEqual(ai_tags, ["9.0"])
        self.assertTrue(ai_carried)

    def test_newer_actual_release_updates_carry_forward(self):
        d1 = date(2025, 5, 1)
        d2 = date(2025, 5, 5)
        d3 = date(2025, 5, 10)
        table = {
            d1: {"operator image": ["1.0"], "agent image": ["9.0"]},
            d2: {"operator image": ["1.1"], "agent image": ["9.1"]},
            d3: {"operator image": ["1.2"]},
        }
        result = fill_forward(table, self.COLS, {})
        ai_tags, ai_carried = result[d3]["agent image"]
        self.assertEqual(ai_tags, ["9.1"])
        self.assertTrue(ai_carried)

    def test_no_baseline_and_no_prior_release_produces_empty(self):
        d = date(2025, 5, 1)
        table = {d: {"operator image": ["1.0"]}}
        result = fill_forward(table, self.COLS, {})
        hc_tags, hc_carried = result[d]["helm chart"]
        self.assertEqual(hc_tags, [])
        self.assertFalse(hc_carried)

    def test_baseline_overridden_by_actual_window_release(self):
        """When an actual release appears in the window, it takes precedence over baseline."""
        d1 = date(2025, 5, 1)
        d2 = date(2025, 5, 5)
        table = {
            d1: {"operator image": ["1.0"], "agent image": ["9.0"]},
            d2: {"operator image": ["1.1"]},
        }
        baseline = {"agent image": "8.0"}  # older than 9.0 in the window
        result = fill_forward(table, self.COLS, baseline)
        # d1 actual release beats baseline
        ai_d1_tags, ai_d1_carried = result[d1]["agent image"]
        self.assertEqual(ai_d1_tags, ["9.0"])
        self.assertFalse(ai_d1_carried)
        # d2 carry-forward should use 9.0 (the window release), not 8.0 (baseline)
        ai_d2_tags, ai_d2_carried = result[d2]["agent image"]
        self.assertEqual(ai_d2_tags, ["9.0"])
        self.assertTrue(ai_d2_carried)


# ---------------------------------------------------------------------------
# fill_forward — helm-specific behaviour
# ---------------------------------------------------------------------------

class TestFillForwardHelmBehaviour(unittest.TestCase):
    """Tests for the helm-chart no-fill-forward and helm_injected_operator logic."""

    COLS = ["date", "helm chart", "operator image", "agent image"]

    def test_helm_chart_not_carried_forward_to_operator_only_row(self):
        """Helm chart column must be empty on operator-only rows when no chart was released."""
        d1 = date(2025, 5, 1)
        d2 = date(2025, 5, 5)
        table = {
            d1: {"helm chart": ["2.0.47"], "operator image": ["2.2.14"]},
            d2: {"operator image": ["2.2.15"]},
        }
        result = fill_forward(table, self.COLS, {}, NO_FILL_FORWARD_COLS)
        hc_tags, hc_carried = result[d2]["helm chart"]
        self.assertEqual(hc_tags, [])
        self.assertFalse(hc_carried)

    def test_helm_chart_shown_only_on_release_day(self):
        """Helm chart column has a value only on the date it was released."""
        d1 = date(2025, 5, 1)
        table = {d1: {"helm chart": ["2.0.47"]}}
        result = fill_forward(table, self.COLS, {}, NO_FILL_FORWARD_COLS)
        hc_tags, hc_carried = result[d1]["helm chart"]
        self.assertEqual(hc_tags, ["2.0.47"])
        self.assertFalse(hc_carried)

    def test_helm_injected_operator_renders_italic(self):
        """Operator version injected from helm tarball is marked as carried (italic)."""
        d1 = date(2025, 5, 1)
        table = {d1: {"helm chart": ["2.0.47"], "operator image": ["2.2.14"]}}
        injected = {(d1, "2.2.14")}
        result = fill_forward(table, self.COLS, {}, NO_FILL_FORWARD_COLS, injected)
        op_tags, op_carried = result[d1]["operator image"]
        self.assertEqual(op_tags, ["2.2.14"])
        self.assertTrue(op_carried)

    def test_same_day_github_operator_not_italic(self):
        """A real GitHub operator release on the same day as a helm chart is NOT italic."""
        d1 = date(2025, 5, 1)
        table = {d1: {"helm chart": ["2.0.47"], "operator image": ["2.2.14"]}}
        # No injection: operator came from GitHub, not from the tarball
        result = fill_forward(table, self.COLS, {}, NO_FILL_FORWARD_COLS)
        op_tags, op_carried = result[d1]["operator image"]
        self.assertEqual(op_tags, ["2.2.14"])
        self.assertFalse(op_carried)

    def test_helm_injected_operator_updates_carry_forward_for_subsequent_rows(self):
        """Even an injected (italic) operator version seeds carry-forward for later rows."""
        d1 = date(2025, 5, 1)  # helm only, operator injected from tarball
        d2 = date(2025, 5, 5)  # operator only, no agent release
        table = {
            d1: {"helm chart": ["2.0.47"], "operator image": ["2.2.14"]},
            d2: {"operator image": ["2.2.15"]},
        }
        injected = {(d1, "2.2.14")}
        result = fill_forward(table, self.COLS, {}, NO_FILL_FORWARD_COLS, injected)
        # d2 has an actual operator release, so it must NOT be italic
        op_tags, op_carried = result[d2]["operator image"]
        self.assertEqual(op_tags, ["2.2.15"])
        self.assertFalse(op_carried)

    def test_scenario_helm_bundles_yesterday_op_new_op_today(self):
        """Scenario: helm chart released on day1 bundles operator from day0.
        A newer operator is released on day2 with no helm chart.

        Expected: two rows.
          day1: helm chart = '2.0.47', operator = '_2.2.14_' (italic, injected)
          day2: helm chart = '' (empty), operator = '2.2.15' (not italic)
        """
        d1 = date(2025, 5, 1)  # helm chart released, bundles operator 2.2.14
        d2 = date(2025, 5, 2)  # new operator released, no helm chart
        table = {
            d1: {"helm chart": ["2.0.47"], "operator image": ["2.2.14"]},
            d2: {"operator image": ["2.2.15"]},
        }
        injected = {(d1, "2.2.14")}
        result = fill_forward(table, self.COLS, {}, NO_FILL_FORWARD_COLS, injected)

        # day1 — helm chart present, operator italic (injected from tarball)
        hc_d1_tags, hc_d1_carried = result[d1]["helm chart"]
        op_d1_tags, op_d1_carried = result[d1]["operator image"]
        self.assertEqual(hc_d1_tags, ["2.0.47"])
        self.assertFalse(hc_d1_carried)
        self.assertEqual(op_d1_tags, ["2.2.14"])
        self.assertTrue(op_d1_carried)  # italic: injected, no GitHub release that day

        # day2 — helm chart empty (no release, no fill-forward), operator actual
        hc_d2_tags, hc_d2_carried = result[d2]["helm chart"]
        op_d2_tags, op_d2_carried = result[d2]["operator image"]
        self.assertEqual(hc_d2_tags, [])
        self.assertFalse(hc_d2_carried)
        self.assertEqual(op_d2_tags, ["2.2.15"])
        self.assertFalse(op_d2_carried)  # actual GitHub release

    def test_render_markdown_helm_no_fill_forward(self):
        """render_markdown with NO_FILL_FORWARD_COLS: helm cell is empty on operator-only rows."""
        COLS = ["date", "helm chart", "operator image"]
        d1 = date(2025, 5, 1)
        d2 = date(2025, 5, 5)
        table = {
            d1: {"helm chart": ["2.0.47"], "operator image": ["2.2.14"]},
            d2: {"operator image": ["2.2.15"]},
        }
        output = render_markdown(
            table, COLS, 6,
            datetime(2025, 6, 1, 12, 0, tzinfo=timezone.utc),
            no_fill_forward_cols={"helm chart"},
        )
        rows = [ln for ln in output.splitlines() if ln.startswith("| 2025")]
        # newest first
        d2_row = rows[0]
        d1_row = rows[1]
        # d2: helm chart cell empty
        self.assertIn("|  | 2.2.15 |", d2_row)
        # d1: helm chart actual value
        self.assertIn("| 2.0.47 |", d1_row)


# ---------------------------------------------------------------------------
# extract_operator_from_helm_chart
# ---------------------------------------------------------------------------

class TestExtractOperatorFromHelmChart(unittest.TestCase):

    _TEMPLATE_CONTENT = (
        "image: icr.io/instana/instana-agent-operator:2.2.15\n"
        "imagePullPolicy: Always\n"
    )

    def _fake_helm_pull(self, tmpdir: str, op_version: str) -> None:
        """Write a minimal fake chart structure that extract_operator_from_helm_chart expects."""
        import os
        template_dir = os.path.join(
            tmpdir,
            "instana-agent", "templates",
        )
        os.makedirs(template_dir, exist_ok=True)
        tpl_path = os.path.join(template_dir, "operator_deployment_instana-agent-controller-manager.yml")
        with open(tpl_path, "w") as fh:
            fh.write(f"image: icr.io/instana/instana-agent-operator:{op_version}\n")

    def test_extracts_operator_version(self):
        """Successfully extracts the operator version when helm pull succeeds."""
        import tempfile, os

        with tempfile.TemporaryDirectory() as outer:
            # We'll use a fake helm command that writes the expected chart structure.
            script = os.path.join(outer, "fake_helm.sh")
            chart_target = os.path.join(
                outer,
                "instana-agent", "templates",
                "operator_deployment_instana-agent-controller-manager.yml",
            )
            os.makedirs(os.path.dirname(chart_target), exist_ok=True)
            with open(chart_target, "w") as fh:
                fh.write("image: icr.io/instana/instana-agent-operator:2.2.15\n")

            # fake_helm: writes files into cwd (which will be a fresh tmpdir inside the function)
            # We need to intercept subprocess.run instead.
            def fake_run(cmd, cwd=None, **kwargs):
                # Write the chart template into cwd so the function can find it.
                if cwd:
                    tpl_dir = os.path.join(
                        cwd, "instana-agent", "templates",
                    )
                    os.makedirs(tpl_dir, exist_ok=True)
                    with open(
                        os.path.join(tpl_dir, "operator_deployment_instana-agent-controller-manager.yml"),
                        "w",
                    ) as fh:
                        fh.write("image: icr.io/instana/instana-agent-operator:2.2.15\n")
                r = MagicMock()
                r.returncode = 0
                r.stderr = ""
                return r

            with patch("subprocess.run", side_effect=fake_run):
                with patch("shutil.which", return_value="/usr/bin/helm"):
                    result = extract_operator_from_helm_chart(
                        "instana-agent", "https://agents.instana.io/helm", "2.0.47"
                    )
            self.assertEqual(result, "2.2.15")

    def test_returns_none_when_helm_not_found(self):
        with patch("shutil.which", return_value=None):
            result = extract_operator_from_helm_chart(
                "instana-agent", "https://agents.instana.io/helm", "2.0.47"
            )
        self.assertIsNone(result)

    def test_returns_none_on_helm_pull_failure(self):
        def fake_run(cmd, cwd=None, **kwargs):
            r = MagicMock()
            r.returncode = 1
            r.stderr = "Error: chart not found"
            return r

        with patch("subprocess.run", side_effect=fake_run):
            with patch("shutil.which", return_value="/usr/bin/helm"):
                result = extract_operator_from_helm_chart(
                    "instana-agent", "https://agents.instana.io/helm", "9.9.9",
                )
        self.assertIsNone(result)

    def test_returns_none_when_template_missing(self):
        def fake_run(cmd, cwd=None, **kwargs):
            r = MagicMock()
            r.returncode = 0
            r.stderr = ""
            return r  # does NOT write any files

        with patch("subprocess.run", side_effect=fake_run):
            with patch("shutil.which", return_value="/usr/bin/helm"):
                result = extract_operator_from_helm_chart(
                    "instana-agent", "https://agents.instana.io/helm", "2.0.47",
                )
        self.assertIsNone(result)

    def test_returns_none_on_timeout(self):
        import subprocess as _sp
        with patch("subprocess.run", side_effect=_sp.TimeoutExpired(cmd="helm", timeout=60)):
            with patch("shutil.which", return_value="/usr/bin/helm"):
                result = extract_operator_from_helm_chart(
                    "instana-agent", "https://agents.instana.io/helm", "2.0.47",
                )
        self.assertIsNone(result)

    def test_helm_cmd_override_used_in_tests(self):
        """_helm_cmd lets callers substitute an alternative helm binary."""
        def fake_run(cmd, cwd=None, **kwargs):
            self.assertTrue(cmd[0].endswith("fake-helm"))
            if cwd:
                import os
                tpl_dir = os.path.join(cwd, "instana-agent", "templates")
                os.makedirs(tpl_dir, exist_ok=True)
                with open(
                    os.path.join(tpl_dir, "operator_deployment_instana-agent-controller-manager.yml"),
                    "w",
                ) as fh:
                    fh.write("image: icr.io/instana/instana-agent-operator:3.0.0\n")
            r = MagicMock()
            r.returncode = 0
            r.stderr = ""
            return r

        with patch("subprocess.run", side_effect=fake_run):
            result = extract_operator_from_helm_chart(
                "instana-agent", "https://agents.instana.io/helm", "3.0.0",
                _helm_cmd=["fake-helm"],
            )
        self.assertEqual(result, "3.0.0")


# ---------------------------------------------------------------------------
# fetch_helm_operator_pins
# ---------------------------------------------------------------------------

class TestFetchHelmOperatorPins(unittest.TestCase):

    def test_returns_mapping_for_all_releases(self):
        helm_releases = [
            (date(2025, 5, 1), "2.0.47"),
            (date(2025, 5, 10), "2.0.48"),
        ]
        # Fake extract: version maps simply chart → op
        call_versions = []

        def fake_extract(chart, repo_url, version, _helm_cmd=None):
            call_versions.append(version)
            return f"op-{version}"

        with patch.object(_mod, "extract_operator_from_helm_chart", side_effect=fake_extract):
            result = fetch_helm_operator_pins(
                helm_releases, "instana-agent", "https://agents.instana.io/helm"
            )

        self.assertEqual(result, {"2.0.47": "op-2.0.47", "2.0.48": "op-2.0.48"})
        self.assertEqual(call_versions, ["2.0.47", "2.0.48"])

    def test_omits_failed_extractions(self):
        helm_releases = [(date(2025, 5, 1), "2.0.47")]

        with patch.object(_mod, "extract_operator_from_helm_chart", return_value=None):
            result = fetch_helm_operator_pins(
                helm_releases, "instana-agent", "https://agents.instana.io/helm"
            )

        self.assertEqual(result, {})

    def test_empty_releases_returns_empty(self):
        result = fetch_helm_operator_pins(
            [], "instana-agent", "https://agents.instana.io/helm"
        )
        self.assertEqual(result, {})


if __name__ == "__main__":
    unittest.main()
