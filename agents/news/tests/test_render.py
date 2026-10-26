# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

from credimi_news_agent.models import Evidence, KeyPoint, NewsDraft
from credimi_news_agent.render import render_payload


def evidence(ev_id: str, url: str, title: str) -> Evidence:
    return Evidence(
        id=ev_id,
        url=url,
        kind="pull_request",
        title=title,
        retrieved_at="2026-10-14T00:00:00+00:00",
        sha256="ab" * 32,
        text="text",
    )


LEDGER = {
    "ev_a": evidence("ev_a", "https://github.com/o/r/pull/1", "PR <one>"),
    "ev_b": evidence("ev_b", "https://github.com/o/r/pull/2", "PR two"),
    "ev_unused": evidence("ev_unused", "https://github.com/o/r/pull/3", "PR three"),
}


def make_draft() -> NewsDraft:
    return NewsDraft(
        title="Title",
        summary='Summary <img src=x onerror="alert(1)">',
        body="Body <script>alert(1)</script> **bold**",
        key_points=[
            KeyPoint(text="second cited first", evidence_ids=["ev_b"]),
            KeyPoint(text="both", evidence_ids=["ev_a", "ev_b"]),
        ],
        quotes=[],
        tags=["a"],
        refer="https://github.com/o/r",
        diff="https://github.com/o/r/pull/2",
        rationale="r",
    )


def test_model_html_is_escaped_not_passed_through():
    payload = render_payload(make_draft(), LEDGER, "news-scan-x")
    assert "<script>" not in payload["news_html"]
    assert "<img" not in payload["summary_html"]
    assert "&lt;script&gt;" in payload["news_html"]
    assert "<strong>bold</strong>" in payload["news_html"]


def test_sources_list_only_cited_evidence_numbered_in_citation_order():
    payload = render_payload(make_draft(), LEDGER, "news-scan-x")
    html = payload["news_html"]
    assert html.index("pull/2") < html.index("pull/1")
    assert "pull/3" not in html
    assert "PR &lt;one&gt;" in html
    assert payload["key_differences_md"].splitlines() == [
        "- second cited first [1]",
        "- both [2] [1]",
    ]
