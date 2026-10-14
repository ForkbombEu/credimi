# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""End to end on a test Temporal server: scripted model, recorded GitHub fixtures, and Python
stand-ins for the two Go activities (same names, same JSON envelope)."""

import uuid
from pathlib import Path
from typing import Any

import pytest
from temporal_agent_harness.harness.agent_client import AgentClient
from temporal_agent_harness.harness.agent_protocol import AgentConfig
from temporal_agent_harness.harness.agent_protocol.events import AgentEventType
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio import activity
from temporalio.client import Client
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import Worker

from credimi_news_agent.activities import ALL_ACTIVITIES
from credimi_news_agent.models import (
    GO_SEARCH_NEWS,
    GO_STORE_NEWS_DRAFT,
    GO_TASK_QUEUE,
    TASK_QUEUE,
    WORKFLOW_TYPE,
    ScanReport,
)
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
    client: Client, source_ids: list[str] | None = None
) -> tuple[ScanReport | None, list[str], str]:
    workflow_id = f"news-scan-test-{uuid.uuid4()}"
    handle = await client.start_workflow(
        WORKFLOW_TYPE, AgentConfig(), id=workflow_id, task_queue=TASK_QUEUE
    )
    request = {
        "since": "2026-10-01",
        "until": "2026-10-14",
        "source_ids": ["openid4vp"] if source_ids is None else source_ids,
    }
    report = None
    failure = ""
    tools: list[str] = []
    stream = await AgentClient(client, workflow_id).send_message(
        "scan", request, on_item=lambda item, _o: item, timeout=120
    )
    async for item in stream:
        event = getattr(item, "event", None)
        if event is None:
            failure = str(item)
            continue
        if event.type == AgentEventType.TOOL_END:
            tools.append(event.tool_name)
        if event.type == AgentEventType.MESSAGE_HANDLER_END:
            report = ScanReport.model_validate(event.output)
    await handle.signal("close")
    return report, tools, failure


async def test_scan_stores_one_checked_draft_and_a_rescan_finds_the_duplicate(env):
    news = FakeCredimiNews()
    async with (
        Worker(
            env.client,
            task_queue=TASK_QUEUE,
            workflows=[NewsAgentWorkflow],
            activities=ALL_ACTIVITIES,
        ),
        Worker(env.client, task_queue=GO_TASK_QUEUE, activities=[news.search, news.store]),
    ):
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


async def ask(client: Client, workflow_id: str, text: str, approve: bool) -> tuple[str, list[str]]:
    """Send one chat message; answer every approval request with `approve`."""
    agent_client = AgentClient(client, workflow_id)
    reply = ""
    approvals: list[str] = []
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
        if event.type == AgentEventType.MESSAGE_HANDLER_END:
            reply = event.output["text"]
    return reply, approvals


@pytest.mark.parametrize("approve", [True, False])
async def test_chat_saves_a_draft_only_after_human_approval(env, approve):
    news = FakeCredimiNews()
    async with (
        Worker(
            env.client,
            task_queue=TASK_QUEUE,
            workflows=[NewsAgentWorkflow],
            activities=ALL_ACTIVITIES,
        ),
        Worker(env.client, task_queue=GO_TASK_QUEUE, activities=[news.search, news.store]),
    ):
        workflow_id = f"news-chat-test-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, AgentConfig(), id=workflow_id, task_queue=TASK_QUEUE
        )

        # Read-only question: no approval, nothing stored.
        reply, approvals = await ask(
            env.client, workflow_id, "What changed in openid4vp 2026-10-01 2026-10-14?", approve
        )
        assert approvals == []
        assert "pull/801" in reply
        assert news.records == []

        # Same session, now a draft: only save_draft waits for a human.
        reply, approvals = await ask(
            env.client, workflow_id, "Please draft news on openid4vp 2026-10-01 2026-10-14", approve
        )
        await handle.signal("close")

        assert approvals == ["save_draft"]
        if approve:
            assert len(news.records) == 1
            assert news.records[0]["diff"] == "https://github.com/openid/OpenID4VP/pull/801"
            assert news.records[0]["id"] in reply
        else:
            assert news.records == []
            assert "not saved" in reply
