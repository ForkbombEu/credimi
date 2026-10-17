# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""NewsAgent: a Temporal Agent Harness agent that researches EUDI changes and drafts news.

Two messages, one agent turn each. The OpenAI Agents SDK drives the reason/act loop (model calls
are Temporal activities) and the tools record evidence in a workflow-owned ledger.

* ``scan``: unattended research of a time window. Drafts go through ``submit_draft``.
* ``ask``: chat. The conversation and its evidence ledger persist across messages. Drafts go
  through ``check_draft`` and then ``save_draft``, which waits for human approval.

After each turn the workflow (not the model) stores the accepted drafts through the Go activity
``Store a news draft``, always unpublished.
"""

import dataclasses
from datetime import timedelta

from temporalio import workflow
from temporalio.contrib.workflow_streams import WorkflowStream
from temporalio.exceptions import ApplicationError

with workflow.unsafe.imports_passed_through():
    from agents import Agent as OpenAIAgent
    from agents import Runner, TResponseInputItem
    from agents.exceptions import AgentsException
    from temporal_agent_harness.ai_sdks.openai_agents_harness import as_openai_agent_tools
    from temporal_agent_harness.harness import agent
    from temporal_agent_harness.harness.agent_protocol import (
        AgentConfig,
        TextMessage,
        TextReply,
        ToolApprovalPolicy,
    )
    from temporal_agent_harness.harness.agent_workflow import AgentWorkflowRunner

    from . import activities
    from .models import (
        CHAT_MAX_DRAFTS,
        CHAT_MAX_MESSAGES,
        CHAT_MAX_TURNS,
        DEFAULT_MODEL,
        GO_STORE_NEWS_DRAFT,
        GO_TASK_QUEUE,
        WORKFLOW_TYPE,
        ScanDecisions,
        ScanReport,
        ScanRequest,
        SourceConfig,
        StoredDraft,
        Usage,
    )
    from .render import render_payload
    from .tools import CHAT_TOOLS, GO_TIMEOUT, RETRY, SCAN_TOOLS, ResearchSession
    from .validation import MAX_WINDOW_DAYS

_ROLE = """\
You are the research editor of Credimi, a conformance and interoperability testing platform for
the EU Digital Identity (EUDI) ecosystem: EUDI Wallets, credential issuers and verifiers, the
EUDI Architecture and Reference Framework (ARF), OpenID4VCI, OpenID4VP, SD-JWT VC and mDoc.
Your readers build and test wallets, issuers and verifiers."""

_RESEARCH = """\
How to research:
- `list_changes` lists a source's releases, merged pull requests and commits between two dates.
  Most commits are editorial (typos, formatting, CI). Look for normative changes: new or changed
  requirements, parameters, formats, flows, security considerations, deprecations, releases.
- Before judging an item, read it: `get_pull_request`, `compare_refs` between releases, or
  `fetch_document` for a referenced specification page. Investigate further when unsure.
- Group related changes into one story instead of one item per commit.
- `search_news` finds stories Credimi already covers, published or not.
- State only what the evidence shows. Drafts cite the evidence ids returned by tools in every
  key point; quotes are verbatim; `diff` is the URL of the evidence for the exact change;
  `refer` is the source's canonical URL.
- Tone: factual and calm, no hype, no clickbait, no exclamation marks."""

SCAN_INSTRUCTIONS = f"""{_ROLE}

Find the changes in the scan window that matter to your readers, investigate them, and draft
news only for what is genuinely newsworthy. Call `list_changes` with the scan window dates.

{_RESEARCH}
- Draft with `submit_draft`. If a submission is rejected, fix every listed problem or drop the
  story.

Finish with your decisions: one entry per item you considered (draft, skip or duplicate) with a
one-line rationale, and a short summary of the scan. Drafting nothing is a valid outcome."""

CHAT_INSTRUCTIONS = f"""{_ROLE}

You answer questions about what changed in the sources below, in a conversation.

{_RESEARCH}
- Dates: use the ones the user gives, as YYYY-MM-DD. Without dates, use the last 14 days and say
  so. A window is at most {MAX_WINDOW_DAYS} days and cannot end after today; split longer
  periods into several `list_changes` calls.
