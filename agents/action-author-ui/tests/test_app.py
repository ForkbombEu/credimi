# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

import asyncio
import base64
import importlib
import json
import logging
import os
import shutil
import time
import uuid
from datetime import UTC, datetime
from pathlib import Path

import httpx
import jwt
import pytest
from cryptography.hazmat.primitives.asymmetric import ec
from temporal_agent_harness.harness.agent_protocol import AgentEvent, ReplyDelta
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporal_agent_harness.web.task_queue_status import RegistryWorkers, TaskQueueWorkers
from temporalio import workflow
from temporalio.client import Client, WorkflowExecutionStatus
from temporalio.service import RPCError, RPCStatusCode
from temporalio.testing import WorkflowEnvironment
from temporalio.worker import UnsandboxedWorkflowRunner, Worker

from credimi_action_author_ui import app as ui_app
from credimi_action_author_ui.app import HTML_CSP, create_app, registry

UI = "http://action_author_ui:8080"
CHAT = "ActionAuthorSession-testrunner00001-0123abcd-0000-4000-8000-00000000abcd-author"
STARTED = datetime(2026, 11, 3, 10, 0, tzinfo=UTC)
# Credimi's signing key, published in the test jwks.json; FOREIGN_KEY is not published.
SIGNING_KEY = ec.generate_private_key(ec.SECP256R1())
FOREIGN_KEY = ec.generate_private_key(ec.SECP256R1())
KID = "credimi-key-1"


def _jwk(key: ec.EllipticCurvePrivateKey, kid: str) -> dict:
    jwk = jwt.algorithms.ECAlgorithm.to_jwk(key.public_key(), as_dict=True)
    return {**jwk, "alg": "ES256", "use": "sig", "kid": kid}


def _write_jwks(directory: Path, *keys: tuple[ec.EllipticCurvePrivateKey, str]) -> None:
    """Replace jwks.json atomically, as Credimi does (temporalauth.WriteJWKS)."""
    tmp = directory / ".jwks-test.json"
    tmp.write_text(json.dumps({"keys": [_jwk(key, kid) for key, kid in keys]}))
    os.replace(tmp, directory / "jwks.json")


@pytest.fixture(autouse=True)
def jwks_dir(tmp_path, monkeypatch) -> Path:
    _write_jwks(tmp_path, (SIGNING_KEY, KID))
    monkeypatch.setenv("CREDIMI_TEMPORAL_JWKS_DIR", str(tmp_path))
    return tmp_path


def _token(
    namespace: str = "org-a",
    *,
    key: ec.EllipticCurvePrivateKey = SIGNING_KEY,
    kid: str = KID,
    expires_in: int = 900,
    permissions: list[str] | None = None,
) -> str:
    """A token as temporalauth.NamespaceToken signs it."""
    now = int(time.time())
    claims = {
        "iss": "credimi",
        "sub": "credimi-action-author-ui",
        "permissions": [f"{namespace}:write"] if permissions is None else permissions,
        "iat": now,
        "exp": now + expires_in,
    }
    return jwt.encode(claims, key, algorithm="ES256", headers={"kid": kid})


def _scope(namespace: str = "org-a", chat: str = CHAT, token: str | None = None) -> dict[str, str]:
    return {
        "X-Credimi-Namespace": namespace,
        "X-Credimi-Action-Author-Chat": chat,
        "X-Credimi-Temporal-Token": _token(namespace) if token is None else token,
    }


class _ChatTemporal:
    """A namespace whose only workflow is the chat ``workflow_id`` in ``status``, or none."""

    def __init__(
        self,
        status: WorkflowExecutionStatus | None = WorkflowExecutionStatus.RUNNING,
        workflow_id: str = CHAT,
        task_queue: str = "runner-1-ActionAuthor",
    ) -> None:
        self.status = status
        self.workflow_id = workflow_id
        self.task_queue = task_queue
        self.described: list[str] = []

    def get_workflow_handle(self, workflow_id):
        temporal = self

        class Description:
            id = workflow_id
            status = temporal.status
            start_time = STARTED
            task_queue = temporal.task_queue

        class Handle:
            async def describe(self):
                temporal.described.append(workflow_id)
                if temporal.status is None or workflow_id != temporal.workflow_id:
                    raise RPCError("workflow not found", RPCStatusCode.NOT_FOUND, b"")
                return Description()

            async def fetch_history_events(self, **_kwargs):
                return
                yield

        return Handle()


