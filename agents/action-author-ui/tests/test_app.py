# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

import asyncio
import base64
import json
import logging
import shutil
import uuid

import httpx
import pytest
from temporal_agent_harness.harness.agent_protocol import AgentEvent, ReplyDelta
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio import workflow
from temporalio.client import Client, WorkflowExecutionStatus, WorkflowQueryFailedError
from temporalio.service import RPCError, RPCStatusCode
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
        chat = await client.post("/api/chat", json={}, headers={"Origin": base_url.lower()})
    assert page.status_code == 200
    assert chat.status_code == 409


@pytest.mark.parametrize("bound", [False, True])
@pytest.mark.parametrize("method", ["GET", "POST", "PUT"])
async def test_reviews_are_never_settled_through_the_ui(bound, method):
    app = create_app()
    if bound:
        app.state.binding = object()
    async with ui_client(app=app) as client:
        response = await client.request(
            method,
            "/api/callback-result",
            json={"session_id": "s", "tool_id": "t", "result": True},
            headers=SAME_ORIGIN,
        )
    assert response.status_code == 403
    assert response.json() == {
        "error": "review_through_credimi",
        "message": "Save reviews are sent to Credimi.",
    }


async def test_reviews_are_sent_to_credimi():
    async with ui_client() as client:
        script = (await client.get("/action-save-review.js")).text
    assert "/api/callback-result" not in script
    # The save callback gets a card; its native response card is hidden.
    assert "const tools = { confirm_wallet_action_save: saveReview };" in script
    assert '`[aria-label="Response: ${name}"]`' in script
    assert "Object.hasOwn(tools, item.tool_name) && !settled.has(item.tool_id)" in script
    assert "fetch(`${auth.origin}${review.endpoint}`" in script
    assert "endpoint: '/api/action-author/reviews'," in script
    assert "Authorization: `Bearer ${auth.token}`" in script
    for field in (
        "namespace,",
        "author_workflow_id: session,",
        "tool_id: proposal.tool_id,",
        "...review.body(proposal, accepted),",
        "candidate_sha256: proposal.tool_input.candidate_sha256,",
        "plan: proposal.tool_input.plan,",
    ):
        assert field in script
    # The token comes from the start page; a missing or expired one asks for a key.
    assert "const authStorageKey = 'credimi-action-author';" in script
    assert "response.status === 401" in script
    assert "'Credimi user API key'" in script
    assert "'Sign in to Credimi'" in script
    assert "'Credimi URL'" in script
    assert "/api/apikey/authenticate" in script
    assert "response.status === 409" in script
    assert "'This review is no longer pending. Reloading…'" in script
    # A refused proposal keeps its card and shows why, so the user can still refuse it.
    assert "if (response.status === 400) {" in script
    assert "statusText(apiMessage(body, response.status));" in script


async def test_save_review_shows_the_yaml_plan_and_checks_its_hash():
    async with ui_client() as client:
        script = (await client.get("/action-save-review.js")).text
    # The YAML and the plan are shown as text only, in bounded scrollable blocks.
    assert "typeof input.code === 'string' ? input.code : ''" in script
    assert "block.textContent = text;" in script
    assert "innerHTML" not in script
    assert "max-height: 40vh; overflow: auto;" in script
    # The plan is the full plan JSON the agent sent; a candidate without one blocks Accept.
    assert "JSON.stringify(input.plan, null, 2)" in script
    assert "'No plan: refuse this candidate.'" in script
    assert "if (!hasPlan(input)) return false;" in script
    assert "cannotAccept: (input) => (hasPlan(input) ? hashMismatch : noPlan)," in script
    # No token or other plan_* field travels with the plan.
    assert "plan_" not in script
    # Accept needs the shown YAML to hash to the digest Credimi's approval signs.
    assert "crypto.subtle.digest('SHA-256', new TextEncoder().encode(text))" in script
    assert "byte.toString(16).padStart(2, '0')" in script
    assert "(await sha256Hex(input.code)) === input.candidate_sha256" in script
    assert "typeof input.code !== 'string'" in script
    assert "'The reviewed YAML does not match its hash. Refuse this candidate.'" in script
    assert "button.dataset.accept === 'true' && card.dataset.canAccept !== 'true'" in script
    assert "if (accepted && card?.dataset.canAccept !== 'true')" in script


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
    assert response.headers["x-frame-options"] == "DENY"
    policy = response.headers["content-security-policy"]
    assert policy.startswith("default-src 'none';")
    assert policy.count("frame-ancestors") == 1
    assert "frame-ancestors 'none'" in policy


