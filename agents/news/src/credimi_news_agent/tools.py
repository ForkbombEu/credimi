# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""The agent's tools. They run inline in the workflow and delegate all I/O to activities.

Running inline lets each tool record what it retrieved in the session's evidence ledger
(``ResearchSession``), which the harness injects per call and hides from the model. Reads go to
the Python activities on this worker; `search_news` and storage go to Go activities on
``NewsToolsTaskQueue``. No tool writes to Credimi: accepted drafts are stored by the workflow
after the agent's turn.

Scans draft with `submit_draft`. Chats check with `check_draft`, then `save_draft`, which is not
inherently safe: the approval policy pauses it until a human approves it in the harness UI.

No ``from __future__ import annotations``: the harness reads tool annotations concretely to
build the model's schema and to find ``Injected`` parameters.
"""

import json
from dataclasses import dataclass, field
from datetime import date, timedelta
from typing import Any

from temporalio import workflow
from temporalio.common import RetryPolicy
from temporalio.exceptions import ActivityError

with workflow.unsafe.imports_passed_through():
    from temporal_agent_harness.harness import agent

    from . import activities
    from .activities import host_allowed
    from .models import (
        GO_SEARCH_NEWS,
        GO_TASK_QUEUE,
        ChangeList,
        CompareInput,
        Evidence,
        FetchInput,
        ListChangesInput,
        NewsDraft,
        PullRequestInput,
        Source,
        SourceConfig,
    )
    from .validation import validate_draft, window_error

IO_TIMEOUT = timedelta(seconds=60)
GO_TIMEOUT = timedelta(seconds=30)
RETRY = RetryPolicy(initial_interval=timedelta(seconds=2), maximum_attempts=3)
EXCERPT = 4_000


@dataclass
class ResearchSession:
    """Workflow-owned state of one scan or one chat; rebuilt deterministically on replay.

    ``window`` pins a scan to its [since, until]; chats have none and may ask for any window
    up to today (``window_error``).
    """

    config: SourceConfig
    today: date
    max_drafts: int
    source_ids: list[str] = field(default_factory=list)
    window: tuple[str, str] | None = None
    ledger: dict[str, Evidence] = field(default_factory=dict)
    accepted: list[NewsDraft] = field(default_factory=list)
    stored: int = 0
    rejected: int = 0

    def sources(self) -> list[Source]:
        wanted = set(self.source_ids)
        return [s for s in self.config.sources if not wanted or s.id in wanted]

    def source(self, source_id: str) -> Source | None:
        return next((s for s in self.sources() if s.id == source_id), None)

    def record(self, evidence: list[Evidence]) -> None:
        for ev in evidence:
            current = self.ledger.get(ev.id)
            # Keep the richest retrieval of a URL (e.g. full PR over its list entry).
            if current is None or len(ev.text) >= len(current.text):
                self.ledger[ev.id] = ev


def _evidence_view(ev: Evidence) -> str:
    text = ev.text if len(ev.text) <= EXCERPT else ev.text[:EXCERPT] + "\n[... truncated]"
    return json.dumps(
        {"evidence_id": ev.id, "url": ev.url, "kind": ev.kind, "title": ev.title, "text": text}
    )


def _failure(exc: ActivityError) -> str:
    cause = exc.cause
    return f"ERROR: {cause.message if cause is not None and hasattr(cause, 'message') else exc}"


async def go_search_news(payload: dict[str, Any]) -> list[dict[str, Any]]:
    result = await workflow.execute_activity(
        GO_SEARCH_NEWS,
        {"payload": payload},
        task_queue=GO_TASK_QUEUE,
        start_to_close_timeout=GO_TIMEOUT,
        retry_policy=RETRY,
        result_type=dict,
    )
    return (result.get("output") or {}).get("items") or []


@agent.tool_defn(inherently_safe=True)
async def list_changes(
    source_id: str, since: str, until: str, session: agent.Injected[ResearchSession]
) -> str:
    """List releases, merged pull requests and commits of one source between two dates.

    Args:
        source_id: Id of a source, e.g. "openid4vp".
        since: First day, YYYY-MM-DD (UTC).
        until: Last day, YYYY-MM-DD (UTC), inclusive.
    """
    source = session.source(source_id)
    if source is None:
        return (
            f"ERROR: unknown source {source_id!r}; use one of {[s.id for s in session.sources()]}"
        )
    if problem := window_error(since, until, session.today, session.window):
        return f"ERROR: {problem}"
    try:
        changes: ChangeList = await workflow.execute_activity(
            activities.list_github_changes,
            ListChangesInput(repo=source.repo, since=since, until=until),
            start_to_close_timeout=IO_TIMEOUT,
            retry_policy=RETRY,
        )
    except ActivityError as exc:
        return _failure(exc)
    session.record(changes.evidence)
    return json.dumps(
        {"repo": changes.repo, "items": [i.model_dump() for i in changes.items]}
        if changes.items
        else {"repo": changes.repo, "items": [], "note": "no changes in the window"}
    )


@agent.tool_defn(inherently_safe=True)
async def get_pull_request(repo: str, number: int, session: agent.Injected[ResearchSession]) -> str:
    """Read one pull request: description, state and the patch of every changed file.

    Args:
        repo: "owner/name" of the repository.
        number: Pull request number.
    """
    try:
        ev: Evidence = await workflow.execute_activity(
            activities.get_pull_request,
            PullRequestInput(repo=repo, number=number),
            start_to_close_timeout=IO_TIMEOUT,
            retry_policy=RETRY,
        )
    except ActivityError as exc:
        return _failure(exc)
    session.record([ev])
    return _evidence_view(ev)


@agent.tool_defn(inherently_safe=True)
async def compare_refs(
    repo: str, base: str, head: str, session: agent.Injected[ResearchSession]
) -> str:
    """Compare two refs (tags, branches or commit SHAs) of a repository: commits and patches.

    Args:
        repo: "owner/name" of the repository.
        base: Older ref, e.g. a previous release tag.
        head: Newer ref.
    """
    try:
        ev: Evidence = await workflow.execute_activity(
            activities.compare_refs,
            CompareInput(repo=repo, base=base, head=head),
            start_to_close_timeout=IO_TIMEOUT,
            retry_policy=RETRY,
        )
    except ActivityError as exc:
        return _failure(exc)
    session.record([ev])
    return _evidence_view(ev)


@agent.tool_defn(inherently_safe=True)
async def fetch_document(url: str, session: agent.Injected[ResearchSession]) -> str:
    """Fetch a web page (specification, changelog, announcement) as plain text.

    Only https URLs on the allowed domains of the scan can be fetched.

    Args:
        url: Absolute https URL.
    """
    domains = session.config.allowed_domains
    if not host_allowed(url, domains):
        return f"ERROR: {url} is not on the allowed domains {domains}"
    try:
        ev: Evidence = await workflow.execute_activity(
            activities.fetch_document,
            FetchInput(url=url, allowed_domains=domains),
            start_to_close_timeout=IO_TIMEOUT,
            retry_policy=RETRY,
        )
    except ActivityError as exc:
        return _failure(exc)
    session.record([ev])
    return _evidence_view(ev)


@agent.tool_defn(inherently_safe=True)
async def search_news(query: str) -> str:
    """Search Credimi news, including unpublished drafts, to find stories already covered.

    Args:
        query: Words to look for in titles and texts, e.g. "OpenID4VP draft 24".
    """
    try:
        items = await go_search_news({"query": query, "limit": 10})
    except ActivityError as exc:
        return _failure(exc)
    return json.dumps({"items": items} if items else {"items": [], "note": "no matching news"})


async def _draft_errors(draft: NewsDraft, session: ResearchSession) -> list[str]:
    """Every reason the draft cannot be accepted (empty list: acceptable)."""
    if len(session.accepted) >= session.max_drafts:
        return [f"the limit of {session.max_drafts} drafts per session is reached"]
    errors = validate_draft(
        draft,
        session.ledger,
        session.config.allowed_domains,
        [d.diff for d in session.accepted],
    )
    if not errors:
        existing = await go_search_news({"diff_urls": [draft.diff], "limit": 1})
        if existing:
            errors.append(
                f"news record {existing[0]['id']} ({existing[0]['title']!r}) already covers "
                "this diff; record the item as a duplicate"
            )
    return errors


async def _accept(draft: NewsDraft, session: ResearchSession) -> str:
    try:
        errors = await _draft_errors(draft, session)
    except ActivityError as exc:
        return _failure(exc)
    if errors:
        session.rejected += 1
        return "REJECTED:\n" + "\n".join(f"- {e}" for e in errors)
    session.accepted.append(draft)
    return (
        f"ACCEPTED as draft {len(session.accepted)} of at most {session.max_drafts}. "
        "It will be stored unpublished for human review."
    )


@agent.tool_defn(inherently_safe=True)
async def submit_draft(draft: NewsDraft, session: agent.Injected[ResearchSession]) -> str:
    """Submit a news draft for storage. It is checked before acceptance; read the reply.

    Accepted drafts are stored unpublished for human review after the scan; nothing is
    published. A rejection lists every problem: fix them and submit again, or drop the story.
    """
    return await _accept(draft, session)


@agent.tool_defn(inherently_safe=True)
async def check_draft(draft: NewsDraft, session: agent.Injected[ResearchSession]) -> str:
    """Check a news draft without saving it. Call this before `save_draft`.

    Replies OK, or lists every problem to fix (citations, quotes, diff, duplicates, limits).
    """
    try:
        errors = await _draft_errors(draft, session)
    except ActivityError as exc:
        return _failure(exc)
    if errors:
        return "PROBLEMS:\n" + "\n".join(f"- {e}" for e in errors)
    return "OK: call save_draft with exactly this draft; a human must approve it."


@agent.tool_defn()
async def save_draft(draft: NewsDraft, session: agent.Injected[ResearchSession]) -> str:
    """Save a checked news draft, unpublished, for human review. Needs human approval.

    Only call it after `check_draft` replied OK and when the user asked for a draft.
    """
    return await _accept(draft, session)


_READ_TOOLS = [list_changes, get_pull_request, compare_refs, fetch_document, search_news]
SCAN_TOOLS = [*_READ_TOOLS, submit_draft]
CHAT_TOOLS = [*_READ_TOOLS, check_draft, save_draft]
