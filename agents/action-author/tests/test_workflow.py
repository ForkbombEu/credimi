# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""End to end on a time-skipping Temporal test server: scripted model, the fake agent-device
MCP server (a real stdio subprocess behind the worker's StatelessMCPServerProvider) and a fake
device for the activities."""

import asyncio
import hashlib
import json
import shutil
import sys
import uuid
from typing import Any

import anthropic
import httpx2 as httpx
import pytest
import yaml
from agents import ModelProvider, set_tracing_disabled
from httpx import ASGITransport, AsyncClient
from temporal_agent_harness.harness.agent_client import AgentClient
from temporal_agent_harness.harness.agent_protocol import AgentConfig, ToolApprovalPolicy
from temporal_agent_harness.harness.agent_protocol.events import AgentEventType
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporal_agent_harness.web.session_manager_worker import create_session_manager_worker
from temporalio import activity
from temporalio.client import Client, WorkflowExecutionStatus
from temporalio.exceptions import ApplicationError
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import Worker

from credimi_action_author.activities import ActionAuthorActivities
from credimi_action_author.anthropic_model import AnthropicModel
from credimi_action_author.contracts import (
    GO_ACTIVITY_QUEUE,
    RESOLVE_BUILD,
    SAVE_ACTION,
    AuthoringContext,
)
from credimi_action_author.fake_device import FakeDevice
from credimi_action_author.fake_model import (
    ScriptedModel,
    _Conversation,
    _text,
    onboarding_action,
)
from credimi_action_author.models import (
    AGENT_DEVICE_SESSION,
    MAX_SUBMISSIONS,
    TASK_QUEUE,
    WORKFLOW_TYPE,
)
from credimi_action_author.ui import create_app
from credimi_action_author.workflow import ActionAuthorWorkflow

APP_ID = "eu.europa.ec.euidi"
PIN = "123456"
SERIAL = "test-serial"
FAKE_MCP = [sys.executable, "-m", "credimi_action_author.fake_device"]
CONTEXT = AuthoringContext(
    wallet_version_identifier="test-org/wallet:build",
    actor_id="test-user",
    actor_collection="users",
    slot="onboarding",
)


class FakeCredimi:
    def __init__(self):
        self.saved = []
        self.save_unavailable = False

    @activity.defn(name=RESOLVE_BUILD)
    async def resolve(self, request: dict) -> dict:
        return {
            "output": {
                "wallet_id": "wallet",
                "version_id": "build",
                "version_identifier": CONTEXT.wallet_version_identifier,
                "owner_namespace": activity.info().workflow_namespace,
                "installer_name": "test.apk",
                "apk_sha256": "a" * 64,
                "download_grant": "test-only-grant",
            }
        }

    @activity.defn(name=SAVE_ACTION)
    async def save(self, request: dict) -> dict:
        if self.save_unavailable:
            raise ApplicationError("save temporarily unavailable", non_retryable=True)
        self.saved.append(request["payload"])
        return {"output": {"record_id": "stored-action", "published": False}}


@pytest.fixture
async def env(monkeypatch, tmp_path, request):
    monkeypatch.setenv("ACTION_AUTHOR_FAKE_MODEL", "1")
    monkeypatch.setenv("ACTION_AUTHOR_FAKE_MCP_LOG", str(tmp_path / "mcp.jsonl"))
    from credimi_action_author.worker import openai_plugin

    provider = getattr(request, "param", None)
    if provider is not None and provider != "local-server":
        monkeypatch.setattr("credimi_action_author.worker.model_provider", lambda: provider)
    set_tracing_disabled(True)
    plugins = [openai_plugin(SERIAL, FAKE_MCP), AgentHarnessPlugin()]
    if provider == "local-server":
        env = await WorkflowEnvironment.start_local(
            plugins=plugins, dev_server_existing_path=shutil.which("temporal")
        )
    else:
        env = await WorkflowEnvironment.start_time_skipping(plugins=plugins)
    monkeypatch.setenv("TEMPORAL_ADDRESS", env.client.service_client.config.target_host)
    monkeypatch.setenv("TEMPORAL_NAMESPACE", env.client.namespace)
    env.author_store = FakeCredimi()
    try:
        async with Worker(
            env.client,
            task_queue=GO_ACTIVITY_QUEUE,
            activities=[env.author_store.resolve, env.author_store.save],
        ):
            yield env
    finally:
        await env.shutdown()


async def ask(client: Client, workflow_id: str, text: str) -> tuple[str, list[str], list[str]]:
    """Exercise authoring without saving; refuse any automatic save review."""
    agent_client = AgentClient(client, workflow_id)
    reply = ""
    tools: list[str] = []
    approvals: list[str] = []
    stream = await agent_client.send_message(
        "ask", {"text": text}, on_item=lambda item, _o: item, timeout=120
    )
    async for item in stream:
        event = getattr(item, "event", None)
        if event is None:
            raise AssertionError(f"turn failed: {item}")
        if event.type == AgentEventType.CALLBACK_REQUESTED:
            assert event.tool_name == "confirm_wallet_action_save"
            await agent_client.provide_callback_result(event.tool_id, result=False)
        if (
            event.type == AgentEventType.TOOL_END
            and event.tool_name != "confirm_wallet_action_save"
        ):
            tools.append(event.tool_name)
        if event.type == AgentEventType.TOOL_APPROVAL_REQUESTED:
            approvals.append(event.tool_name)
        if event.type == AgentEventType.MESSAGE_HANDLER_END:
            reply = event.output["text"]
    return reply, tools, approvals


async def test_accepted_action_embeds_the_literal_pin_and_replays_the_saved_yaml(env, tmp_path):
    device = FakeDevice(serial=SERIAL)
    out_dir = tmp_path / "out"
    activities = ActionAuthorActivities(device, out_dir)
    worker = Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    )
    async with worker:
        workflow_id = f"action-author-test-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        reply, _, approvals = await ask(
            env.client,
            workflow_id,
            f"Write the onboarding action for {APP_ID} with PIN {PIN}",
        )
        await handle.signal("close")

    assert approvals == []
    assert "ACCEPTED" in reply

    accepted = out_dir / APP_ID / "onboarding.yaml"
    saved = accepted.read_text()
    _, commands = list(yaml.safe_load_all(saved))
    assert [command["inputText"] for command in commands if "inputText" in command] == [PIN, PIN]
    assert "${PIN}" not in saved
    assert f"```yaml\n{saved.rstrip()}\n```" in reply

    # Both cleared-state replays execute the self-contained artifact.
    replays = [c for c in device.calls if c[0] == "maestro_test"]
    assert len(replays) == 2
    assert all(call[1] == saved for call in replays)
    assert [c for c in device.calls if c[0] == "clear_app"] == [("clear_app", APP_ID)] * 2
    # Every Maestro step closed the agent-device session first.
    names = [c[0] for c in device.calls]
    assert names.index("close_session") < names.index("inspect_screen")
    for i, name in enumerate(names):
        if name == "maestro_test":
            assert names[i - 2 : i] == ["close_session", "clear_app"]

    # The MCP calls reached the (fake) agent-device pinned to the worker's serial and session.
    calls: list[dict[str, Any]] = [
        json.loads(line) for line in (tmp_path / "mcp.jsonl").read_text().splitlines()
    ]
    assert [c["tool"] for c in calls] == ["open", "snapshot"]
    for call in calls:
        assert call["args"]["serial"] == SERIAL
        assert call["args"]["session"] == AGENT_DEVICE_SESSION
        assert call["args"]["platform"] == "android"
    assert calls[0]["args"]["app"] == APP_ID


async def test_a_failed_replay_is_rejected_with_the_failing_command(env, tmp_path):
    device = FakeDevice(serial=SERIAL, fail_runs=1)
    out_dir = tmp_path / "out"
    activities = ActionAuthorActivities(device, out_dir)
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-test-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        reply, tools, _ = await ask(
            env.client, workflow_id, f"Write the onboarding action for {APP_ID} with PIN {PIN}"
        )
        await handle.signal("close")

    assert tools[-1] == "submit_action"
    assert "REJECTED (attempt 1 of 3): the replay failed" in reply
    assert 'Assert that "Welcome to your Wallet" is visible' in reply
    assert "pin_screen_title" in reply  # the compacted screen at the failure
    assert not (out_dir / APP_ID / "onboarding.yaml").exists()
    assert len([c for c in device.calls if c[0] == "maestro_test"]) == 1


class SequenceModel(ScriptedModel):
    """Exercise the real harness/SDK using required feedback from each boundary."""

    def __init__(self, steps):
        self.steps = steps

    async def get_response(self, input, **kwargs):  # noqa: A002
        conversation = _Conversation(input)
        for (name, output), (expected_name, _, expected_output) in zip(
            conversation.calls, self.steps, strict=False
        ):
            assert name == expected_name
            assert expected_output in output, (name, output)
        if len(conversation.calls) == len(self.steps):
            return _text(conversation.last[1])
        name, arguments, _ = self.steps[len(conversation.calls)]
        return conversation.call(name, arguments)


class SequenceProvider(ModelProvider):
    def __init__(self, steps):
        self.steps = steps

    def get_model(self, model_name):
        return SequenceModel(self.steps)


TARGET = {
    "app_id": APP_ID,
    "slot": "onboarding",
    "parameters": [{"name": "PIN", "value": PIN}],
}
CORRECTED = onboarding_action(APP_ID, PIN)
WRONG_SELECTOR = CORRECTED.replace("dashboard_screen_bottom_navigation_item_home", "missing_home")
COMMENT_ONLY = (
    f"appId: {APP_ID}\n# PIN: ${{PIN}}\n---\n"
    "- launchApp\n- assertVisible: Welcome\n- takeScreenshot: done\n"
)
BAD_HEADER = CORRECTED.replace("name: Onboarding with a new PIN", "1: invalid\nunknown: invalid")


@pytest.mark.parametrize(
    "env",
    [
        SequenceProvider(
            [
                ("open", {"app": APP_ID}, "call set_target"),
                ("maestro_inspect_screen", {}, "call set_target"),
                ("set_target", TARGET, "OK."),
                ("open", {"app": "com.android.settings"}, "only the app under test"),
                ("open", {"app": APP_ID}, "Opened:"),
                ("click", {"target": {"kind": "ref", "ref": "@e1"}}, "Clicked"),
                (
                    "fill",
                    {"target": {"kind": "selector", "selector": "id=pin"}, "text": PIN},
                    f"Filled PIN {PIN}",
                ),
                ("snapshot", {}, f"PIN {PIN}"),
                ("maestro_inspect_screen", {}, "pin_screen_title"),
                ("check_action", {"yaml": BAD_HEADER}, "PROBLEMS"),
                ("check_action", {"yaml": COMMENT_ONLY}, "PROBLEMS"),
                ("submit_action", {"yaml": WRONG_SELECTOR}, "REJECTED"),
                ("check_action", {"yaml": CORRECTED}, "OK:"),
                ("submit_action", {"yaml": CORRECTED}, "ACCEPTED"),
            ]
        )
    ],
    indirect=True,
)
async def test_no_phone_gate_rejects_bad_candidates_and_recovers_after_failed_replay(env, tmp_path):
    device = FakeDevice(serial=SERIAL, fail_runs=1)
    activities = ActionAuthorActivities(device, tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-gate-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        reply, _, approvals = await ask(
            env.client, workflow_id, "Exercise the onboarding boundaries."
        )
        await handle.signal("close")
        await handle.result()

    assert "ACCEPTED" in reply
    assert not approvals
    replays = [call for call in device.calls if call[0] == "maestro_test"]
    # The refused inspection before `set_target` never reaches the phone: one inspection
    # after the target, one for the failed replay's report.
    assert [call for call in device.calls if call[0] == "inspect_screen"] == [
        ("inspect_screen",)
    ] * 2
    assert len(replays) == 3
    assert "missing_home" in replays[0][1]
    assert all("missing_home" not in call[1] for call in replays[1:])
    saved = (tmp_path / "out" / APP_ID / "onboarding.yaml").read_text()
    assert "missing_home" not in saved
    assert "${PIN}" not in saved
    assert PIN in saved
    calls = [json.loads(line) for line in (tmp_path / "mcp.jsonl").read_text().splitlines()]
    assert all(call["args"].get("app", APP_ID) == APP_ID for call in calls)
    assert any(call["tool"] == "click" for call in calls)
    assert any(call["tool"] == "fill" for call in calls)


@pytest.mark.parametrize(
    "env",
    [
        SequenceProvider(
            [
                ("set_target", TARGET, "OK."),
                ("open", {"app": APP_ID}, "Opened:"),
                (
                    "set_target",
                    {
                        "app_id": APP_ID,
                        "slot": "onboarding",
                        "parameters": [{"name": "PIN", "value": "654321"}],
                    },
                    "OK.",
                ),
                ("open", {"app": "org.example.wallet"}, "only the app under test"),
                ("open", {"app": APP_ID}, "Opened:"),
                (
                    "fill",
                    {"target": {"kind": "point", "x": 100, "y": 200}, "text": "654321"},
                    "Filled PIN 654321",
                ),
            ],
        )
    ],
    indirect=True,
)
async def test_same_turn_target_change_updates_app_scope(env, tmp_path):
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-retarget-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        await ask(env.client, workflow_id, "Select and change the onboarding target.")
        await handle.signal("close")
        await handle.result()
    calls = [json.loads(line) for line in (tmp_path / "mcp.jsonl").read_text().splitlines()]
    assert [call["args"]["app"] for call in calls if call["tool"] == "open"] == [
        APP_ID,
        APP_ID,
    ]


@pytest.mark.parametrize(
    "env",
    [
        SequenceProvider(
            [
                ("set_target", TARGET, "OK."),
                ("click", {"target": {"kind": "ref", "ref": "@e1"}}, ""),
            ]
        )
    ],
    indirect=True,
)
async def test_failed_device_interaction_is_not_retried(env, tmp_path, monkeypatch):
    from credimi_action_author.device import AgentDeviceServer

    original = AgentDeviceServer.call_tool
    attempts = []

    async def fail_click(server, tool_name, arguments, meta=None):
        if tool_name == "click":
            attempts.append(tool_name)
            raise TypeError("simulated device boundary failure")
        return await original(server, tool_name, arguments, meta)

    monkeypatch.setattr(AgentDeviceServer, "call_tool", fail_click)
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-no-retry-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )

        async def consume():
            stream = await AgentClient(env.client, workflow_id).send_message(
                "ask", {"text": "Select target then click."}, on_item=lambda item, offset: item
            )
            return [item async for item in stream]

        await asyncio.wait_for(consume(), 15)
        history = await handle.fetch_history()
        failures = [
            event.activity_task_failed_event_attributes
            for event in history.events
            if event.HasField("activity_task_failed_event_attributes")
        ]
        assert len(failures) == 1
        assert "simulated device boundary failure" in str(failures[0].failure)
        assert attempts == ["click"]
        await handle.signal("close")
        await handle.result()


async def send_handler(client, workflow_id, name, payload):
    reply = ""
    stream = await AgentClient(client, workflow_id).send_message(
        name, payload, on_item=lambda item, _offset: item, timeout=120
    )
    async for item in stream:
        event = getattr(item, "event", None)
        assert event is not None, item
        if event.type == AgentEventType.MESSAGE_HANDLER_END:
            reply = event.output["text"]
    return reply


async def pending_review(browser_client, workflow_id, previous_id=""):
    async with asyncio.timeout(15):
        while True:
            response = await browser_client.get(f"/api/status/{workflow_id}")
            response.raise_for_status()
            for pending in response.json()["pending_callbacks"]:
                if (
                    pending["tool_name"] == "confirm_wallet_action_save"
                    and pending["tool_id"] != previous_id
                ):
                    return pending
            await asyncio.sleep(0.02)


@pytest.mark.parametrize("env", ["local-server"], indirect=True)
@pytest.mark.parametrize(
    ("prompt", "expected_name"),
    [
        ("Write the onboarding action", "Onboarding"),
        ('Write action "First setup" for onboarding', "First setup"),
        ('Write the onboarding action and store it as "Demo onboarding"', "Demo onboarding"),
    ],
)
async def test_accept_saves_the_named_candidate_and_preserves_closed_history(
    env, tmp_path, prompt, expected_name
):
    from temporal_agent_harness.harness.agent_client import CallbackResultError

    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-approval-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE,
            args=[
                AgentConfig(
                    approval_policy=ToolApprovalPolicy(dangerously_skip_all_approvals=True)
                ),
                CONTEXT,
            ],
            id=workflow_id,
            task_queue=TASK_QUEUE,
        )
        turn = None
        try:
            app = create_app()
            app.state.temporal = env.client
            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://harness"
            ) as browser_client:
                turn = asyncio.create_task(
                    browser_client.post(
                        "/api/chat",
                        json={
                            "session_id": workflow_id,
                            "message": {
                                "type": "ask",
                                "payload": {"text": f"{prompt} for {APP_ID} with PIN {PIN}"},
                            },
                        },
                    )
                )
                review = await pending_review(browser_client, workflow_id)
                saved_yaml = (tmp_path / "out" / APP_ID / "onboarding.yaml").read_text()
                digest = hashlib.sha256(saved_yaml.encode()).hexdigest()
                assert review["tool_input"] == {
                    "name": expected_name,
                    "wallet": "wallet",
                    "build": CONTEXT.wallet_version_identifier,
                    "category": "onboarding",
                    "candidate_sha256": digest,
                    "replay_count": 2,
                    "published": False,
                }
                assert env.author_store.saved == []
                assert not turn.done()
                agent_client = AgentClient(env.client, workflow_id)
                for invalid in ("true", 1, {"confirm": True}):
                    with pytest.raises(CallbackResultError) as failure:
                        await agent_client.provide_callback_result(
                            review["tool_id"], result=invalid
                        )
                    assert failure.value.error_type == "MalformedCallbackResult"
                with pytest.raises(CallbackResultError) as failure:
                    await agent_client.provide_callback_result("another-candidate", result=True)
                assert failure.value.error_type == "UnknownCallback"
                interface = (await browser_client.get(f"/api/agent-interface/{workflow_id}")).json()
                assert [handler["name"] for handler in interface] == ["ask"]
                result = await browser_client.post(
                    "/api/callback-result",
                    json={
                        "session_id": workflow_id,
                        "tool_id": review["tool_id"],
                        "result": True,
                    },
                )
                assert result.status_code == 200
                receipt = await asyncio.wait_for(turn, timeout=5)
                assert receipt.status_code == 200
                assert "Saved wallet action" in receipt.text
                await asyncio.wait_for(handle.result(), timeout=3)
                history = await browser_client.get(
                    "/api/attach", params={"session_id": workflow_id}
                )
                events = [
                    json.loads(line[6:])
                    for line in history.text.splitlines()
                    if line.startswith("data: ")
                ]
                saved_event = next(
                    event
                    for event in events
                    if "Saved wallet action" in event.get("output", {}).get("text", "")
                )
                resumed = await browser_client.get(
                    "/api/attach",
                    params={
                        "session_id": workflow_id,
                        "from_offset": saved_event["resume_offset"] - 1,
                    },
                )
                assert "Saved wallet action" in resumed.text
                async with (
                    create_session_manager_worker(env.client),
                    app.router.lifespan_context(app),
                ):
                    listed = (await browser_client.get("/api/sessions")).json()
                    closed = next(row for row in listed if row["workflow_id"] == workflow_id)
                    assert closed["closed"] is True
                    assert closed["execution_status"] == "COMPLETED"
            assert [record["code"] for record in env.author_store.saved] == [saved_yaml]
            assert env.author_store.saved[0]["name"] == expected_name
            assert env.author_store.saved[0]["replay_count"] == 2
            assert env.author_store.saved[0]["apk_sha256"] == "a" * 64
            assert env.author_store.saved[0]["replace_action_id"] == ""
            assert (await handle.describe()).status == WorkflowExecutionStatus.COMPLETED
        finally:
            if (await handle.describe()).status == WorkflowExecutionStatus.RUNNING:
                await handle.signal("close")
                await handle.result()
            if turn:
                await asyncio.gather(turn, return_exceptions=True)


@pytest.mark.parametrize("env", ["local-server"], indirect=True)
async def test_refusal_and_failed_save_require_a_new_explicit_decision(env, tmp_path):
    from temporal_agent_harness.harness.agent_client import CallbackResultError

    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-refusal-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        turn = None
        try:
            app = create_app()
            app.state.temporal = env.client
            async with AsyncClient(
                transport=ASGITransport(app=app), base_url="http://harness"
            ) as browser_client:

                def start_turn(pin):
                    return asyncio.create_task(
                        browser_client.post(
                            "/api/chat",
                            json={
                                "session_id": workflow_id,
                                "message": {
                                    "type": "ask",
                                    "payload": {
                                        "text": f"Write onboarding for {APP_ID} with PIN {pin}"
                                    },
                                },
                            },
                        )
                    )

                agent_client = AgentClient(env.client, workflow_id)
                turn = start_turn(PIN)
                first = await pending_review(browser_client, workflow_id)
                await agent_client.provide_callback_result(first["tool_id"], result=False)
                assert "Not saved" in (await asyncio.wait_for(turn, timeout=5)).text
                assert env.author_store.saved == []
                assert (await handle.describe()).status == WorkflowExecutionStatus.RUNNING
                with pytest.raises(CallbackResultError):
                    await agent_client.provide_callback_result(first["tool_id"], result=True)
                bypass = await send_handler(
                    env.client,
                    workflow_id,
                    "save_wallet_action",
                    {
                        "name": "Bypass",
                        "candidate_sha256": first["tool_input"]["candidate_sha256"],
                        "confirm": True,
                    },
                )
                assert "Accept/Refuse" in bypass
                assert env.author_store.saved == []

                turn = start_turn("654321")
                revised = await pending_review(browser_client, workflow_id, first["tool_id"])
                assert (
                    revised["tool_input"]["candidate_sha256"]
                    != first["tool_input"]["candidate_sha256"]
                )
                with pytest.raises(CallbackResultError):
                    await agent_client.provide_callback_result(first["tool_id"], result=True)
                env.author_store.save_unavailable = True
                await agent_client.provide_callback_result(revised["tool_id"], result=True)
                retry = await pending_review(browser_client, workflow_id, revised["tool_id"])
                assert retry["tool_input"] == revised["tool_input"]
                assert env.author_store.saved == []
                assert (await handle.describe()).status == WorkflowExecutionStatus.RUNNING
                env.author_store.save_unavailable = False
                await agent_client.provide_callback_result(retry["tool_id"], result=True)
                response = await asyncio.wait_for(turn, timeout=5)
                assert "Saved wallet action" in response.text
                await asyncio.wait_for(handle.result(), timeout=3)
                assert len(env.author_store.saved) == 1
                _, commands = list(yaml.safe_load_all(env.author_store.saved[0]["code"]))
                assert [item["inputText"] for item in commands if "inputText" in item] == [
                    "654321",
                    "654321",
                ]
        finally:
            if (await handle.describe()).status == WorkflowExecutionStatus.RUNNING:
                await handle.signal("close")
                await handle.result()
            if turn:
                await asyncio.gather(turn, return_exceptions=True)


@pytest.mark.parametrize("env", ["local-server"], indirect=True)
async def test_ending_a_session_with_a_pending_review_does_not_save(env, tmp_path):
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-cancel-review-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        app = create_app()
        app.state.temporal = env.client
        async with AsyncClient(
            transport=ASGITransport(app=app), base_url="http://harness"
        ) as browser_client:
            turn = asyncio.create_task(
                browser_client.post(
                    "/api/chat",
                    json={
                        "session_id": workflow_id,
                        "message": {
                            "type": "ask",
                            "payload": {"text": f"Write onboarding for {APP_ID} with PIN {PIN}"},
                        },
                    },
                )
            )
            try:
                await pending_review(browser_client, workflow_id)
                await handle.signal("close")
                response = await asyncio.wait_for(turn, timeout=5)
                assert "Not saved" in response.text
                await asyncio.wait_for(handle.result(), timeout=3)
                assert env.author_store.saved == []
                assert (await handle.describe()).status == WorkflowExecutionStatus.COMPLETED
            finally:
                if (await handle.describe()).status == WorkflowExecutionStatus.RUNNING:
                    await handle.signal("close")
                    await handle.result()
                await asyncio.gather(turn, return_exceptions=True)


@pytest.mark.parametrize(
    "env",
    [
        SequenceProvider(
            [
                (
                    "set_target",
                    {**TARGET, "app_id": "org.other.wallet"},
                    "cannot target another app",
                ),
                ("set_target", TARGET, "OK."),
            ]
        )
    ],
    indirect=True,
)
async def test_selected_apk_rejects_another_package(env, tmp_path):
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-build-bound-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        try:
            await ask(env.client, workflow_id, "Select the requested app.")
            assert not (tmp_path / "out").exists()
        finally:
            await handle.signal("close")
            await handle.result()


class SubmitUntilAcceptedModel(ScriptedModel):
    """Submits the corrected action on every new message until a replay accepts it."""

    async def get_response(self, input, **kwargs):  # noqa: A002
        conversation = _Conversation(input)
        if not conversation.calls:
            return conversation.call("set_target", TARGET)
        _, output = conversation.last
        new_message = input[-1].get("role") == "user"
        if new_message or ("the replay failed" in output) or output.startswith("OK."):
            return conversation.call("submit_action", {"yaml": CORRECTED})
        return _text(output)


class SubmitUntilAcceptedProvider(ModelProvider):
    def get_model(self, model_name):
        return SubmitUntilAcceptedModel()


@pytest.mark.parametrize("env", [SubmitUntilAcceptedProvider()], indirect=True)
async def test_refusing_the_last_submission_restores_the_submission_budget(env, tmp_path):
    # The first two submissions fail their first replay; the third is accepted.
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL, fail_runs=2), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-refusal-budget-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        try:
            # `ask` refuses every save review.
            first, _, _ = await ask(env.client, workflow_id, f"Write onboarding with PIN {PIN}")
            assert f"ACCEPTED (attempt {MAX_SUBMISSIONS} of {MAX_SUBMISSIONS})" in first
            assert "Not saved" in first
            second, _, _ = await ask(env.client, workflow_id, "Revise it.")
            assert f"ACCEPTED (attempt 1 of {MAX_SUBMISSIONS})" in second
            assert env.author_store.saved == []
        finally:
            await handle.signal("close")
            await handle.result()


class BudgetRecoveryModel(ScriptedModel):
    async def get_response(self, input, **kwargs):  # noqa: A002
        outputs = [item for item in input if item.get("type") == "function_call_output"]
        conversation = _Conversation(input)
        if not outputs:
            return conversation.call("set_target", TARGET)
        if not any(
            item.get("role") == "user" and PIN in str(item.get("content")) for item in input
        ):
            return _text("Original onboarding request is missing.")
        if len(outputs) == 1:
            return conversation.call("reset_app", {})
        return _text("Exploration resumed.")


class BudgetRecoveryProvider(ModelProvider):
    def get_model(self, model_name):
        return BudgetRecoveryModel()


@pytest.mark.parametrize(
    "env",
    [BudgetRecoveryProvider()],
    indirect=True,
)
@pytest.mark.parametrize(
    ("limit", "value", "stop"),
    [
        ("ACTION_AUTHOR_TOKEN_BUDGET", "1500", "TokenBudgetExceeded"),
        ("ACTION_AUTHOR_MAX_MODEL_CALLS", "1", "MaxTurnsExceeded"),
    ],
)
async def test_per_message_limits_preserve_completed_exploration(
    env, tmp_path, monkeypatch, limit, value, stop
):
    monkeypatch.setenv(limit, value)
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-budget-recovery-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        try:
            first, first_tools, _ = await ask(
                env.client, workflow_id, f"Write onboarding with PIN {PIN}"
            )
            assert stop in first
            assert "Send Continue to resume" in first
            assert first_tools == ["set_target"]
            second, second_tools, _ = await ask(env.client, workflow_id, "Continue.")
            assert stop in second
            assert second_tools == ["reset_app"]
            assert env.author_store.saved == []
        finally:
            await handle.signal("close")
            await handle.result()


class NativeLoopProvider(ModelProvider):
    def __init__(self):
        self.requests = []
        self.client = anthropic.AsyncAnthropic(
            api_key="test-only",
            max_retries=0,
            http_client=httpx.AsyncClient(transport=httpx.MockTransport(self.respond)),
        )
        self.model = AnthropicModel(self.client, "claude-sonnet-5-5")

    def respond(self, request):
        body = json.loads(request.content)
        if self.requests:
            assert body["system"] == self.requests[0]["system"]
            assert body["messages"][0] == self.requests[0]["messages"][0]
        self.requests.append(body)
        results = [
            block
            for message in body["messages"]
            for block in message["content"]
            if block["type"] == "tool_result"
        ]
        if len(results) < 2:
            name, arguments = ("set_target", TARGET) if not results else ("reset_app", {})
            content = [
                {
                    "type": "tool_use",
                    "id": f"native_call_{len(results)}",
                    "name": name,
                    "input": arguments,
                }
            ]
        else:
            assert any(
                block.get("signature") == "signed-native-state"
                for message in body["messages"]
                for block in message["content"]
            )
            content = [{"type": "text", "text": "Native exploration ready."}]
        return httpx.Response(
            200,
            json={
                "id": f"msg_native_{len(self.requests)}",
                "type": "message",
                "role": "assistant",
                "model": "claude-sonnet-5-5",
                "content": [
                    {"type": "thinking", "thinking": "", "signature": "signed-native-state"},
                    *content,
                ],
                "stop_reason": "tool_use" if len(results) < 2 else "end_turn",
                "stop_sequence": None,
                "usage": {
                    "input_tokens": 100,
                    "output_tokens": 20,
                    "cache_creation_input_tokens": 200,
                    "cache_read_input_tokens": 700,
                },
            },
        )

    def get_model(self, model_name):
        return self.model


@pytest.mark.parametrize("env", [NativeLoopProvider()], indirect=True)
async def test_native_tool_loop_keeps_signed_prefix_and_human_save_boundary(env, tmp_path):
    activities = ActionAuthorActivities(FakeDevice(serial=SERIAL), tmp_path / "out")
    async with Worker(
        env.client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    ):
        workflow_id = f"action-author-native-{uuid.uuid4()}"
        handle = await env.client.start_workflow(
            WORKFLOW_TYPE, args=[AgentConfig(), CONTEXT], id=workflow_id, task_queue=TASK_QUEUE
        )
        try:
            first, tools, _ = await ask(env.client, workflow_id, f"Write onboarding with PIN {PIN}")
            assert tools == ["set_target", "reset_app"]
            assert "Native exploration ready." in first
            second, tools, _ = await ask(
                env.client, workflow_id, "Continue reviewing the same wallet."
            )
            assert "Native exploration ready." in second
            assert tools == []
            assert env.author_store.saved == []
        finally:
            await handle.signal("close")
            await handle.result()
