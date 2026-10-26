# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Turn an accepted draft into the `Store a news draft` payload.

The model writes Markdown; raw HTML in it is escaped, never passed through. The Sources list
is built from the evidence ledger, not from model text, so every link is a URL that was
actually retrieved.
"""

from __future__ import annotations

from collections.abc import Mapping
from html import escape

from markdown_it import MarkdownIt

from .models import Evidence, NewsDraft

_md = MarkdownIt("commonmark", {"html": False})


def cited_ids(draft: NewsDraft) -> list[str]:
    ordered: list[str] = []
    for ev_id in [i for p in draft.key_points for i in p.evidence_ids] + [
        q.evidence_id for q in draft.quotes
    ]:
        if ev_id not in ordered:
            ordered.append(ev_id)
    return ordered


def render_payload(
    draft: NewsDraft, ledger: Mapping[str, Evidence], workflow_id: str
) -> dict[str, object]:
    ids = cited_ids(draft)
    number = {ev_id: n for n, ev_id in enumerate(ids, start=1)}

    sources = "".join(
        f'<li>[{number[ev_id]}] <a href="{escape(ledger[ev_id].url, quote=True)}">'
        f"{escape(ledger[ev_id].title)}</a> (retrieved {escape(ledger[ev_id].retrieved_at)}, "
        f"sha256 {ledger[ev_id].sha256[:12]})</li>"
        for ev_id in ids
    )
    news_html = (
        _md.render(draft.body)
        + f"<h4>Sources</h4><ol>{sources}</ol>"
        + f"<p><small>Drafted by the Credimi news agent, run {escape(workflow_id)}. "
        + "Not reviewed yet.</small></p>"
    )
    key_differences = "\n".join(
        f"- {p.text} " + " ".join(f"[{number[i]}]" for i in p.evidence_ids if i in number)
        for p in draft.key_points
    )
    return {
        "title": draft.title.strip(),
        "summary_html": _md.render(draft.summary),
        "news_html": news_html,
        "key_differences_md": key_differences,
        "tags": draft.tags,
        "refer": draft.refer,
        "diff": draft.diff,
    }
