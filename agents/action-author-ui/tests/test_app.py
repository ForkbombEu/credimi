# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

import asyncio
import shutil
import uuid

import httpx
import pytest
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio import workflow
from temporalio.client import Client, WorkflowExecutionStatus
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import UnsandboxedWorkflowRunner, Worker

from credimi_action_author_ui.app import MANAGER_WORKFLOW_ID, _unbind, create_app, registry

UI = "http://127.0.0.1:8001"
SAME_ORIGIN = {"Origin": UI}


def ui_client(base_url: str = UI, app=None) -> httpx.AsyncClient:
    return httpx.AsyncClient(
        transport=httpx.ASGITransport(app=app or create_app()), base_url=base_url
    )


@pytest.mark.parametrize(
    "origin",
    [None, "http://evil.example", "null", "http://127.0.0.1:9999", "https://127.0.0.1:8001"],
)
async def test_state_changes_from_another_origin_are_refused(origin):
    headers = {"Origin": origin} if origin else {}
    async with ui_client() as client:
        response = await client.post(
            "/api/callback-result", json={"tool_id": "t", "result": True}, headers=headers
        )
        bind = await client.post(
            "/api/action-author/namespace", json={"namespace": "org"}, headers=headers
        )
    assert (response.status_code, bind.status_code) == (403, 403)
    assert response.json()["error"] == "cross_origin_request"
    assert "access-control-allow-origin" not in response.headers


@pytest.mark.parametrize("path", ["/api/sessions", "/api/status/session-1"])
async def test_api_reads_from_another_origin_are_refused(path):
    async with ui_client() as client:
        response = await client.get(path, headers={"Origin": "http://evil.example"})
    assert response.status_code == 403
    assert "access-control-allow-origin" not in response.headers


async def test_cors_preflights_get_no_wildcard():
    async with ui_client() as client:
        response = await client.options(
            "/api/callback-result",
            headers={
                "Origin": "http://evil.example",
                "Access-Control-Request-Method": "POST",
            },
        )
    assert response.status_code == 403
    assert "access-control-allow-origin" not in response.headers


@pytest.mark.parametrize(
    "base_url", ["http://evil.example", "http://attacker.test:8001", "http://127.0.0.1:9999"]
)
async def test_requests_for_another_host_are_refused(base_url):
    async with ui_client(base_url) as client:
        page = await client.get("/start")
        post = await client.post("/api/callback-result", json={}, headers={"Origin": base_url})
    assert (page.status_code, post.status_code) == (421, 421)


@pytest.mark.parametrize("base_url", [UI, "http://localhost:8001", "http://LOCALHOST:8001"])
async def test_same_origin_requests_reach_the_ui(base_url):
    async with ui_client(base_url) as client:
        page = await client.get("/start")
        # The unbound guard answers: the request got past the origin check.
        callback = await client.post(
            "/api/callback-result", json={}, headers={"Origin": base_url.lower()}
        )
    assert page.status_code == 200
    assert callback.status_code == 409


async def test_the_ui_port_follows_its_setting(monkeypatch):
    monkeypatch.setenv("ACTION_AUTHOR_UI_PORT", "9123")
    async with ui_client("http://127.0.0.1:9123") as client:
        assert (await client.get("/start")).status_code == 200
    monkeypatch.delenv("ACTION_AUTHOR_UI_PORT")
    async with ui_client() as client:
        assert (await client.get("/start")).status_code == 200
    async with ui_client("http://127.0.0.1:9123") as client:
        assert (await client.get("/start")).status_code == 421


async def test_harness_cannot_start_sessions_with_caller_supplied_actor():
    app = create_app()
    app.state.binding = object()
    async with ui_client(app=app) as client:
        response = await client.post(
            "/api/sessions",
            headers=SAME_ORIGIN,
            json={
                "agent_workflow_type": "ActionAuthor",
                "data": {
                    "actor_id": "someone-else",
                    "actor_collection": "_superusers",
                    "wallet_version_identifier": "other/wallet/build",
                    "slot": "onboarding",
                },
            },
        )
    assert response.status_code == 403
    assert response.json()["error"] == "authenticated_start_required"


async def test_harness_landing_page_opens_authenticated_start():
    async with ui_client() as client:
        response = await client.get("/")
    assert response.status_code == 307
    assert response.headers["location"] == "/start"


async def test_start_page_is_not_cached_or_embeddable():
    async with ui_client() as client:
        response = await client.get("/start")
    assert response.status_code == 200
    assert response.headers["cache-control"] == "no-store"
    assert "frame-ancestors 'none'" in response.headers["content-security-policy"]


def test_registry_never_needs_a_runner_queue():
    (agent,) = registry().agents
    assert (agent.key, agent.workflow_type, agent.task_queue, agent.start) == (
        "credimi-action-author",
        "ActionAuthor",
        "credimi-runner-ActionAuthor",
        None,
    )