class _BusyTemporal:
    def __init__(self, status: RPCStatusCode) -> None:
        self._status = status

    def get_workflow_handle(self, _workflow_id):
        status = self._status

        class Handle:
            async def describe(self):
                raise RPCError("consistent query buffer is full", status, b"")

        return Handle()


def _connecting(temporal) -> tuple[list[str], object]:
    connected: list[str] = []

    async def connect(namespace, _token):
        connected.append(namespace)
        return temporal

    return connected, connect


def ui_client(app=None, headers: dict[str, str] | None = None) -> httpx.AsyncClient:
    if app is None:
        app = create_app(connect=_connecting(_ChatTemporal())[1])
    return httpx.AsyncClient(
        transport=httpx.ASGITransport(app=app),
        base_url=UI,
        headers=_scope() if headers is None else headers,
    )


@pytest.mark.parametrize(
    "headers",
    [
        {},
        {"X-Credimi-Namespace": "org-a"},
        {"X-Credimi-Action-Author-Chat": CHAT},
        {"X-Credimi-Namespace": "org-a", "X-Credimi-Action-Author-Chat": CHAT},
        _scope(token=""),
        _scope(namespace=""),
        _scope(namespace="-org"),
        _scope(namespace="org/a"),
        _scope(chat=""),
        _scope(chat="other-author"),
        _scope(chat=CHAT.removesuffix("-author")),
        _scope(chat=CHAT.upper()),
    ],
)
async def test_requests_without_scope_are_refused(headers):
    connected, connect = _connecting(_ChatTemporal())
    async with ui_client(create_app(connect=connect), headers=headers) as client:
        response = await client.get("/api/sessions")
    assert (response.status_code, response.json()) == (400, {"error": "missing_scope"})
    assert connected == []


async def test_each_namespace_gets_its_own_client():
    connected, connect = _connecting(_ChatTemporal())
    app = create_app(connect=connect)
    async with ui_client(app) as client:
        for namespace in ("org-a", "org-a", "org-b"):
            response = await client.get("/action-embed.css", headers=_scope(namespace))
            assert response.status_code == 200
    assert connected == ["org-a", "org-b"]


async def test_each_request_sets_its_token_on_the_client():
    clients: dict[str, _ChatTemporal] = {}
    connected: list[tuple[str, str]] = []

    async def connect(namespace, token):
        connected.append((namespace, token))
        clients[namespace] = _ChatTemporal()
        return clients[namespace]

    first_token = _token("runner-a")
    renewed_token = _token("runner-a", expires_in=1800)
    async with ui_client(create_app(connect=connect)) as client:
        first = await client.get("/api/sessions", headers=_scope("runner-a", token=first_token))
        assert first.status_code == 200
        assert clients["runner-a"].api_key == first_token
        # A renewed token reaches the same client, without reconnecting.
        again = await client.get("/api/sessions", headers=_scope("runner-a", token=renewed_token))
        assert again.status_code == 200
    assert connected == [("runner-a", first_token)]
    assert clients["runner-a"].api_key == renewed_token


def _unsigned_token(namespace: str = "org-a") -> str:
    claims = {"permissions": [f"{namespace}:write"], "exp": int(time.time()) + 900}
    return jwt.encode(claims, None, algorithm="none", headers={"kid": KID})


@pytest.mark.parametrize(
    "token",
    [
        pytest.param(_token("org-b"), id="other namespace"),
        pytest.param(_token(permissions=["org-a:read"]), id="other role"),
        pytest.param(_token(permissions=["org-a-x:write"]), id="namespace prefix"),
        pytest.param(_token(expires_in=-60), id="expired"),
        pytest.param(_token(key=FOREIGN_KEY), id="foreign key, published kid"),
        pytest.param(_token(key=FOREIGN_KEY, kid="foreign"), id="unknown kid"),
        pytest.param(_unsigned_token(), id="unsigned"),
        pytest.param(_token()[:-4] + "AAAA", id="bad signature"),
        pytest.param("token-1", id="not a jwt"),
    ],
)
async def test_unverified_tokens_are_refused(token):
    connected, connect = _connecting(_ChatTemporal())
    async with ui_client(create_app(connect=connect), headers=_scope("org-a", token=token)) as c:
        response = await c.get("/api/sessions")
    assert (response.status_code, response.json()) == (403, {"error": "invalid_token"})
    assert connected == []


async def test_a_token_cannot_steer_another_namespace_client():
    clients: dict[str, _ChatTemporal] = {}

    async def connect(namespace, _token):
        clients[namespace] = _ChatTemporal()
        return clients[namespace]

    org_a_token = _token("org-a")
    async with ui_client(create_app(connect=connect)) as client:
        assert (await client.get("/api/sessions", headers=_scope(token=org_a_token))).is_success
        refused = await client.get("/api/sessions", headers=_scope(token=_token("org-b")))
    assert refused.status_code == 403
    assert clients["org-a"].api_key == org_a_token


