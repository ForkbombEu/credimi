# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

from datetime import date

import pytest

from credimi_news_agent.models import Evidence, KeyPoint, NewsDraft, Quote
from credimi_news_agent.validation import validate_draft, window_error

TODAY = date(2026, 10, 14)


@pytest.mark.parametrize(
    ("since", "until", "bounds", "expected"),
    [
        ("2026-10-01", "2026-10-14", None, None),
        ("2026-07-15", "2026-10-14", None, None),  # exactly 92 days
        ("2026-07-14", "2026-10-14", None, "longer than 92 days"),
        ("2026-10-10", "2026-10-15", None, "in the future"),
        ("2026-10-14", "2026-10-01", None, "since is after until"),
        ("01/10/2026", "2026-10-14", None, "YYYY-MM-DD"),
        ("2026-10-02", "2026-10-10", ("2026-10-01", "2026-10-14"), None),
        ("2026-09-30", "2026-10-10", ("2026-10-01", "2026-10-14"), "inside the scan window"),
    ],
)
def test_window_limits(since, until, bounds, expected):
    error = window_error(since, until, TODAY, bounds)
    if expected is None:
        assert error is None
    else:
        assert expected in error


PR_URL = "https://github.com/openid/OpenID4VP/pull/802"
DOMAINS = ["github.com", "openid.net"]
LEDGER = {
    "ev_pr": Evidence(
        id="ev_pr",
        url=PR_URL,
        kind="pull_request",
        title="PR #802",
        retrieved_at="2026-10-14T00:00:00+00:00",
        sha256="0" * 64,
        text="PR #802: List encrypted_response_enc_values_supported\n  in client metadata.",
    )
}


def draft(**overrides: object) -> NewsDraft:
    base = {
        "title": "OpenID4VP lists encrypted_response_enc_values_supported in client metadata",
        "summary": "A short summary.",
        "body": "A body.",
        "key_points": [KeyPoint(text="The parameter is listed.", evidence_ids=["ev_pr"])],
        "quotes": [Quote(text="encrypted_response_enc_values_supported in", evidence_id="ev_pr")],
        "tags": ["oid4vp"],
        "refer": "https://github.com/openid/OpenID4VP",
        "diff": PR_URL,
        "rationale": "Verifiers must handle the new metadata.",
    }
    return NewsDraft(**{**base, **overrides})


def test_valid_draft_passes_with_whitespace_insensitive_quote():
    assert validate_draft(draft(), LEDGER, DOMAINS, []) == []


@pytest.mark.parametrize(
    ("overrides", "accepted", "expected"),
    [
        ({"key_points": [KeyPoint(text="x", evidence_ids=[])]}, [], "cites no evidence"),
        (
            {"key_points": [KeyPoint(text="x", evidence_ids=["ev_made_up"])]},
            [],
            "which no tool returned",
        ),
        ({"key_points": []}, [], "at least one key point"),
        (
            {"quotes": [Quote(text="a sentence nobody wrote", evidence_id="ev_pr")]},
            [],
            "not an exact excerpt",
        ),
        (
            {"diff": "https://github.com/openid/OpenID4VP/pull/999"},
            [],
            "diff must be the URL of evidence",
        ),
        ({"refer": "https://evil.example/openid"}, [], "refer must be an https URL"),
        ({"refer": "http://github.com/openid/OpenID4VP"}, [], "refer must be an https URL"),
        ({}, [PR_URL], "already accepted"),
        ({"tags": []}, [], "between 1 and 6 tags"),
        ({"title": "x" * 121}, [], "longer than 120"),
    ],
)
def test_invalid_drafts_are_rejected(overrides, accepted, expected):
    errors = validate_draft(draft(**overrides), LEDGER, DOMAINS, accepted)
    assert any(expected in e for e in errors), errors


def test_subdomain_of_allowed_domain_is_allowed_but_lookalike_is_not():
    ok = draft(refer="https://openid.github.com/x")
    lookalike = draft(refer="https://notgithub.com/x")
    assert validate_draft(ok, LEDGER, DOMAINS, []) == []
    assert validate_draft(lookalike, LEDGER, DOMAINS, []) != []
