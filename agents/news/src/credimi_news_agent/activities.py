# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Python I/O activities: source registry, GitHub REST, and allowlisted document fetches.

Each retrieval returns ``Evidence``: the URL, a sha256 of the retrieved text, and the text
itself (capped). The workflow keeps a ledger of these, and drafts may cite only ledger ids.

No ``from __future__ import annotations``: activity argument and return types cross Temporal's
data converter, which resolves them from these annotations.
"""

import hashlib
import os
import re
import tomllib
from datetime import UTC, date, datetime, timedelta
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import urlparse

import httpx
from temporalio import activity
from temporalio.exceptions import ApplicationError

from . import http
from .models import (
    ChangeItem,
    ChangeList,
    CompareInput,
    Evidence,
    FetchInput,
    ListChangesInput,
    PullRequestInput,
    SourceConfig,
)

GITHUB_API = "https://api.github.com"
TEXT_CAP = 60_000
ITEM_TEXT_CAP = 6_000
PATCH_CAP = 4_000
MAX_ITEMS = 40


def evidence_id(url: str) -> str:
    return "ev_" + hashlib.sha256(url.encode()).hexdigest()[:12]


def make_evidence(url: str, kind: str, title: str, text: str) -> Evidence:
    text = text[:TEXT_CAP]
    return Evidence(
        id=evidence_id(url),
        url=url,
        kind=kind,  # type: ignore[arg-type]
        title=title,
        retrieved_at=datetime.now(UTC).isoformat(timespec="seconds"),
        sha256=hashlib.sha256(text.encode()).hexdigest(),
        text=text,
    )


def host_allowed(url: str, allowed_domains: list[str]) -> bool:
    parsed = urlparse(url)
    host = (parsed.hostname or "").lower()
    if parsed.scheme != "https" or not host:
        return False
    return any(host == d or host.endswith("." + d) for d in allowed_domains)


def _window(since: str, until: str) -> tuple[datetime, datetime]:
    start = datetime.combine(date.fromisoformat(since), datetime.min.time(), UTC)
    end = datetime.combine(date.fromisoformat(until), datetime.min.time(), UTC) + timedelta(days=1)
    return start, end


def _parse_ts(value: str | None) -> datetime | None:
    if not value:
        return None
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def _raise_for_status(response: httpx.Response) -> None:
    if response.status_code < 400:
        return
    message = f"HTTP {response.status_code} for {response.request.url}: {response.text[:300]}"
    if response.status_code in (403, 429):
        message += " (GitHub rate limit? set GITHUB_TOKEN)"
    # Server errors may recover; client errors (incl. rate limits) are not worth retrying here.
    raise ApplicationError(message, non_retryable=response.status_code < 500)


async def _github_get(client: httpx.AsyncClient, path: str, **params: str | int) -> object:
    response = await client.get(f"{GITHUB_API}{path}", params=params, headers=http.github_headers())
    _raise_for_status(response)
    return response.json()


@activity.defn(name="news.load_sources")
async def load_sources() -> SourceConfig:
    default = Path(__file__).resolve().parents[2] / "sources.toml"
    path = Path(os.environ.get("NEWS_AGENT_SOURCES", default))
    return SourceConfig.model_validate(tomllib.loads(path.read_text()))


@activity.defn(name="news.list_github_changes")
async def list_github_changes(inp: ListChangesInput) -> ChangeList:
    start, end = _window(inp.since, inp.until)
    items: list[tuple[datetime, ChangeItem, Evidence]] = []

    def add(kind: str, when: datetime, url: str, title: str, ref: str, text: str) -> None:
        ev = make_evidence(url, kind, title, text[:ITEM_TEXT_CAP])
        item = ChangeItem(
            evidence_id=ev.id,
            kind=kind,  # type: ignore[arg-type]
            title=title,
            url=url,
            date=when.date().isoformat(),
            ref=ref,
        )
        items.append((when, item, ev))

    async with http.client() as client:
        releases = await _github_get(client, f"/repos/{inp.repo}/releases", per_page=20)
        for rel in releases:  # type: ignore[union-attr]
            when = _parse_ts(rel.get("published_at"))
            if when and start <= when < end:
                title = f"Release {rel['tag_name']}: {rel.get('name') or ''}".strip(": ")
                add("release", when, rel["html_url"], title, rel["tag_name"], rel.get("body") or "")

        pulls = await _github_get(
            client,
            f"/repos/{inp.repo}/pulls",
            state="closed",
            sort="updated",
            direction="desc",
            per_page=50,
        )
        merged_shas: set[str] = set()
        for pr in pulls:  # type: ignore[union-attr]
            when = _parse_ts(pr.get("merged_at"))
            if when and start <= when < end:
                merged_shas.add(pr.get("merge_commit_sha") or "")
                add(
                    "pull_request",
                    when,
                    pr["html_url"],
                    f"PR #{pr['number']}: {pr['title']}",
                    str(pr["number"]),
                    f"{pr['title']}\n\n{pr.get('body') or ''}",
                )

        commits = await _github_get(
            client,
            f"/repos/{inp.repo}/commits",
            since=start.isoformat().replace("+00:00", "Z"),
            until=end.isoformat().replace("+00:00", "Z"),
            per_page=30,
        )
        for c in commits:  # type: ignore[union-attr]
            message = c["commit"]["message"]
            if c["sha"] in merged_shas or message.startswith("Merge pull request"):
                continue
            when = _parse_ts(c["commit"]["committer"]["date"]) or start
            add(
                "commit", when, c["html_url"], message.splitlines()[0][:200], c["sha"][:12], message
            )

    items.sort(key=lambda entry: entry[0], reverse=True)
    items = items[:MAX_ITEMS]
    return ChangeList(
        repo=inp.repo,
        items=[item for _, item, _ in items],
        evidence=[ev for _, _, ev in items],
    )


@activity.defn(name="news.get_pull_request")
async def get_pull_request(inp: PullRequestInput) -> Evidence:
    async with http.client() as client:
        pr = await _github_get(client, f"/repos/{inp.repo}/pulls/{inp.number}")
        files = await _github_get(
            client, f"/repos/{inp.repo}/pulls/{inp.number}/files", per_page=100
        )
    lines = [
        f"PR #{pr['number']}: {pr['title']}",  # type: ignore[index]
        f"State: {pr['state']}, merged at: {pr.get('merged_at')}",  # type: ignore[index,union-attr]
        f"Author: {pr['user']['login']}",  # type: ignore[index]
        "",
        pr.get("body") or "",  # type: ignore[union-attr]
        "",
        "Files changed:",
    ]
    for f in files:  # type: ignore[union-attr]
        lines.append(f"--- {f['filename']} ({f['status']}, +{f['additions']} -{f['deletions']})")
        if patch := f.get("patch"):
            lines.append(patch[:PATCH_CAP])
    return make_evidence(
        pr["html_url"],  # type: ignore[index]
        "pull_request",
        f"PR #{pr['number']}: {pr['title']}",  # type: ignore[index]
        "\n".join(lines),
    )


@activity.defn(name="news.compare_refs")
async def compare_refs(inp: CompareInput) -> Evidence:
    async with http.client() as client:
        cmp = await _github_get(client, f"/repos/{inp.repo}/compare/{inp.base}...{inp.head}")
    lines = [f"Compare {inp.base}...{inp.head}: {cmp['status']}, {cmp['total_commits']} commits"]  # type: ignore[index]
    for c in cmp.get("commits", []):  # type: ignore[union-attr]
        lines.append(f"* {c['commit']['message'].splitlines()[0]}")
    lines.append("")
    for f in cmp.get("files", []):  # type: ignore[union-attr]
        lines.append(f"--- {f['filename']} ({f['status']}, +{f['additions']} -{f['deletions']})")
        if patch := f.get("patch"):
            lines.append(patch[:PATCH_CAP])
    return make_evidence(
        cmp["html_url"],  # type: ignore[index]
        "compare",
        f"{inp.repo} {inp.base}...{inp.head}",
        "\n".join(lines),
    )


class _TextExtractor(HTMLParser):
    _SKIP = {"script", "style", "noscript", "svg", "head"}

    def __init__(self) -> None:
        super().__init__()
        self.parts: list[str] = []
        self.title = ""
        self._skip = 0
        self._in_title = False

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag in self._SKIP:
            self._skip += 1
        if tag == "title":
            self._in_title = True

    def handle_endtag(self, tag: str) -> None:
        if tag in self._SKIP and self._skip:
            self._skip -= 1
        if tag == "title":
            self._in_title = False

    def handle_data(self, data: str) -> None:
        if self._in_title:
            self.title += data
        elif not self._skip:
            self.parts.append(data)


@activity.defn(name="news.fetch_document")
async def fetch_document(inp: FetchInput) -> Evidence:
    if not host_allowed(inp.url, inp.allowed_domains):
        raise ApplicationError(f"{inp.url} is not on the allowed domain list", non_retryable=True)
    async with http.client() as client:
        response = await client.get(inp.url)
    if response.is_redirect:
        raise ApplicationError(
            f"{inp.url} redirects to {response.headers.get('location')}; fetch that URL instead",
            non_retryable=True,
        )
    _raise_for_status(response)
    content_type = response.headers.get("content-type", "")
    if "html" in content_type:
        parser = _TextExtractor()
        parser.feed(response.text)
        text = re.sub(r"\n\s*\n+", "\n\n", "".join(parser.parts))
        title = parser.title.strip() or inp.url
    else:
        text, title = response.text, inp.url
    return make_evidence(inp.url, "document", title, text.strip())


ALL_ACTIVITIES = [load_sources, list_github_changes, get_pull_request, compare_refs, fetch_document]