async def test_rotated_keys_are_reloaded(jwks_dir):
    rotated = _token(key=FOREIGN_KEY, kid="credimi-key-2")
    async with ui_client() as client:
        before = await client.get("/api/sessions", headers=_scope(token=rotated))
        _write_jwks(jwks_dir, (FOREIGN_KEY, "credimi-key-2"), (SIGNING_KEY, KID))
        after = await client.get("/api/sessions", headers=_scope(token=rotated))
        old = await client.get("/api/sessions", headers=_scope())
        # A dropped key no longer verifies.
        _write_jwks(jwks_dir, (FOREIGN_KEY, "credimi-key-2"))
        dropped = await client.get("/api/sessions", headers=_scope())
    assert before.status_code == 403
    assert after.status_code == 200
    assert old.status_code == 200
    assert dropped.status_code == 403


@pytest.mark.parametrize("content", [None, "", "{", '{"keys": []}'])
async def test_without_readable_keys_every_token_is_refused(jwks_dir, content):
    if content is None:
        (jwks_dir / "jwks.json").unlink()
    else:
        (jwks_dir / "jwks.json").write_text(content)
    async with ui_client(create_app(connect=_connecting(_ChatTemporal())[1])) as client:
        response = await client.get("/api/sessions")
    assert (response.status_code, response.json()) == (403, {"error": "invalid_token"})


def test_the_jwks_dir_is_required(monkeypatch):
    monkeypatch.delenv("CREDIMI_TEMPORAL_JWKS_DIR")
    with pytest.raises(RuntimeError, match="CREDIMI_TEMPORAL_JWKS_DIR is required"):
        create_app()


async def test_connect_namespace_sends_the_token_with_explicit_tls(monkeypatch):
    calls: list[dict] = []

    async def fake_connect(**kwargs):
        calls.append(kwargs)
        return "client"

    monkeypatch.setattr(ui_app.Client, "connect", fake_connect)
    monkeypatch.setenv("TEMPORAL_ADDRESS", "temporal:7233")
    for tls, expected in (("false", False), ("", False), ("TRUE", True)):
        monkeypatch.setenv("TEMPORAL_TLS", tls)
        assert await ui_app.connect_namespace("runner-a", "token-1") == "client"
        assert calls[-1]["namespace"] == "runner-a"
        assert calls[-1]["api_key"] == "token-1"
        assert calls[-1]["tls"] is expected


async def test_connect_failures_are_retried():
    attempts: list[str] = []

    async def connect(namespace, _token):
        attempts.append(namespace)
        if len(attempts) == 1:
            raise RuntimeError("temporal is down")
        return _ChatTemporal()

    async with ui_client(create_app(connect=connect)) as client:
        failed = await client.get("/api/sessions")
        served = await client.get("/api/sessions")
    assert (failed.status_code, failed.json()) == (503, {"error": "temporal_unavailable"})
    assert served.status_code == 200
    assert attempts == ["org-a", "org-a"]


async def test_a_slow_namespace_does_not_block_another():
    released = asyncio.Event()

    async def connect(namespace, _token):
        if namespace == "org-a":
            await released.wait()
        return _ChatTemporal()

    async with ui_client(create_app(connect=connect)) as client:
        slow = asyncio.create_task(client.get("/api/sessions", headers=_scope("org-a")))
        fast = await asyncio.wait_for(client.get("/api/sessions", headers=_scope("org-b")), 5)
        assert fast.status_code == 200
        assert not slow.done()
        released.set()
        assert (await asyncio.wait_for(slow, 5)).status_code == 200


async def test_requests_for_another_host_are_refused():
    async with ui_client() as client:
        response = await client.get("/api/sessions", headers={"Host": "evil.example"})
    assert (response.status_code, response.json()) == (421, {"error": "misdirected_request"})


async def test_an_empty_allowed_host_does_not_match_an_empty_host(monkeypatch):
    monkeypatch.setenv("ACTION_AUTHOR_UI_ALLOWED_HOSTS", "action_author_ui,")
    importlib.reload(ui_app)
    try:
        assert "" not in ui_app.ALLOWED_HOSTS
        async with ui_client(ui_app.create_app(connect=_connecting(_ChatTemporal())[1])) as client:
            response = await client.get("/api/sessions", headers={"Host": ""})
    finally:
        monkeypatch.delenv("ACTION_AUTHOR_UI_ALLOWED_HOSTS")
        importlib.reload(ui_app)
    assert (response.status_code, response.json()) == (421, {"error": "misdirected_request"})