async def test_start_page_queues_a_runner_device_session():
    async with ui_client() as client:
        page = (await client.get("/start")).text
    # Devices and versions come from Credimi; the device decides the platform.
    assert "/api/action-author/devices" in page
    assert "/api/action-author/wallet-versions" in page
    assert 'id="platform"' not in page
    assert "No admin-managed runner device is available to your organizations." in page
    # The session is queued, polled, cancellable, and its chat is the `-author` child.
    assert "device_id: device.path" in page
    assert "/api/pipeline/queue/" in page
    assert "method: 'DELETE'" in page
    assert "`${queued.workflow_id}-author`" in page
    assert "Waiting for the device (position ${(queued.position ?? 0) + 1})" in page
    assert "Preparing the device and installing the wallet…" in page
    assert "console.log(\n\t\t\t\t'%c" in page
    # The UI follows the namespace Credimi returns for the session or the device.
    assert "await bindNamespace(result.workflow_namespace)" in page
    assert "selectedDevice()?.namespace" in page


async def test_unbound_ui_sends_the_user_to_the_start_page():
    async with ui_client() as client:
        sessions = await client.get("/api/sessions", headers=SAME_ORIGIN)
        chat = await client.get("/?s=x")
        namespace = await client.get("/api/action-author/namespace")
    assert sessions.status_code == 409
    assert sessions.json()["error"] == "namespace_not_selected"
    assert (chat.status_code, chat.headers["location"]) == (307, "/start?unbound=1")
    assert namespace.json() == {"namespace": None}
    assert namespace.headers["cache-control"] == "no-store"


@pytest.mark.parametrize(
    "body", [{"namespace": "Bad NS"}, {"namespace": 1}, {}, ["org"], {"namespace": "-org"}]
)
async def test_binding_needs_an_organization_name(body):
    async def connect(_namespace):
        raise AssertionError("not called")

    async with ui_client(app=create_app(connect=connect)) as client:
        response = await client.post("/api/action-author/namespace", json=body, headers=SAME_ORIGIN)
    assert response.status_code == 400
    assert response.json()["error"] == "invalid_namespace"


async def test_binding_failure_keeps_the_ui_unbound():
    async def connect(_namespace):
        raise RuntimeError("down")

    async with ui_client(app=create_app(connect=connect)) as client:
        response = await client.post(
            "/api/action-author/namespace", json={"namespace": "org-a"}, headers=SAME_ORIGIN
        )
        namespace = await client.get("/api/action-author/namespace")
    assert response.status_code == 503
    assert response.json() == {"error": "temporal_unavailable", "message": "down"}
    assert namespace.json() == {"namespace": None}


@workflow.defn(name="ActionAuthor")
class FinishedSession:
    @workflow.run
    async def run(self) -> None:
        return None


@pytest.mark.skipif(shutil.which("temporal") is None, reason="needs the temporal CLI")
async def test_binding_follows_the_organization_namespace():
    async with await WorkflowEnvironment.start_local(
        namespace="org-a",
        dev_server_extra_args=["--namespace", "org-b"],
        dev_server_existing_path=shutil.which("temporal"),
    ) as env:
        target = env.client.service_client.config.target_host

        async def connect(namespace):
            return await Client.connect(target, namespace=namespace, plugins=[AgentHarnessPlugin()])

        org_b = await connect("org-b")
        session_id = f"session-{uuid.uuid4()}"
        async with Worker(
            org_b,
            task_queue="test-runner",
            workflows=[FinishedSession],
            # The test module imports httpx, which the workflow sandbox refuses.
            workflow_runner=UnsandboxedWorkflowRunner(),
        ):
            await org_b.execute_workflow(
                FinishedSession.run, id=session_id, task_queue="test-runner"
            )

        app = create_app(connect=connect)
        try:
            async with ui_client(app=app) as client:
                bound = await client.post(
                    "/api/action-author/namespace",
                    json={"namespace": "org-a"},
                    headers=SAME_ORIGIN,
                )
                assert bound.json() == {"namespace": "org-a"}
                sessions = await client.get("/api/sessions", headers=SAME_ORIGIN)
                assert (sessions.status_code, sessions.json()) == (200, [])
                manager = await env.client.get_workflow_handle(MANAGER_WORKFLOW_ID).describe()
                assert manager.status == WorkflowExecutionStatus.RUNNING

                bound = await client.post(
                    "/api/action-author/namespace",
                    json={"namespace": "org-b"},
                    headers=SAME_ORIGIN,
                )
                assert bound.status_code == 200
                namespace = await client.get("/api/action-author/namespace")
                assert namespace.json() == {"namespace": "org-b"}
                listed = []
                for _ in range(50):
                    response = await client.get("/api/sessions", headers=SAME_ORIGIN)
                    assert response.status_code == 200
                    listed = [s for s in response.json() if s["workflow_id"] == session_id]
                    if listed:
                        break
                    await asyncio.sleep(0.1)
                assert [s["closed"] for s in listed] == [True]
        finally:
            await _unbind(app)
