# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""End to end on a test Temporal server: scripted model, recorded GitHub fixtures, and Python
stand-ins for the two Go activities (same names, same JSON envelope)."""

import uuid
from datetime import UTC, datetime, timedelta
from pathlib import Path
from typing import Any

import pytest
from temporal_agent_harness.harness.agent_client import AgentClient
from temporal_agent_harness.harness.agent_protocol import AgentConfig
from temporal_agent_harness.harness.agent_protocol.events import AgentEventType
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio import activity
from temporalio.client import Client, WorkflowFailureError
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import Worker

from credimi_news_agent.activities import ALL_ACTIVITIES
from credimi_news_agent.daily import NewsDailyScanWorkflow
from credimi_news_agent.models import (
    GO_SEARCH_NEWS,
    GO_STORE_NEWS_DRAFT,
    GO_TASK_QUEUE,
    TASK_QUEUE,
    WORKFLOW_TYPE,
    DailyScanInput,
    DailyScanResult,
    RunScanInput,
    ScanReport,
    ScanRequest,
    Usage,
)
from credimi_news_agent.scan_runner import ScanFailedError, run_scan, run_scan_activity
from credimi_news_agent.workflow import NewsAgentWorkflow

FIXTURES = Path(__file__).resolve().parents[1] / "fixtures" / "github"


class FakeCredimiNews:
    """In-memory `news` collection behind the Go activity names."""

    def __init__(self) -> None:
        self.records: list[dict[str, Any]] = []

    @activity.defn(name=GO_SEARCH_NEWS)
    async def search(self, inp: dict[str, Any]) -> dict[str, Any]:
        p = inp["payload"]
        query = (p.get("query") or "").lower()
        diffs = set(p.get("diff_urls") or [])
        items = [
            {"id": r["id"], "title": r["title"], "diff": r["diff"], "published": False}
            for r in self.records
            if (diffs and r["diff"] in diffs) or (query and query in r["title"].lower())
        ]
        return {"output": {"items": items}}

    @activity.defn(name=GO_STORE_NEWS_DRAFT)
    async def store(self, inp: dict[str, Any]) -> dict[str, Any]:
        p = inp["payload"]
        for r in self.records:
            if r["diff"] == p["diff"]:
                return {"output": {"id": r["id"], "duplicate": True}}
        record = {"id": f"news{len(self.records)}", **p}
        self.records.append(record)
        return {"output": {"id": record["id"], "duplicate": False}}


@pytest.fixture
async def env(monkeypatch):
    monkeypatch.setenv("NEWS_AGENT_FAKE_MODEL", "1")
    monkeypatch.setenv("NEWS_AGENT_FIXTURES_DIR", str(FIXTURES))
    from agents import set_tracing_disabled

    from credimi_news_agent.worker import openai_plugin

    set_tracing_disabled(True)

    env = await WorkflowEnvironment.start_time_skipping(
        plugins=[openai_plugin(), AgentHarnessPlugin()]
    )
    yield env
    await env.shutdown()


async def scan(
    client: Client, source_ids: list[str] | None = None, **overrides: Any
) -> tuple[ScanReport | None, list[str], str]:
    fields: dict[str, Any] = {
        "since": "2026-10-01",
        "until": "2026-10-14",
        "source_ids": ["openid4vp"] if source_ids is None else source_ids,
    }
    request = ScanRequest(**(fields | overrides))
    tools: list[str] = []

    def on_event(item: Any) -> None:
        if item.event.type == AgentEventType.TOOL_END:
            tools.append(item.event.tool_name)

    try:
        report = await run_scan(client, request, f"news-scan-test-{uuid.uuid4()}", on_event)
    except ScanFailedError as exc:
        return None, tools, str(exc)
    return report, tools, ""


def workers(env: WorkflowEnvironment, news: FakeCredimiNews, *extra_activities) -> tuple:
    return (
        Worker(
            env.client,
            task_queue=TASK_QUEUE,
            workflows=[NewsAgentWorkflow, NewsDailyScanWorkflow],
            activities=[*ALL_ACTIVITIES, *extra_activities],
        ),
        Worker(env.client, task_queue=GO_TASK_QUEUE, activities=[news.search, news.store]),
    )


async def test_scan_stores_one_checked_draft_and_a_rescan_finds_the_duplicate(env):
    news = FakeCredimiNews()
    agent_worker, go_worker = workers(env, news)
    async with agent_worker, go_worker:
        report, tools, failure = await scan(env.client)

        assert failure == ""

        assert tools == [
            "list_changes",
            "get_pull_request",
            "search_news",
            "submit_draft",
            "submit_draft",
        ]
        # The first submission cites evidence that was never retrieved and must be refused.
        assert report.rejected_submissions == 1
        assert [d.decision for d in report.decisions] == ["draft"]
        assert report.problems == []
        assert report.error == ""
        assert len(report.stored) == 1 and not report.stored[0].duplicate
        assert len(news.records) == 1
        stored = news.records[0]
        assert stored["diff"] == "https://github.com/openid/OpenID4VP/pull/801"
        assert "published" not in stored
        assert "<script>" not in stored["news_html"]
        assert "https://github.com/openid/OpenID4VP/pull/801" in stored["news_html"]

        # A form-typed id with stray whitespace still selects the source.
        again, _, failure = await scan(env.client, [" openid4vp "])

        assert failure == ""
        assert [d.decision for d in again.decisions] == ["duplicate"]
        assert again.stored == []
        assert len(news.records) == 1

        unknown, tools, failure = await scan(env.client, ["openid4vpp"])

        assert unknown is None and tools == []
        assert "unknown source ids ['openid4vpp']" in failure


async def test_scan_stops_at_its_token_budget(env):
    news = FakeCredimiNews()
    agent_worker, go_worker = workers(env, news)
    async with agent_worker, go_worker:
        # The scripted model reports 1100 tokens per call: the third call exceeds 2500.
        report, tools, failure = await scan(env.client, max_tokens=2500)

    assert failure == ""
    assert report.error.startswith("TokenBudgetExceeded"), report.error
    assert news.records == []


async def test_scan_repairs_a_report_that_claims_an_unsubmitted_draft(env):
    news = FakeCredimiNews()
    agent_worker, go_worker = workers(env, news)
    async with agent_worker, go_worker:
        report, tools, failure = await scan(env.client, model="scripted-unsubmitted-repaired")

    assert failure == ""
    # The first report claims a draft without submit_draft; the repair turn submits it.
    assert tools == ["list_changes", "get_pull_request", "submit_draft"]
    assert report.problems == []
    assert report.error == ""
    assert [d.decision for d in report.decisions] == ["draft"]
    assert len(news.records) == 1
    # 3 model calls in the first run (two tools, final), 2 in the repair (submit, final).
    assert report.usage.requests == 5


async def test_scan_fails_when_the_repaired_report_still_has_problems(env):
    news = FakeCredimiNews()
    agent_worker, go_worker = workers(env, news)
    async with agent_worker, go_worker:
        report, tools, failure = await scan(env.client, model="scripted-unsubmitted")

    assert failure == ""
    assert tools == ["list_changes", "get_pull_request"]
    assert len(report.problems) == 1
    assert "says draft but no accepted draft covers it" in report.problems[0]
    assert report.error == f"invalid report: {report.problems[0]}"
    assert report.stored == [] and news.records == []


async def test_scan_rejects_an_invalid_window_before_any_model_call(env):
    news = FakeCredimiNews()
    agent_worker, go_worker = workers(env, news)
    async with agent_worker, go_worker:
        report, tools, failure = await scan(env.client, until="2010-10-26")

    assert report is None and tools == []
    assert "invalid window 2026-10-01..2010-10-26: since is after until" in failure


async def ask(
    client: Client, workflow_id: str, text: str, approve: bool
) -> tuple[str, list[str], list[str]]:
    """Send one chat message; answer every approval request with `approve`.

    Returns the reply, the tools that asked for approval and the tools that reported errors.
    """
    agent_client = AgentClient(client, workflow_id)
    reply = ""
    approvals: list[str] = []
    tool_errors: list[str] = []
    stream = await agent_client.send_message(
        "ask", {"text": text}, on_item=lambda item, _o: item, timeout=120
    )
    async for item in stream:
        event = getattr(item, "event", None)
        if event is None:
            raise AssertionError(f"turn failed: {item}")
        if event.type == AgentEventType.TOOL_APPROVAL_REQUESTED:
            approvals.append(event.tool_name)
            await agent_client.approve_tool(event.tool_id, approved=approve, reason="test")
        if event.type == AgentEventType.TOOL_ERROR:
            tool_errors.append(event.tool_name)
        if event.type == AgentEventType.MESSAGE_HANDLER_END:
            reply = event.output["text"]
    return reply, approvals, tool_errors


@pytest.mark.parametrize("approve", [True, False])
async def test_chat_saves_a_draft_only_after_human_approval(env, approve):
    news = FakeCredimiNews()
    agent_worker, go_worker = workers(env, news)
    async with agent_worker, go_worker:
        workflow_id = f"news-chat-test-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, AgentConfig(), id=workflow_id, task_queue=TASK_QUEUE
        )

        # Read-only question: no approval, nothing stored, the retrieved evidence is listed.
        reply, approvals, _ = await ask(
            env.client, workflow_id, "What changed in openid4vp 2026-10-01 2026-10-14?", approve
        )
        assert approvals == []
        assert "pull/801" in reply
        assert "**Evidence retrieved for this answer:**" in reply
        assert news.records == []

        # Same session, now a draft: only save_draft waits for a human.
        reply, approvals, tool_errors = await ask(
            env.client, workflow_id, "Please draft news on openid4vp 2026-10-01 2026-10-14", approve
        )
        await handle.signal("close")

        assert approvals == ["save_draft"]
        # The scripted model's first check_draft call omits `quotes`.
        assert tool_errors == ["check_draft"]
        if approve:
            assert len(news.records) == 1
            assert news.records[0]["diff"] == "https://github.com/openid/OpenID4VP/pull/801"
            assert news.records[0]["id"] in reply
        else:
            assert news.records == []
            assert "not saved" in reply


def _yesterday() -> str:
    return (datetime.now(UTC).date() - timedelta(days=1)).isoformat()


def _stub_scan(error: str, received: list[RunScanInput]):
    @activity.defn(name="news.run_scan")
    async def stub(inp: RunScanInput) -> ScanReport:
        received.append(inp)
        return ScanReport(
            workflow_id=inp.workflow_id,
            window=f"{inp.request.since}..{inp.request.until}",
            decisions=[],
            summary="",
            stored=[],
            rejected_submissions=0,
            evidence_count=0,
            usage=Usage(),
            error=error,
        )

    return stub


async def run_daily(env: WorkflowEnvironment) -> DailyScanResult:
    return await env.client.execute_workflow(
        "NewsDailyScan",
        DailyScanInput(source_ids=["openid4vp"]),
        id=f"news-daily-test-{uuid.uuid4()}",
        task_queue=TASK_QUEUE,
        result_type=DailyScanResult,
    )


async def test_daily_scan_advances_its_cursor_to_yesterday(env):
    received: list[RunScanInput] = []
    agent_worker, go_worker = workers(env, FakeCredimiNews(), _stub_scan("", received))
    async with agent_worker, go_worker:
        result = await run_daily(env)

    yesterday = _yesterday()
    assert result.until == yesterday
    assert [r.workflow_id for r in received] == [f"news-scan-{yesterday}-{yesterday}"]
    assert received[0].request.source_ids == ["openid4vp"]


async def test_daily_scan_fails_when_the_scan_reports_an_error(env):
    agent_worker, go_worker = workers(env, FakeCredimiNews(), _stub_scan("boom", []))
    async with agent_worker, go_worker:
        with pytest.raises(WorkflowFailureError) as failure:
            await run_daily(env)

    assert "failed: boom" in str(failure.value.cause)


async def test_daily_scan_runs_a_real_scan_session(env):
    # No fixtures exist for yesterday, so list_changes fails and the scan is invalid.
    agent_worker, go_worker = workers(env, FakeCredimiNews(), run_scan_activity)
    async with agent_worker, go_worker:
        with pytest.raises(WorkflowFailureError) as failure:
            await run_daily(env)

    assert "invalid scan: no list_changes call succeeded" in str(failure.value.cause)