CHAT_PAGE_CSP = (
    "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "
    "img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' https: http:; "
    "object-src 'none'; base-uri 'none'; frame-ancestors 'none'"
)


async def test_chat_page_is_not_embeddable():
    app = create_app()
    app.state.binding = object()
    async with ui_client(app=app) as client:
        response = await client.get("/?s=x")
    assert response.status_code == 200
    assert response.headers["x-frame-options"] == "DENY"
    assert response.headers["content-security-policy"] == CHAT_PAGE_CSP


async def test_chat_page_runs_only_scripts_of_its_own_origin():
    app = create_app()
    app.state.binding = object()
    async with ui_client(app=app) as client:
        page = (await client.get("/?s=x")).text
        new_session = await client.get("/action-new-session.js")
    # script-src 'self' refuses inline scripts: every script the page runs has a src.
    assert "<script>" not in page
    assert '<script src="/action-new-session.js"></script>' in page
    assert '<script type="module" src="/action-save-review.js"></script>' in page
    assert new_session.status_code == 200
    assert new_session.headers["content-type"].startswith("text/javascript")
    assert "window.location.assign('/start')" in new_session.text


async def test_other_html_pages_get_the_chat_page_policy():
    async with ui_client() as client:
        response = await client.get("/start?unbound=1")
        redirect = await client.get("/?s=x")
        script = await client.get("/action-save-review.js")
    # /start keeps its own policy.
    assert response.headers["content-security-policy"].startswith("default-src 'none';")
    # Non-HTML responses only refuse framing.
    assert redirect.status_code == 307
    assert redirect.headers["content-security-policy"] == "frame-ancestors 'none'"
    assert script.headers["content-security-policy"] == "frame-ancestors 'none'"


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
    # Each device names whose runner and which model would author on it.
    assert "Runner of ${device.runner_owner}" in page
    assert "model: ${device.model || 'unknown'}" in page


async def test_start_page_needs_a_running_action_author():
    async with ui_client() as client:
        page = (await client.get("/start")).text
    # Devices of runners without a polling agent are listed but not selectable.
    assert "device.is_online && device.agent_available" in page
    assert "' · offline'" in page
    assert "' · agent not enabled'" in page
    assert "error.reason === 'agent_unavailable'" in page
    # The page queries the chat only after it ran a workflow task, and blames the runner
    # if it does not, keeping Cancel available while it polls on.
    assert "/api/action-author/chat-ready/${id}" in page
    assert (
        "The runner's action author is not answering. Check that [agent] is enabled on the "
        "runner, then cancel and retry."
    ) in page
    assert "CHAT_READY_TIMEOUT = 60000" in page
    assert "cancelButton.hidden = !preparing && !notAnswering" in page


async def test_start_page_labels_store_wallets():
    async with ui_client() as client:
        page = (await client.get("/start")).text
    # A wallet installed from the store at session start is labeled as such.
    assert "version.source === 'store'" in page
    assert (
        "`${version.wallet_name} · installed from the store · "
        "${version.organization_name} (${version.identifier})`"
    ) in page
    assert (
        "Play Store installs need a signed-in account on the device. "
        "iOS simulators have no App Store: upload a simulator .app as a wallet version."
    ) in page


async def test_start_page_leaves_the_plan_to_the_chat():
    async with ui_client() as client:
        page = (await client.get("/start")).text
    # The start request names only the device and the wallet version.
    assert (
        "body: JSON.stringify({\n"
        "\t\t\t\t\t\t\tdevice_id: device.path,\n"
        "\t\t\t\t\t\t\twallet_version_identifier: versionSelect.value,\n"
        "\t\t\t\t\t\t}),"
    ) in page
    for removed in ('id="slot"', 'id="spec-section"', "sessionSpec", "/wallet-actions"):
        assert removed not in page
    assert (
        "Describe the action in the chat: the agent sets its plan, and you review it with the "
        "YAML before saving."
    ) in page
    # The chat's save reviews reuse the Credimi session token of this tab.
    assert "sessionStorage.setItem(AUTH_STORAGE_KEY" in page
    assert "sessionStorage.removeItem(AUTH_STORAGE_KEY)" in page


class _BusyHandle:
    def __init__(self, status: RPCStatusCode) -> None:
        self._status = status

    async def describe(self):
        raise RPCError("consistent query buffer is full", self._status, b"")

    async def query(self, _name):
        raise RPCError("consistent query buffer is full", self._status, b"")


class _BusyTemporal:
    def __init__(self, status: RPCStatusCode) -> None:
        self._status = status

    def get_workflow_handle(self, _workflow_id):
        return _BusyHandle(self._status)