- Answer in short Markdown: what changed, why it matters, with the URL of each change.
- Draft news only when the user asks for it: first `check_draft`, fix every problem it lists,
  then `save_draft` with the same draft. `save_draft` waits until a human approves it in the
  harness UI. Never say a draft is saved unless `save_draft` replied ACCEPTED; if it was denied,
  say so."""


def _context(session: ResearchSession) -> str:
    lines = ["Sources:"]
    for s in session.sources():
        notes = f" Notes: {s.notes}" if s.notes else ""
        lines.append(f"- source `{s.id}`: {s.title} (repo {s.repo}, refer {s.url}).{notes}")
    lines.append(f"Fetchable domains: {', '.join(session.config.allowed_domains)}.")
    return "\n".join(lines)


def _usage(result) -> Usage:
    u = result.context_wrapper.usage
    return Usage(requests=u.requests, input_tokens=u.input_tokens, output_tokens=u.output_tokens)


def _adapt_tools(runner: AgentWorkflowRunner, tools: list, session: ResearchSession) -> list:
    """Harness tools for the SDK, with invalid arguments sent back to the model.

    The harness adapter raises a non-retryable `InvalidToolInput` when the model's arguments
    do not match the tool schema, which fails the whole turn. Models without strict schema
    enforcement (e.g. through Ollama) omit fields, so the error becomes the tool's reply and
    the model can call again.
    """
    adapted = as_openai_agent_tools(runner, tools, injections={"session": session})
    return [dataclasses.replace(t, on_invoke_tool=_reply_invalid_input(t)) for t in adapted]


def _reply_invalid_input(tool):
    invoke = tool.on_invoke_tool

    async def on_invoke_tool(ctx, input: str):  # noqa: A002
        try:
            return await invoke(ctx, input)
        except ApplicationError as exc:
            if exc.type != "InvalidToolInput":
                raise
            return (
                f"ERROR: invalid arguments for {tool.name}: {exc.message}\n"
                "Call it again with every required field."
            )

    return on_invoke_tool


@agent.defn(name=WORKFLOW_TYPE)
class NewsAgentWorkflow:
    @agent.init
    def __init__(self, config: AgentConfig) -> None:
        self._runner = AgentWorkflowRunner(
            config,
            stream=WorkflowStream(),
            # Every tool but `save_draft` is inherently safe; `save_draft` waits for a human.
            approval_policy_default=ToolApprovalPolicy.allow_inherently_safe(),
        )
        self._chat: ResearchSession | None = None
        self._conversation: list[TResponseInputItem] = []
        self._asks = 0
        self._chat_usage = Usage()

    async def _load_sources(self) -> SourceConfig:
        return await workflow.execute_activity(
            activities.load_sources, start_to_close_timeout=timedelta(seconds=10)
        )

    async def _store(self, session: ResearchSession) -> list[StoredDraft]:
        """Store the drafts accepted since the last call, unpublished."""
        stored: list[StoredDraft] = []
        for draft in session.accepted[session.stored :]:
            out = await workflow.execute_activity(
                GO_STORE_NEWS_DRAFT,
                {"payload": render_payload(draft, session.ledger, workflow.info().workflow_id)},
                task_queue=GO_TASK_QUEUE,
                start_to_close_timeout=GO_TIMEOUT,
                retry_policy=RETRY,
                result_type=dict,
            )
            output = out.get("output") or {}
            stored.append(
                StoredDraft(
                    news_id=output.get("id", ""),
                    title=draft.title,
                    diff=draft.diff,
                    duplicate=bool(output.get("duplicate")),
                )
            )
            session.stored += 1
        return stored

    @agent.accepts
    async def scan(self, request: ScanRequest) -> ScanReport:
        """Research the sources over a time window and store newsworthy drafts, unpublished."""
        config = await self._load_sources()
        known = {s.id for s in config.sources}
        if unknown := [i for i in request.source_ids if i not in known]:
            raise ApplicationError(
                f"unknown source ids {unknown}; known: {sorted(known)}", non_retryable=True
            )
        session = ResearchSession(
            config=config,
            today=workflow.now().date(),
            max_drafts=request.max_drafts,
            source_ids=request.source_ids,
            window=(request.since, request.until),
        )
        sdk_agent = OpenAIAgent(
            name="CredimiNewsEditor",
            instructions=SCAN_INSTRUCTIONS,
            model=request.model,
            tools=_adapt_tools(self._runner, SCAN_TOOLS, session),
            output_type=ScanDecisions,
        )
        prompt = (
            f"Scan window: {request.since} to {request.until} (inclusive, UTC).\n"
            f"Draft at most {request.max_drafts} stories.\n{_context(session)}"
        )

        decisions = ScanDecisions(decisions=[], summary="")
        usage = Usage()
        error = ""
        try:
            result = await Runner.run(sdk_agent, input=prompt, max_turns=request.max_turns)
            decisions = result.final_output_as(ScanDecisions)
            usage = _usage(result)
        except AgentsException as exc:
            # e.g. MaxTurnsExceeded: drafts accepted so far passed every check, so keep them.
            error = f"{type(exc).__name__}: {exc}"

        report = ScanReport(
            workflow_id=workflow.info().workflow_id,
            window=f"{request.since}..{request.until}",
            decisions=decisions.decisions,
            summary=decisions.summary,
            stored=await self._store(session),
            rejected_submissions=session.rejected,
            evidence_count=len(session.ledger),
            usage=usage,
            error=error,
        )
        report.text = report_text(report)
        return report

    @agent.accepts
    async def ask(self, message: TextMessage) -> TextReply:
        """Ask about changes in the EUDI sources, e.g. "What changed in OpenID4VP between
        2026-10-01 and 2026-10-14?". Ask for a news draft to have one saved after your approval."""
        if self._asks >= CHAT_MAX_MESSAGES:
            return TextReply(
                text=f"This session reached its limit of {CHAT_MAX_MESSAGES} messages. "
                "Start a new session to continue."
            )
        self._asks += 1
        if self._chat is None:
            self._chat = ResearchSession(
                config=await self._load_sources(),
                today=workflow.now().date(),
                max_drafts=CHAT_MAX_DRAFTS,
            )
        session = self._chat
        session.today = workflow.now().date()
        sdk_agent = OpenAIAgent(
            name="CredimiNewsResearcher",
            instructions=(
                f"{CHAT_INSTRUCTIONS}\n\nToday is {session.today.isoformat()}.\n{_context(session)}"
            ),
            model=DEFAULT_MODEL,
            tools=_adapt_tools(self._runner, CHAT_TOOLS, session),
        )
        try:
            result = await Runner.run(
                sdk_agent,
                input=[*self._conversation, {"role": "user", "content": message.text}],
                max_turns=CHAT_MAX_TURNS,
            )
        except AgentsException as exc:
            # The conversation is left as it was; a draft approved before the failure is kept.
            stored = await self._store(session)
            return TextReply(
                text=f"I could not finish: {type(exc).__name__}: {exc}"
                + _stored_text(stored)
                + self._limits_text()
            )
        self._conversation = result.to_input_list()
        usage = _usage(result)
        self._chat_usage = Usage(
            requests=self._chat_usage.requests + usage.requests,
            input_tokens=self._chat_usage.input_tokens + usage.input_tokens,
            output_tokens=self._chat_usage.output_tokens + usage.output_tokens,
        )
        stored = await self._store(session)
        return TextReply(
            text=str(result.final_output) + _stored_text(stored) + self._limits_text(usage)
        )

    def _limits_text(self, usage: Usage | None = None) -> str:
        used = (
            f"this answer {usage.requests} model calls, {usage.input_tokens}/"
            f"{usage.output_tokens} tokens in/out; "
            if usage
            else ""
        )
        return (
            f"\n\n_Message {self._asks} of {CHAT_MAX_MESSAGES}; {used}session total "
            f"{self._chat_usage.input_tokens}/{self._chat_usage.output_tokens} tokens._"
        )


