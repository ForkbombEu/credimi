# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Deterministic checks a draft must pass before it can be stored.

They run inside the workflow (pure functions, no I/O), so a draft is accepted only when every
statement points at evidence an activity actually retrieved during this scan.
"""

from __future__ import annotations

import re
from collections.abc import Collection, Iterable, Mapping, Sequence
from datetime import date

from .activities import host_allowed
from .models import Evidence, ItemDecision, NewsDraft

MAX_TITLE = 120
MAX_TAGS = 6
MAX_WINDOW_DAYS = 92


def window_error(since: str, until: str, today: date, bounds: tuple[str, str] | None) -> str | None:
    """Why [since, until] cannot be listed, or None. ``bounds`` pins a scan to its window."""
    try:
        start, end = date.fromisoformat(since), date.fromisoformat(until)
    except ValueError:
        return "dates must be YYYY-MM-DD"
    if start > end:
        return "since is after until"
    if end > today:
        return f"until is in the future (today is {today.isoformat()})"
    if (end - start).days + 1 > MAX_WINDOW_DAYS:
        return f"the window is longer than {MAX_WINDOW_DAYS} days; split it"
    if bounds is not None and (
        start < date.fromisoformat(bounds[0]) or end > date.fromisoformat(bounds[1])
    ):
        return f"stay inside the scan window {bounds[0]} to {bounds[1]}"
    return None


def _normalize(text: str) -> str:
    return re.sub(r"\s+", " ", text).strip().lower()


def validate_draft(
    draft: NewsDraft,
    ledger: Mapping[str, Evidence],
    allowed_domains: list[str],
    accepted_diffs: Iterable[str],
) -> list[str]:
    errors: list[str] = []

    if not draft.title.strip():
        errors.append("title is empty")
    elif len(draft.title) > MAX_TITLE:
        errors.append(f"title is longer than {MAX_TITLE} characters")
    if not draft.summary.strip():
        errors.append("summary is empty")
    if not draft.body.strip():
        errors.append("body is empty")
    if not 1 <= len(draft.tags) <= MAX_TAGS:
        errors.append(f"give between 1 and {MAX_TAGS} tags")

    if not draft.key_points:
        errors.append("at least one key point is required")
    for i, point in enumerate(draft.key_points, start=1):
        if not point.evidence_ids:
            errors.append(f"key point {i} cites no evidence")
        for ev_id in point.evidence_ids:
            if ev_id not in ledger:
                errors.append(f"key point {i} cites {ev_id}, which no tool returned in this scan")
        if (
            point.evidence_ids
            and all(ev_id in ledger for ev_id in point.evidence_ids)
            and not any(ledger[ev_id].opened for ev_id in point.evidence_ids)
        ):
            errors.append(
                f"key point {i} cites only listed items; open the change first "
                "(get_pull_request, compare_refs or fetch_document)"
            )

    for i, quote in enumerate(draft.quotes, start=1):
        evidence = ledger.get(quote.evidence_id)
        if evidence is None:
            errors.append(f"quote {i} cites {quote.evidence_id}, which no tool returned")
        elif _normalize(quote.text) not in _normalize(evidence.text):
            errors.append(f"quote {i} is not an exact excerpt of {quote.evidence_id}")

    by_url = {ev.url: ev for ev in ledger.values()}
    if draft.diff not in by_url:
        errors.append("diff must be the URL of evidence retrieved in this scan")
    elif not by_url[draft.diff].opened:
        errors.append(
            f"diff {draft.diff} was only listed; open it with get_pull_request, compare_refs "
            "or fetch_document"
        )
    if not host_allowed(draft.refer, allowed_domains):
        errors.append("refer must be an https URL on an allowed domain")
    if draft.diff in set(accepted_diffs):
        errors.append("a draft for this diff was already accepted in this scan")

    return errors


def check_scan_report(
    decisions: Sequence[ItemDecision],
    ledger: Mapping[str, Evidence],
    accepted_diffs: Collection[str],
    listed: bool,
) -> list[str]:
    """Where the agent's decisions disagree with what the scan retrieved and accepted."""
    problems: list[str] = []
    if not listed:
        problems.append("invalid scan: no list_changes call succeeded")
    urls = {ev.url for ev in ledger.values()}
    drafted = set(accepted_diffs) | {ev.id for ev in ledger.values() if ev.url in accepted_diffs}
    for i, decision in enumerate(decisions, start=1):
        if decision.ref not in ledger and decision.ref not in urls:
            problems.append(f"decision {i} ({decision.ref}) is not an item retrieved in this scan")
    for i, decision in enumerate(decisions, start=1):
        if decision.decision == "draft" and decision.ref not in drafted:
            problems.append(
                f"decision {i} ({decision.ref}) says draft but no accepted draft covers it"
            )
    return problems