class _IdleTemporal:
    """A chat in ``status`` whose idle_state query answers ``state``, or raises ``query_error``."""

    def __init__(
        self,
        state: dict,
        status: WorkflowExecutionStatus = WorkflowExecutionStatus.RUNNING,
        query_error: Exception | None = None,
    ) -> None:
        self.state = state
        self.status = status
        self.query_error = query_error
        self.queries: list[tuple[str, str]] = []

    def get_workflow_handle(self, workflow_id):
        temporal = self

        class Description:
            status = temporal.status

        class Handle:
            async def describe(self):
                return Description()

            async def query(self, name):
                temporal.queries.append((workflow_id, name))
                if temporal.query_error is not None:
                    raise temporal.query_error
                return temporal.state

        return Handle()


async def _idle(temporal: _IdleTemporal) -> httpx.Response:
    app = create_app()
    app.state.binding = object()
    app.state.temporal = temporal
    async with ui_client(app=app) as client:
        return await client.get("/api/action-author/idle/session-1-author", headers=SAME_ORIGIN)


async def test_idle_state_is_the_chat_workflows_query():
    temporal = _IdleTemporal({"closes_at": "2026-11-03T10:05:00+00:00"})
    response = await _idle(temporal)
    assert response.status_code == 200
    assert response.json() == {"closes_at": "2026-11-03T10:05:00+00:00"}
    assert response.headers["cache-control"] == "no-store"
    assert temporal.queries == [("session-1-author", "idle_state")]


async def test_idle_state_is_empty_for_a_closed_chat():
    temporal = _IdleTemporal(
        {"closes_at": "2026-11-03T10:05:00+00:00"}, status=WorkflowExecutionStatus.COMPLETED
    )
    response = await _idle(temporal)
    assert response.status_code == 200
    assert response.json() == {"closes_at": ""}
    assert temporal.queries == []


async def test_idle_state_is_empty_when_the_query_fails():
    temporal = _IdleTemporal({}, query_error=WorkflowQueryFailedError("no such query"))
    response = await _idle(temporal)
    assert response.status_code == 200
    assert response.json() == {"closes_at": ""}


async def test_the_idle_banner_polls_the_idle_state():
    async with ui_client() as client:
        script = (await client.get("/action-save-review.js")).text
    assert "idleBanner.textContent =" in script
    assert "window.setInterval(refreshIdle, 10000)" in script


@pytest.mark.parametrize("status", [RPCStatusCode.RESOURCE_EXHAUSTED, RPCStatusCode.UNAVAILABLE])
@pytest.mark.parametrize(
    "path",
    [
        "/api/workflow-status/chat-1",
        "/api/action-author/chat-ready/chat-1",
        "/api/action-author/idle/chat-1",
    ],
)
async def test_temporal_errors_answer_503_without_a_traceback(status, path, caplog):
    app = create_app()
    app.state.binding = object()
    app.state.temporal = _BusyTemporal(status)
    with caplog.at_level(logging.WARNING):
        async with ui_client(app=app) as client:
            response = await client.get(path, headers=SAME_ORIGIN)
    assert response.status_code == 503
    assert response.json() == {
        "error": "temporal_unavailable",
        "message": "consistent query buffer is full",
    }
    assert response.headers["retry-after"] == "2"
    assert response.headers["x-frame-options"] == "DENY"
    assert [(r.levelname, r.getMessage(), r.exc_info) for r in caplog.records] == [
        ("WARNING", f"temporal {status.name} on {path}: consistent query buffer is full", None)
    ]


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
    "body",
    [
        {"namespace": "Bad NS"},
        {"namespace": 1},
        {},
        ["org"],
        {"namespace": "-org"},
        {"namespace": "-x"},
        {"namespace": "a/b"},
        {"namespace": ""},
        {"namespace": "a" * 1001},
    ],
)
async def test_binding_needs_an_organization_name(body):
    async def connect(_namespace):
        raise AssertionError("not called")

    async with ui_client(app=create_app(connect=connect)) as client:
        response = await client.post("/api/action-author/namespace", json=body, headers=SAME_ORIGIN)
    assert response.status_code == 400
    assert response.json()["error"] == "invalid_namespace"


