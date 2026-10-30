# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Serve the harness chat UI for Credimi action author sessions.

Environment: TEMPORAL_ADDRESS (through the Temporal client config), ACTION_AUTHOR_UI_PORT
(default 8001). The namespace is not configured: the start page binds the UI to the
organization namespace of the device the user picks (POST /api/action-author/namespace), and
the UI runs that namespace's session-manager worker in process. It never starts agent
workflows: Credimi starts each chat on `<runner_id>-ActionAuthor`.

The UI listens on loopback and answers its own origin only: the Host header must name the
loopback port (no DNS rebinding), state-changing requests and cross-origin API reads must come
from the UI's own origin, and the upstream wildcard CORS middleware is removed.
"""

import asyncio
import base64
import contextlib
import json
import logging
import os
import re
from collections.abc import Awaitable, Callable
from contextlib import asynccontextmanager
from dataclasses import dataclass
from pathlib import Path

import uvicorn
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import (
    FileResponse,
    HTMLResponse,
    JSONResponse,
    RedirectResponse,
    StreamingResponse,
)
from temporal_agent_harness.harness.agent_protocol import AgentEvent
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporal_agent_harness.web import create_agent_harness_app
from temporal_agent_harness.web.agent_starts import agent_starts
from temporal_agent_harness.web.app import (
    _ensure_session_manager_workflow,
    _sse_headers,
    _yield_item,
)
from temporal_agent_harness.web.session_manager import (
    SESSION_MANAGER_TASK_QUEUE,
    AgentDescriptor,
    AgentRegistry,
)
from temporal_agent_harness.web.session_manager_worker import create_session_manager_worker
from temporalio.api.common.v1 import Payload
from temporalio.client import Client, WorkflowExecutionStatus
from temporalio.contrib.workflow_streams import WorkflowStreamClient
from temporalio.envconfig import ClientConfig
from temporalio.worker import Worker

logger = logging.getLogger(__name__)

WORKFLOW_TYPE = "ActionAuthor"
MANAGER_WORKFLOW_ID = "action-author-session-manager"
# The registry needs a queue, but this UI never starts agents; Credimi starts each chat on
# `<runner_id>-ActionAuthor`.
AGENT_TASK_QUEUE = "credimi-runner-ActionAuthor"
NAMESPACE_PATH = "/api/action-author/namespace"
# Canonified Credimi organization names: lowercase letters, digits and `-`.
NAMESPACE_PATTERN = re.compile(r"[a-z0-9][a-z0-9-]{0,62}")

AGENT_LABEL = "Credimi action author"
AGENT_DESCRIPTION = (
    "Authors an action for the wallet version selected on the start page, on a device of an "
    "admin-managed Credimi runner (an Android device or an iOS simulator). Ask with the slot's "
    'input values, e.g. "Write the onboarding action with PIN 123456". After two clean '
    "replays on the runner device, review the accepted YAML and choose Accept or Refuse to save "
    "an unpublished Credimi action. The name defaults to Onboarding unless your request "
    "specifies another name. No form fields or hash copying are required. The model cannot "
    "approve saving."
)

START_PAGE = Path(__file__).with_name("start.html")
SAVE_REVIEW_SCRIPT = Path(__file__).with_name("save-review.js")
NO_STORE = {"Cache-Control": "no-store"}
DEFAULT_PORT = 8001
# Methods that never change state; every other method needs the UI's own Origin.
SAFE_METHODS = {"GET", "HEAD"}
START_PAGE_HEADERS = {
    **NO_STORE,
    "Content-Security-Policy": (
        "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; "
        "connect-src http: https:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
    ),
}
# Keep the packaged UI's new-session buttons on the authenticated setup path.
AUTHENTICATED_START_SCRIPT = """<script>
document.addEventListener('click', (event) => {
    const button = event.target instanceof Element ? event.target.closest('button') : null;
    if (button && (button.getAttribute('aria-label') === 'New session'
        || button.textContent.trim().toLowerCase() === 'start a new session')) {
        event.preventDefault();
        event.stopImmediatePropagation();
        window.location.assign('/start');
    }
}, true);
</script><script type="module" src="/action-save-review.js"></script>"""


async def connect_namespace(namespace: str) -> Client:
    """Connect to the configured Temporal server, in one organization's namespace."""
    return await Client.connect(
        **{**ClientConfig.load_client_connect_config(), "namespace": namespace},
        plugins=[AgentHarnessPlugin()],
    )


@dataclass
class Binding:
    """The namespace the UI serves, with its client and session-manager worker."""

    namespace: str
    client: Client
    worker: Worker
    worker_task: asyncio.Task


async def _stop_worker(worker: Worker, task: asyncio.Task) -> None:
    if not task.done():
        await worker.shutdown()
    with contextlib.suppress(asyncio.CancelledError):
        try:
            await task
        except Exception:
            logger.exception("session-manager worker failed")


async def _body(response) -> bytes:
    return b"".join([chunk async for chunk in response.body_iterator])