OTHER = CHAT.replace("abcd-author", "dcba-author")


@pytest.mark.parametrize(
    ("method", "path", "kwargs", "error"),
    [
        ("GET", f"/api/status/{OTHER}", {}, "other_chat"),
        ("GET", f"/api/workflow-status/{OTHER}", {}, "other_chat"),
        ("GET", "/api/attach", {"params": {"session_id": OTHER}}, "other_chat"),
        ("POST", "/api/messages", {"json": {"session_id": OTHER, "text": "hi"}}, "other_chat"),
        ("POST", "/api/chat", {"json": {"session_id": OTHER, "message": "hi"}}, "other_chat"),
        ("POST", "/api/chat", {"json": [CHAT]}, "other_chat"),
        ("POST", "/api/chat", {"content": b"{"}, "other_chat"),
        ("POST", f"/api/sessions/{OTHER}/close", {}, "other_chat"),
        ("POST", "/api/approve", {"json": {"session_id": CHAT}}, "approvals_disabled"),
        ("POST", "/api/files/view", {"json": {"path": "/etc/passwd"}}, "files_disabled"),
        ("GET", "/api/files/x", {}, "files_disabled"),
    ],
)
async def test_requests_for_another_chat_are_refused(method, path, kwargs, error):
    async with ui_client() as client:
        response = await client.request(method, path, **kwargs)
    assert (response.status_code, response.json()) == (403, {"error": error})


@pytest.mark.parametrize("method", ["GET", "POST", "PUT"])
async def test_reviews_are_never_settled_through_the_ui(method):
    async with ui_client() as client:
        response = await client.request(
            method,
            "/api/callback-result",
            json={"session_id": CHAT, "tool_id": "t", "result": True},
        )
    assert response.status_code == 403
    assert response.json() == {
        "error": "review_through_credimi",
        "message": "Save reviews are sent to Credimi.",
    }


def test_the_ui_port_follows_its_setting(monkeypatch):
    served = []
    monkeypatch.setattr(ui_app.uvicorn, "run", lambda app, **kwargs: served.append(kwargs))
    monkeypatch.setenv("ACTION_AUTHOR_UI_PORT", "9123")
    ui_app.main()
    monkeypatch.delenv("ACTION_AUTHOR_UI_PORT")
    ui_app.main()
    assert served == [{"host": "0.0.0.0", "port": 9123}, {"host": "0.0.0.0", "port": 8080}]


