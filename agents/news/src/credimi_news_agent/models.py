# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Typed contracts shared by the workflow, its activities, the CLI and the scripted model.

Models the LLM fills in (``NewsDraft``, ``ScanDecisions`` and their parts) declare no defaults,
so the OpenAI strict JSON schema keeps every field required.
"""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field, field_validator

# Task queue polled by this Python worker, in the Credimi `default` namespace.
TASK_QUEUE = "news-agent"
# Task queue of the Go activities registered by `credimi serve` (defaultWorkers).
GO_TASK_QUEUE = "NewsToolsTaskQueue"
GO_SEARCH_NEWS = "Search news"
GO_STORE_NEWS_DRAFT = "Store a news draft"
WORKFLOW_TYPE = "NewsAgent"
DEFAULT_MODEL = "gpt-5.1"

# Chat (`ask`) limits. Each answer may take at most CHAT_MAX_TURNS model calls; a session takes
# at most CHAT_MAX_MESSAGES questions, which also bounds its Temporal history (the harness does
# not continue-as-new); a session saves at most CHAT_MAX_DRAFTS drafts.
CHAT_MAX_TURNS = 15
CHAT_MAX_MESSAGES = 20
CHAT_MAX_DRAFTS = 3


class Source(BaseModel):
    """One monitored source from ``sources.toml``."""

    id: str
    repo: str
    url: str
    title: str
    notes: str = ""


class SourceConfig(BaseModel):
    """The parsed ``sources.toml``: sources plus the domains `fetch_document` may reach."""

    sources: list[Source]
    allowed_domains: list[str]


class ScanRequest(BaseModel):
    """Input of one scan. The window is the cursor: [since, until] as YYYY-MM-DD (UTC)."""

    since: str
    until: str
    source_ids: list[str] = Field(default_factory=list)
    max_drafts: int = 3
    max_turns: int = 30
    model: str = DEFAULT_MODEL

    @field_validator("since", "until", "model")
    @classmethod
    def _strip(cls, value: str) -> str:
        return value.strip()

    @field_validator("source_ids")
    @classmethod
    def _strip_ids(cls, value: list[str]) -> list[str]:
        # Form inputs carry stray whitespace ("openid4vp "); empty entries mean nothing.
        return [v.strip() for v in value if v.strip()]


class Evidence(BaseModel):
    """Something an activity actually retrieved. Drafts may cite only these ids."""

    id: str
    url: str
    kind: Literal["release", "pull_request", "commit", "compare", "document"]
    title: str
    retrieved_at: str
    sha256: str
    text: str


class ChangeItem(BaseModel):
    """One change listed by `list_changes`."""

    evidence_id: str
    kind: Literal["release", "pull_request", "commit"]
    title: str
    url: str
    date: str
    ref: str


class ListChangesInput(BaseModel):
    """Repository and window for `news.list_github_changes`."""

    repo: str
    since: str
    until: str


class ChangeList(BaseModel):
    """Changes of one repository in the window, with their evidence."""

    repo: str
    items: list[ChangeItem]
    evidence: list[Evidence]


class PullRequestInput(BaseModel):
    """A pull request to read."""

    repo: str
    number: int


class CompareInput(BaseModel):
    """Two refs of a repository to compare."""

    repo: str
    base: str
    head: str


class FetchInput(BaseModel):
    """A page to fetch, and the domains it may be on."""

    url: str
    allowed_domains: list[str]


# --- Filled in by the model -------------------------------------------------------------------


class KeyPoint(BaseModel):
    """One statement of a draft, with the evidence that supports it."""

    text: str = Field(description="One factual statement about the change.")
    evidence_ids: list[str] = Field(
        description="Evidence ids (from tool results) that support this statement."
    )


class Quote(BaseModel):
    """A verbatim excerpt of retrieved evidence."""

    text: str = Field(description="Exact text copied from the cited evidence.")
    evidence_id: str


class NewsDraft(BaseModel):
    """A news draft proposed by the agent."""

    title: str = Field(description="Factual headline, at most 120 characters, no clickbait.")
    summary: str = Field(description="One short Markdown paragraph for the news card.")
    body: str = Field(description="Markdown article: what changed, why it matters, who it affects.")
    key_points: list[KeyPoint] = Field(description="The key differences, each with evidence.")
    quotes: list[Quote] = Field(description="Optional verbatim quotes; may be empty.")
    tags: list[str] = Field(description="1 to 6 short topic tags, e.g. oid4vp, arf, sd_jwt_vc.")
    refer: str = Field(description="Canonical URL of the source (repository or spec page).")
    diff: str = Field(
        description="URL of the exact change (pull request, release or compare page). "
        "Must be the URL of evidence you retrieved."
    )
    rationale: str = Field(description="Why this is newsworthy for EUDI conformance testing.")


class ItemDecision(BaseModel):
    """What the agent decided about one change it considered."""

    ref: str = Field(description="Evidence id or URL of the item.")
    decision: Literal["draft", "skip", "duplicate"]
    rationale: str


class ScanDecisions(BaseModel):
    """The agent's final answer for a scan."""

    decisions: list[ItemDecision] = Field(description="One entry per change you considered.")
    summary: str = Field(description="Two or three sentences about this scan.")


# --- Handler output ---------------------------------------------------------------------------


class StoredDraft(BaseModel):
    """A draft stored (or found already stored) in the `news` collection."""

    news_id: str
    title: str
    diff: str
    duplicate: bool


class Usage(BaseModel):
    """Model usage of a scan, as reported by the OpenAI Agents SDK."""

    requests: int = 0
    input_tokens: int = 0
    output_tokens: int = 0


class ScanReport(BaseModel):
    """Result of one scan."""

    workflow_id: str
    window: str
    decisions: list[ItemDecision]
    summary: str
    stored: list[StoredDraft]
    rejected_submissions: int
    evidence_count: int
    usage: Usage
    error: str = ""
    # Markdown digest of the fields above. The harness UI renders a reply's `text` only.
    text: str = ""
