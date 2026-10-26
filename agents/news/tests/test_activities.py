# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""`list_github_changes` against fixture files written per test: pagination, truncation and
noise filtering."""

import json
from pathlib import Path
from typing import Any

import httpx
import pytest

from credimi_news_agent import activities, http
from credimi_news_agent.activities import GITHUB_API, list_github_changes
from credimi_news_agent.models import ListChangesInput

REPO = "acme/spec"
INPUT = ListChangesInput(repo=REPO, since="2026-10-01", until="2026-10-14")
COMMITS_PARAMS = {"since": "2026-10-01T00:00:00Z", "until": "2026-10-15T00:00:00Z"}


@pytest.fixture
def github(tmp_path, monkeypatch):
    monkeypatch.setenv("NEWS_AGENT_FIXTURES_DIR", str(tmp_path))

    def write(path: str, body: Any, **params: str | int) -> None:
        url = httpx.URL(f"{GITHUB_API}{path}", params=params)
        (Path(tmp_path) / http.fixture_name(url)).write_text(
            json.dumps(
                {
                    "url": str(url),
                    "status": 200,
                    "content_type": "application/json",
                    "body": json.dumps(body),
                }
            )
        )

    return write


def pull(number: int, day: int) -> dict[str, Any]:
    stamp = f"2026-10-{day:02d}T12:00:00Z"
    return {
        "number": number,
        "title": f"Change {number}",
        "body": "",
        "html_url": f"https://github.com/{REPO}/pull/{number}",
        "merge_commit_sha": f"merge{number}",
        "merged_at": stamp,
        "updated_at": stamp,
        "state": "closed",
    }


def commit(sha: str, message: str) -> dict[str, Any]:
    return {
        "sha": sha,
        "html_url": f"https://github.com/{REPO}/commit/{sha}",
        "commit": {"message": message, "committer": {"date": "2026-10-05T10:00:00Z"}},
    }


PULLS_PARAMS = {"state": "closed", "sort": "updated", "direction": "desc"}


def write_pulls(github, per_page: int, *pages: list[dict[str, Any]]) -> None:
    for n, page in enumerate(pages, start=1):
        extra = {"page": n} if n > 1 else {}
        github(f"/repos/{REPO}/pulls", page, per_page=per_page, **PULLS_PARAMS, **extra)


def write_rest(github, commits: list[dict[str, Any]] | None = None) -> None:
    github(f"/repos/{REPO}/releases", [], per_page=activities.RELEASES_PER_PAGE)
    github(
        f"/repos/{REPO}/commits",
        commits or [],
        per_page=activities.COMMITS_PER_PAGE,
        **COMMITS_PARAMS,
    )


async def test_pull_requests_are_read_across_pages(github, monkeypatch):
    monkeypatch.setattr(activities, "PULLS_PER_PAGE", 2)
    write_pulls(github, 2, [pull(5, 14), pull(4, 12)], [pull(3, 10), pull(2, 8)], [pull(1, 6)])
    write_rest(github)

    changes = await list_github_changes(INPUT)

    assert [i.ref for i in changes.items if i.kind == "pull_request"] == ["5", "4", "3", "2", "1"]
    assert not changes.truncated


async def test_a_full_last_page_inside_the_window_marks_the_list_truncated(github, monkeypatch):
    monkeypatch.setattr(activities, "PULLS_PER_PAGE", 2)
    monkeypatch.setattr(activities, "MAX_PAGES", 1)
    write_pulls(github, 2, [pull(5, 14), pull(4, 12)])
    write_rest(github)

    changes = await list_github_changes(INPUT)

    assert [i.ref for i in changes.items] == ["5", "4"]
    assert changes.truncated


NOISY_COMMITS = [
    commit("aaa111", "Merge branch 'main' into x"),
    commit("bbb222", "Fix wording\n\nfrom pull request 7"),
    commit("ccc333", "Clarify the response mode"),
]


async def test_merge_commits_and_pull_request_commits_are_dropped(github):
    write_pulls(github, activities.PULLS_PER_PAGE, [pull(7, 9)])
    write_rest(github, NOISY_COMMITS)
    github(f"/repos/{REPO}/pulls/7/commits", [{"sha": "bbb222"}], per_page=100)

    changes = await list_github_changes(INPUT)

    assert [i.ref for i in changes.items if i.kind == "commit"] == ["ccc333"]


async def test_unreadable_pull_request_commits_keep_the_commits(github):
    write_pulls(github, activities.PULLS_PER_PAGE, [pull(7, 9)])
    write_rest(github, NOISY_COMMITS)

    changes = await list_github_changes(INPUT)

    assert [i.ref for i in changes.items if i.kind == "commit"] == ["bbb222", "ccc333"]