async def test_harness_cannot_start_sessions_with_caller_supplied_actor():
    async with ui_client() as client:
        response = await client.post(
            "/api/sessions",
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


async def test_chat_page_is_embeddable_by_credimi_only():
    async with ui_client() as client:
        page = await client.get("/", params={"s": CHAT})
        css = await client.get("/action-embed.css")
    assert page.status_code == 200
    assert page.headers["x-frame-options"] == "SAMEORIGIN"
    assert page.headers["content-security-policy"] == HTML_CSP
    assert "frame-ancestors 'self'" in HTML_CSP
    assert page.headers["cache-control"] == "no-store"
    assert '<link rel="stylesheet" href="./action-embed.css"></head>' in page.text
    assert css.status_code == 200
    assert css.headers["content-type"].startswith("text/css")
    assert css.headers["x-frame-options"] == "SAMEORIGIN"
    assert "button.session-new, button.start-new, section.session-manager" in css.text
    # Runner namespaces are not open to users in the Temporal UI.
    assert "a.session-temporal { display: none !important; }" in css.text
    assert "color-scheme: light" in css.text
    assert ".svelte-flow.dark" in css.text


def test_registry_uses_the_chat_task_queue():
    (agent,) = registry("r-ActionAuthor").agents
    assert (agent.workflow_type, agent.task_queue, agent.start) == (
        "ActionAuthor",
        "r-ActionAuthor",
        None,
    )


async def test_sessions_lists_only_the_scoped_chat():
    running = _ChatTemporal()
    async with ui_client(create_app(connect=_connecting(running)[1])) as client:
        response = await client.get("/api/sessions")
    assert response.status_code == 200
    assert response.headers["cache-control"] == "no-store"
    assert response.json() == [
        {
            "workflow_id": CHAT,
            "created_at": STARTED.timestamp(),
            "label": "Credimi action author",
            "agent_workflow_type": "ActionAuthor",
            "is_discovered": True,
            "execution_status": "RUNNING",
            "closed": False,
        }
    ]

    closed = _ChatTemporal(WorkflowExecutionStatus.COMPLETED)
    async with ui_client(create_app(connect=_connecting(closed)[1])) as client:
        response = await client.get("/api/sessions")
    assert [(s["workflow_id"], s["execution_status"], s["closed"]) for s in response.json()] == [
        (CHAT, "COMPLETED", True)
    ]

    async with ui_client(create_app(connect=_connecting(_ChatTemporal(None))[1])) as client:
        response = await client.get("/api/sessions")
    assert (response.status_code, response.json()) == (200, [])


async def test_agents_lists_the_chat_task_queue(monkeypatch):
    described: list[list[str]] = []

    async def describe_workers(_temporal, task_queues):
        queues = list(task_queues)
        described.append(queues)
        return RegistryWorkers(
            by_task_queue={
                queue: TaskQueueWorkers(task_queue=queue, status="ready", poller_count=1)
                for queue in queues
            }
        )

    monkeypatch.setattr(ui_app, "describe_task_queue_workers", describe_workers)
    temporal = _ChatTemporal(task_queue="runner-7-ActionAuthor")
    async with ui_client(create_app(connect=_connecting(temporal)[1])) as client:
        response = await client.get("/api/agents")
    assert response.status_code == 200
    assert response.headers["cache-control"] == "no-store"
    (agent,) = response.json()["agents"]
    assert (agent["workflow_type"], agent["task_queue"], agent["init_data"]) == (
        "ActionAuthor",
        "runner-7-ActionAuthor",
        None,
    )
    assert (agent["worker"]["status"], agent["worker"]["poller_count"]) == ("ready", 1)
    assert described == [["runner-7-ActionAuthor"]]
    assert temporal.described == [CHAT]


@pytest.mark.parametrize("status", [RPCStatusCode.RESOURCE_EXHAUSTED, RPCStatusCode.UNAVAILABLE])
@pytest.mark.parametrize("path", [f"/api/workflow-status/{CHAT}", "/api/sessions", "/api/agents"])
async def test_temporal_errors_answer_503_without_a_traceback(status, path, caplog):
    app = create_app(connect=_connecting(_BusyTemporal(status))[1])
    with caplog.at_level(logging.WARNING):
        async with ui_client(app) as client:
            response = await client.get(path)
    assert response.status_code == 503
    assert response.json() == {
        "error": "temporal_unavailable",
        "message": "consistent query buffer is full",
    }
    assert response.headers["retry-after"] == "2"
    assert response.headers["x-frame-options"] == "SAMEORIGIN"
    assert [(r.levelname, r.getMessage(), r.exc_info) for r in caplog.records] == [
        ("WARNING", f"temporal {status.name} on {path}: consistent query buffer is full", None)
    ]


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


@pytest.mark.skipif(shutil.which("temporal") is None, reason="needs the temporal CLI")
async def test_closed_chat_replays_its_events():
    async with await WorkflowEnvironment.start_local(
        namespace="org-a",
        dev_server_extra_args=["--namespace", "org-b"],
        dev_server_existing_path=shutil.which("temporal"),
    ) as env:
        target = env.client.service_client.config.target_host

        async def connect(namespace, _token=""):
            return await Client.connect(target, namespace=namespace, plugins=[AgentHarnessPlugin()])

        org_b = await connect("org-b")
        chat = f"ActionAuthorSession-testrunner00001-{uuid.uuid4()}-author"
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
                FinishedSession.run, events, id=chat, task_queue="test-runner"
            )

            async with ui_client(create_app(connect=connect), _scope("org-b", chat)) as client:
                elsewhere = await client.get("/api/sessions", headers=_scope("org-a", chat))
                assert (elsewhere.status_code, elsewhere.json()) == (200, [])

                sessions = await client.get("/api/sessions")
                assert sessions.status_code == 200
                assert [(s["workflow_id"], s["closed"]) for s in sessions.json()] == [(chat, True)]

                attached = await client.get("/api/attach", params={"session_id": chat})
                assert attached.status_code == 200
                assert attached.headers["content-type"].startswith("text/event-stream")
                assert [
                    (e["type"], e["text"], e["resume_offset"]) for e in _sse_events(attached.text)
                ] == [("reply_delta", "Hello", 1), ("reply_delta", " world", 2)]

                resumed = await client.get(
                    "/api/attach", params={"session_id": chat, "from_offset": 1}
                )
                assert resumed.status_code == 200
                assert [(e["text"], e["resume_offset"]) for e in _sse_events(resumed.text)] == [
                    (" world", 2)
                ]