@pytest.mark.parametrize("namespace", ["a" * 80, "организация-1"])
async def test_binding_accepts_canonified_organization_names(namespace):
    connected = []

    async def connect(name):
        connected.append(name)
        raise RuntimeError("down")

    async with ui_client(app=create_app(connect=connect)) as client:
        response = await client.post(
            "/api/action-author/namespace", json={"namespace": namespace}, headers=SAME_ORIGIN
        )
    assert response.status_code == 503
    assert connected == [namespace]


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
    """A closed ActionAuthor run whose recorded events stay readable by query."""

    def __init__(self) -> None:
        self._log: list[str] = []

    @workflow.run
    async def run(self, events: list[str]) -> None:
        self._log = events

    # Same page shape as the runner's ActionAuthor.authoring_events.
    @workflow.query
    def authoring_events(self, from_offset: int = 0) -> dict:
        start = max(0, from_offset)
        page = self._log[start : start + 25]
        return {
            "items": [{"data": data, "offset": start + index} for index, data in enumerate(page)],
            "next_offset": start + len(page),
            "end_offset": len(self._log),
        }


async def _encoded_reply(temporal: Client, text: str) -> str:
    event = AgentEvent(
        agent_id="agent",
        turn_id="turn-1",
        turn_number=1,
        timestamp=1.0,
        event=ReplyDelta(text=text),
    )
    (payload,) = await temporal.data_converter.encode([event])
    return base64.b64encode(payload.SerializeToString()).decode("ascii")


def _sse_events(body: str) -> list[dict]:
    return [
        json.loads(line[len("data: ") :])
        for frame in body.split("\n\n")
        for line in frame.splitlines()
        if line.startswith("data: ")
    ]


@workflow.defn(name="ActionAuthor")
class NeverRunSession:
    """A chat no worker ever picks up, as on a runner whose agent is disabled."""

    @workflow.run
    async def run(self) -> None:
        return None


@pytest.mark.skipif(shutil.which("temporal") is None, reason="needs the temporal CLI")
async def test_chat_ready_waits_for_a_workflow_task():
    async with await WorkflowEnvironment.start_local(
        namespace="org-a",
        dev_server_existing_path=shutil.which("temporal"),
    ) as env:
        target = env.client.service_client.config.target_host

        async def connect(namespace):
            return await Client.connect(target, namespace=namespace, plugins=[AgentHarnessPlugin()])

        org_a = await connect("org-a")
        never_run = f"session-{uuid.uuid4()}-author"
        # Nothing polls this runner queue.
        await org_a.start_workflow(
            NeverRunSession.run, id=never_run, task_queue="disabled-runner-ActionAuthor"
        )
        finished = f"session-{uuid.uuid4()}-author"
        async with Worker(
            org_a,
            task_queue="test-runner",
            workflows=[FinishedSession],
            workflow_runner=UnsandboxedWorkflowRunner(),
        ):
            await org_a.execute_workflow(
                FinishedSession.run, [], id=finished, task_queue="test-runner"
            )

        app = create_app(connect=connect)
        try:
            async with ui_client(app=app) as client:
                bound = await client.post(
                    "/api/action-author/namespace", json={"namespace": "org-a"}, headers=SAME_ORIGIN
                )
                assert bound.status_code == 200
                states = {
                    workflow_id: (
                        await client.get(
                            f"/api/action-author/chat-ready/{workflow_id}", headers=SAME_ORIGIN
                        )
                    ).json()
                    for workflow_id in (never_run, finished, "missing-author")
                }
        finally:
            await _unbind(app)
    assert states == {
        never_run: {"ready": False, "running": True, "execution_status": "RUNNING"},
        finished: {"ready": True, "running": False, "execution_status": "COMPLETED"},
        "missing-author": {"ready": False, "running": False, "execution_status": "NOT_FOUND"},
    }


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
        events = [await _encoded_reply(org_b, text) for text in ("Hello", " world")]
        # Queries on the closed run replay it, so the worker stays up.
        async with Worker(
            org_b,
            task_queue="test-runner",
            workflows=[FinishedSession],
            # The test module imports httpx, which the workflow sandbox refuses.
            workflow_runner=UnsandboxedWorkflowRunner(),
        ):
            await org_b.execute_workflow(
                FinishedSession.run, events, id=session_id, task_queue="test-runner"
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

                    attached = await client.get(
                        "/api/attach", params={"session_id": session_id}, headers=SAME_ORIGIN
                    )
                    assert attached.status_code == 200
                    assert attached.headers["content-type"].startswith("text/event-stream")
                    assert [
                        (e["type"], e["text"], e["resume_offset"])
                        for e in _sse_events(attached.text)
                    ] == [("reply_delta", "Hello", 1), ("reply_delta", " world", 2)]

                    resumed = await client.get(
                        "/api/attach",
                        params={"session_id": session_id, "from_offset": 1},
                        headers=SAME_ORIGIN,
                    )
                    assert resumed.status_code == 200
                    assert [(e["text"], e["resume_offset"]) for e in _sse_events(resumed.text)] == [
                        (" world", 2)
                    ]
            finally:
                await _unbind(app)