async def _closed_history(temporal, workflow_id: str, offset: int):
    """Use read-only queries for closed runs; the upstream stream reader uses updates."""
    handle = temporal.get_workflow_handle(workflow_id)
    while True:
        page = await handle.query("authoring_events", offset)
        for item in page["items"]:
            payload = Payload.FromString(base64.b64decode(item["data"]))
            (event,) = await temporal.data_converter.decode([payload], [AgentEvent])
            yield _yield_item(event, item["offset"] + 1)
        offset = page["next_offset"]
        if offset >= page["end_offset"]:
            return


async def _complete_stream(response, temporal, workflow_id: str, offset: int):
    pending = b""
    async for chunk in response.body_iterator:
        if isinstance(chunk, str):
            chunk = chunk.encode()
        pending += chunk
        while b"\n\n" in pending:
            frame, pending = pending.split(b"\n\n", 1)
            for line in frame.splitlines():
                if line.startswith(b"data: "):
                    data = json.loads(line[6:])
                    offset = max(offset, data.get("resume_offset", offset))
        yield chunk
    # Completing a workflow can race the upstream update-based stream's last poll.
    if (
        await temporal.get_workflow_handle(workflow_id).describe()
    ).status != WorkflowExecutionStatus.RUNNING:
        async for chunk in _closed_history(temporal, workflow_id, offset):
            yield chunk


def _closed_session(execution) -> dict:
    return {
        "workflow_id": execution.id,
        "created_at": execution.start_time.timestamp(),
        "label": AGENT_LABEL,
        "agent_workflow_type": WORKFLOW_TYPE,
        "is_discovered": True,
        "execution_status": execution.status.name,
        "closed": True,
    }


def registry() -> AgentRegistry:
    """The harness registry: the one agent, whose chats Credimi starts."""
    return AgentRegistry(
        agents=[
            AgentDescriptor(
                key="credimi-action-author",
                workflow_type=WORKFLOW_TYPE,
                task_queue=AGENT_TASK_QUEUE,
                label=AGENT_LABEL,
                description=AGENT_DESCRIPTION,
                start=None,
            )
        ]
    )


def _port() -> int:
    return int(os.environ.get("ACTION_AUTHOR_UI_PORT", str(DEFAULT_PORT)))


def _own_hosts(port: int) -> set[str]:
    return {f"127.0.0.1:{port}", f"localhost:{port}"}


def _foreign_request(method: str, path: str, host: str, origin: str | None, port: int) -> str:
    """Why a request is refused (empty: it comes from the UI itself)."""
    host = host.lower()
    if host not in _own_hosts(port):
        return "host"
    if origin is None and method in SAFE_METHODS:
        return ""
    if (method not in SAFE_METHODS or path.startswith("/api/")) and origin != f"http://{host}":
        return "origin"
    return ""


def _reset_state(app) -> None:
    app.state.binding = None
    app.state.binding_lock = asyncio.Lock()
    app.state.agent_starts = agent_starts(registry())


async def _unbind(app) -> None:
    binding = app.state.binding
    if binding is None:
        return
    app.state.binding = None
    await _stop_worker(binding.worker, binding.worker_task)


async def bind(app, namespace: str) -> Binding:
    """Serve `namespace`: connect, run its session-manager worker, ensure its manager."""
    async with app.state.binding_lock:
        current = app.state.binding
        if current is not None and current.namespace == namespace:
            return current
        client = await app.state.connect(namespace)
        worker = create_session_manager_worker(client)
        task = asyncio.create_task(worker.run())
        try:
            manager = await _ensure_session_manager_workflow(
                client,
                registry=registry(),
                manager_workflow_id=MANAGER_WORKFLOW_ID,
                manager_task_queue=SESSION_MANAGER_TASK_QUEUE,
            )
        except BaseException:
            await _stop_worker(worker, task)
            raise
        await _unbind(app)
        app.state.temporal = client
        app.state.manager_handle = manager
        app.state.binding = Binding(namespace, client, worker, task)
        return app.state.binding


async def _namespace_request(app, request) -> JSONResponse:
    if request.method == "GET":
        binding = app.state.binding
        return JSONResponse({"namespace": binding.namespace if binding else None}, headers=NO_STORE)
    try:
        body = await request.json()
    except ValueError:
        body = None
    namespace = body.get("namespace") if isinstance(body, dict) else None
    if not isinstance(namespace, str) or not NAMESPACE_PATTERN.fullmatch(namespace):
        return JSONResponse(
            status_code=400,
            content={
                "error": "invalid_namespace",
                "message": "namespace must be a Credimi organization name",
            },
        )
    try:
        binding = await bind(app, namespace)
    except Exception as exc:
        logger.exception("bind namespace %s", namespace)
        return JSONResponse(
            status_code=503,
            content={"error": "temporal_unavailable", "message": str(exc)},
        )
    return JSONResponse({"namespace": binding.namespace}, headers=NO_STORE)