def _stored_text(stored: list[StoredDraft]) -> str:
    if not stored:
        return ""
    lines = [
        f"- {s.title} (news `{s.news_id}`{', already stored' if s.duplicate else ''})"
        for s in stored
    ]
    return "\n\n**Saved as unpublished drafts:**\n" + "\n".join(lines)


def report_text(report: ScanReport) -> str:
    """Markdown digest of a report, built from its fields (not by the model)."""
    lines = [f"**Scan {report.window}**", "", report.summary or "_No summary._", ""]
    if report.error:
        lines += [f"**Error:** {report.error}", ""]
    lines.append("**Decisions:**")
    lines += [f"- `{d.decision}` {d.ref}: {d.rationale}" for d in report.decisions] or ["- none"]
    lines += ["", "**Stored drafts (unpublished):**"]
    lines += [
        f"- {s.title} (news `{s.news_id}`{', already stored' if s.duplicate else ''}) {s.diff}"
        for s in report.stored
    ] or ["- none"]
    lines += [
        "",
        f"Rejected submissions: {report.rejected_submissions}. "
        f"Evidence retrieved: {report.evidence_count}. "
        f"Model requests: {report.usage.requests}, tokens in/out: "
        f"{report.usage.input_tokens}/{report.usage.output_tokens}.",
    ]
    return "\n".join(lines)