def create_app(*, connect: Callable[[str], Awaitable[Client]] = connect_namespace):
    app = create_agent_harness_app(registry=registry, manager_workflow_id=MANAGER_WORKFLOW_ID)
    # Upstream allows every origin; the UI only serves its own pages.
    app.user_middleware = [m for m in app.user_middleware if m.cls is not CORSMiddleware]
    app.state.connect = connect
    _reset_state(app)

    # Replaces the upstream startup, which connects to one fixed namespace.
    @asynccontextmanager
    async def _lifespan(app):
        _reset_state(app)
        try:
            yield
        finally:
            await _unbind(app)

    app.router.lifespan_context = _lifespan
    port = _port()

    @app.middleware("http")
    async def authenticated_start_only(request, call_next):
        method, path = request.method, request.url.path
        if path == NAMESPACE_PATH and method in {"GET", "POST"}:
            return await _namespace_request(app, request)
        if method == "GET" and path == "/action-save-review.js":
            return FileResponse(SAVE_REVIEW_SCRIPT, media_type="text/javascript", headers=NO_STORE)
        if method == "GET" and path == "/start":
            return FileResponse(START_PAGE, headers=START_PAGE_HEADERS)
        if method == "GET" and path == "/" and not request.query_params:
            return RedirectResponse("/start")
        if app.state.binding is None:
            if path.startswith("/api/"):
                return JSONResponse(
                    status_code=409,
                    content={
                        "error": "namespace_not_selected",
                        "message": "Open /start and choose a device first.",
                    },
                )
            if method == "GET" and path == "/":
                return RedirectResponse("/start?unbound=1")
        if method == "GET" and path == "/":
            response = await call_next(request)
            if response.status_code != 200 or not response.headers.get(
                "content-type", ""
            ).startswith("text/html"):
                return response
            html = (await _body(response)).decode()
            return HTMLResponse(
                html.replace("</head>", AUTHENTICATED_START_SCRIPT + "</head>", 1),
                headers=NO_STORE,
            )
        if method == "POST" and path.rstrip("/") == "/api/sessions":
            return JSONResponse(
                status_code=403,
                content={
                    "error": "authenticated_start_required",
                    "message": "Start sessions through authenticated POST "
                    "/api/action-author/sessions on Credimi, then select the session here.",
                },
            )
        if method == "GET" and path == "/api/sessions":
            response = await call_next(request)
            if response.status_code != 200:
                return response
            sessions = json.loads(await _body(response))
            known = {session["workflow_id"] for session in sessions}
            # Direct Credimi starts are not in the harness manager's registry.
            # Upstream discovers only running workflows, so add recent closed runs
            # (newest first, one entry per workflow ID).
            async for execution in app.state.temporal.list_workflows(
                query=f"WorkflowType='{WORKFLOW_TYPE}' AND ExecutionStatus!='Running'",
                limit=200,
            ):
                if execution.id not in known:
                    known.add(execution.id)
                    sessions.append(_closed_session(execution))
            return JSONResponse(sessions, headers=NO_STORE)

        session_id, offset = None, 0
        if method == "GET" and path == "/api/attach":
            session_id = request.query_params.get("session_id")
            try:
                offset = int(request.query_params.get("from_offset", "0"))
            except ValueError:
                offset = -1
            if not session_id or offset < 0:
                # Upstream validation answers malformed requests.
                return await call_next(request)
            temporal = app.state.temporal
            state = await temporal.get_workflow_handle(session_id).describe()
            if state.status != WorkflowExecutionStatus.RUNNING:
                return StreamingResponse(
                    _closed_history(temporal, session_id, offset),
                    media_type="text/event-stream",
                    headers=_sse_headers(),
                )
        elif method == "POST" and path == "/api/chat":
            try:
                body = await request.json()
            except ValueError:
                return await call_next(request)
            if isinstance(body, dict) and isinstance(body.get("session_id"), str):
                session_id = body["session_id"]
                offset = await WorkflowStreamClient.create(
                    app.state.temporal, session_id
                ).get_offset()
        if not session_id:
            return await call_next(request)
        response = await call_next(request)
        if response.status_code != 200 or not response.headers.get("content-type", "").startswith(
            "text/event-stream"
        ):
            return response
        return StreamingResponse(
            _complete_stream(response, app.state.temporal, session_id, offset),
            media_type="text/event-stream",
            headers=_sse_headers(),
        )

    # Registered last, so it runs first.
    @app.middleware("http")
    async def own_origin_only(request, call_next):
        refused = _foreign_request(
            request.method,
            request.url.path,
            request.headers.get("host", ""),
            request.headers.get("origin"),
            port,
        )
        if refused == "host":
            return JSONResponse(
                status_code=421,
                content={"error": "misdirected_request", "message": "unknown Host"},
            )
        if refused:
            return JSONResponse(
                status_code=403,
                content={"error": "cross_origin_request", "message": "foreign Origin"},
            )
        return await call_next(request)

    return app


def main() -> None:
    logging.basicConfig(level=logging.INFO)
    uvicorn.run(create_app(), host="127.0.0.1", port=_port())


if __name__ == "__main__":
    main()
