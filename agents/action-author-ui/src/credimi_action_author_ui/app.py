# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Serve the harness chat UI embedded in Credimi action author session pages.

An internal service: only Credimi reaches it, through the session-scoped proxy at
``/action-author-ui/<session id>`` (pkg/internal/actionauthorui). The proxy authenticates the
user, allows only the routes of that session's chat and names the scope of every request in two
headers: ``X-Credimi-Namespace`` (the caller's organization namespace) and
``X-Credimi-Action-Author-Chat`` (the chat workflow ID). A request without both is refused.

The service keeps one harness app, with its own Temporal client, per namespace. It never starts
agent workflows: Credimi starts each chat on ``<runner_id>-ActionAuthor``.

Environment: TEMPORAL_ADDRESS and the other TEMPORAL_* settings read by the Temporal client
config (the namespace comes from each request), ACTION_AUTHOR_UI_PORT (default 8080).
"""

import asyncio
import base64
import json
import logging
import os
import re
from collections.abc import Awaitable, Callable
from dataclasses import asdict
from pathlib import Path

import uvicorn
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, HTMLResponse, JSONResponse, StreamingResponse
from starlette.applications import Starlette
from starlette.middleware import Middleware
from starlette.types import ASGIApp, Receive, Scope, Send
from temporal_agent_harness.harness.agent_protocol import AgentEvent
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporal_agent_harness.web import create_agent_harness_app
from temporal_agent_harness.web.app import (
    _sessions_with_execution_state,
    _sse_headers,
    _yield_item,
)
from temporal_agent_harness.web.session_manager import AgentDescriptor, AgentRegistry, Session
from temporal_agent_harness.web.task_queue_status import describe_task_queue_workers
from temporalio.api.common.v1 import Payload
from temporalio.client import Client, WorkflowExecutionStatus
from temporalio.contrib.workflow_streams import WorkflowStreamClient
from temporalio.envconfig import ClientConfig
from temporalio.service import RPCError, RPCStatusCode

logger = logging.getLogger(__name__)

WORKFLOW_TYPE = "ActionAuthor"
MANAGER_WORKFLOW_ID = "action-author-session-manager"
# The registry needs a queue; the agents route describes the chat's own queue,
# `<runner_id>-ActionAuthor`, where Credimi starts it.
AGENT_TASK_QUEUE = "credimi-runner-ActionAuthor"
CALLBACK_RESULT_PATH = "/api/callback-result"
NAMESPACE_HEADER = "x-credimi-namespace"
CHAT_HEADER = "x-credimi-action-author-chat"
# Names Credimi's canonify produces for organizations (pkg/internal/canonify): Unicode
# letters and digits joined by `-`, never leading with `-`; at most 1000 characters.
NAMESPACE_PATTERN = re.compile(r"[^\W_](?:[^\W_]|-){0,999}")
# `<session id>-author`, the chat of a session Credimi started.
CHAT_PATTERN = re.compile(
    r"ActionAuthorSession-[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}-author"
)

AGENT_LABEL = "Credimi action author"
AGENT_DESCRIPTION = (
    "Authors actions for the wallet version selected in Credimi, on a device of an "
    "admin-managed Credimi runner (an Android device or an iOS simulator). Describe the action "
    "in the chat with its input values and references (a Credimi URL or identifier, or a "
    'deeplink), e.g. "Write the onboarding action with PIN 123456". The agent works out the '
    "plan (category, outcome, starting actions and parameter sources) and asks only when "
    "something is unclear. After a clean replay on the runner device, review the YAML and the "
    "plan on the Credimi session page and choose Accept or Refuse to save an unpublished Credimi "
    "action; then ask for the next action in the same chat. Credimi signs the save: the model "
    "cannot approve saving."
)

# Hides what Credimi does itself: starting and listing sessions, and the save review.
EMBED_CSS = Path(__file__).with_name("embed.css")
EMBED_CSS_PATH = "/action-embed.css"
# Relative: the page sits under Credimi's /action-author-ui/<session id>/.
EMBED_CSS_LINK = f'<link rel="stylesheet" href=".{EMBED_CSS_PATH}">'
NO_STORE = {"Cache-Control": "no-store"}
# How long a client waits before retrying after a Temporal error.
TEMPORAL_RETRY_AFTER = "2"
DEFAULT_PORT = 8080
# Only Credimi's own pages may frame the UI.
FRAME_ANCESTORS_CSP = "frame-ancestors 'self'"
# Every HTML page: the packaged chat page loads from the UI itself (through the proxy) and styles
# itself inline. The packaged index.html has no inline script, so script-src needs no hash.
HTML_CSP = (
    "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "
    "img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; "
    f"object-src 'none'; base-uri 'none'; {FRAME_ANCESTORS_CSP}"
)


async def connect_namespace(namespace: str) -> Client:
    """Connect to the configured Temporal server, in one organization's namespace."""
    return await Client.connect(
        **{**ClientConfig.load_client_connect_config(), "namespace": namespace},
        plugins=[AgentHarnessPlugin()],
    )


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


def registry(task_queue: str = AGENT_TASK_QUEUE) -> AgentRegistry:
    """The harness registry: the one agent, whose chats Credimi starts on ``task_queue``."""
    return AgentRegistry(
        agents=[
            AgentDescriptor(
                key="credimi-action-author",
                workflow_type=WORKFLOW_TYPE,
                task_queue=task_queue,
                label=AGENT_LABEL,
                description=AGENT_DESCRIPTION,
                start=None,
            )
        ]
    )


def _port() -> int:
    return int(os.environ.get("ACTION_AUTHOR_UI_PORT", str(DEFAULT_PORT)))


def _frame_for_credimi(response) -> None:
    """Let only Credimi's pages frame the UI, and give every HTML page HTML_CSP."""
    response.headers["X-Frame-Options"] = "SAMEORIGIN"
    if response.headers.get("content-type", "").startswith("text/html"):
        response.headers["Content-Security-Policy"] = HTML_CSP
        return
    policy = response.headers.get("Content-Security-Policy")
    if not policy:
        response.headers["Content-Security-Policy"] = FRAME_ANCESTORS_CSP
    elif "frame-ancestors" not in policy:
        response.headers["Content-Security-Policy"] = f"{policy}; {FRAME_ANCESTORS_CSP}"


def _temporal_error(exc: RPCError, path: str) -> JSONResponse:
    """Answer a Temporal service error (e.g. a full query buffer) without a traceback."""
    logger.warning("temporal %s on %s: %s", exc.status.name, path, exc.message)
    return JSONResponse(
        status_code=503,
        content={"error": "temporal_unavailable", "message": exc.message},
        headers={**NO_STORE, "Retry-After": TEMPORAL_RETRY_AFTER},
    )


async def _describe_chat(temporal, chat: str):
    """The chat's execution, or None when Temporal does not know it."""
    try:
        return await temporal.get_workflow_handle(chat).describe()
    except RPCError as exc:
        if exc.status != RPCStatusCode.NOT_FOUND:
            raise
        return None


async def _agents(temporal, chat: str) -> JSONResponse:
    """The agent of the scoped chat, with whether a worker polls the chat's task queue."""
    execution = await _describe_chat(temporal, chat)
    agents = []
    if execution is not None:
        task_queue = execution.task_queue
        workers = await describe_task_queue_workers(temporal, [task_queue])
        (descriptor,) = registry(task_queue).agents
        agents.append(
            {
                **asdict(descriptor),
                "init_data": None,
                "worker": asdict(workers.for_task_queue(task_queue)),
            }
        )
    return JSONResponse({"agents": agents}, headers=NO_STORE)


async def _sessions(temporal, chat: str) -> JSONResponse:
    """Only the scoped chat: running with its upstream state, closed, or none yet."""
    execution = await _describe_chat(temporal, chat)
    if execution is None:
        sessions = []
    elif execution.status == WorkflowExecutionStatus.RUNNING:
        sessions = await _sessions_with_execution_state(
            temporal,
            [
                Session(
                    workflow_id=chat,
                    created_at=execution.start_time.timestamp(),
                    label=AGENT_LABEL,
                    agent_workflow_type=WORKFLOW_TYPE,
                    is_discovered=True,
                )
            ],
        )
    else:
        sessions = [_closed_session(execution)]
    return JSONResponse(sessions, headers=NO_STORE)


def _create_harness(temporal) -> FastAPI:
    """The upstream harness app on ``temporal``, serving the chat each request is scoped to.

    Its lifespan never runs, so nothing reads its session manager or agent starts: the routes
    that do are answered here.
    """
    harness = create_agent_harness_app(registry=registry, manager_workflow_id=MANAGER_WORKFLOW_ID)
    # Upstream allows every origin; only Credimi's proxy reaches the service.
    harness.user_middleware = [m for m in harness.user_middleware if m.cls is not CORSMiddleware]
    harness.state.temporal = temporal

    async def dispatch(request, call_next):
        method, path = request.method, request.url.path
        chat = request.scope["state"]["chat"]
        if method == "GET" and path == EMBED_CSS_PATH:
            return FileResponse(EMBED_CSS, media_type="text/css", headers=NO_STORE)
        if method == "GET" and path == "/":
            response = await call_next(request)
            if response.status_code != 200 or not response.headers.get(
                "content-type", ""
            ).startswith("text/html"):
                return response
            html = (await _body(response)).decode()
            return HTMLResponse(
                html.replace("</head>", EMBED_CSS_LINK + "</head>", 1), headers=NO_STORE
            )
        if method == "POST" and path.rstrip("/") == "/api/sessions":
            return JSONResponse(
                status_code=403,
                content={
                    "error": "authenticated_start_required",
                    "message": "Start sessions from the wallet page on Credimi.",
                },
            )
        if method == "GET" and path == "/api/agents":
            return await _agents(temporal, chat)
        if method == "GET" and path == "/api/sessions":
            return await _sessions(temporal, chat)

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
                offset = await WorkflowStreamClient.create(temporal, session_id).get_offset()
        if not session_id:
            return await call_next(request)
        response = await call_next(request)
        if response.status_code != 200 or not response.headers.get("content-type", "").startswith(
            "text/event-stream"
        ):
            return response
        return StreamingResponse(
            _complete_stream(response, temporal, session_id, offset),
            media_type="text/event-stream",
            headers=_sse_headers(),
        )

    @harness.middleware("http")
    async def scoped_chat(request, call_next):
        if request.url.path.rstrip("/") == CALLBACK_RESULT_PATH:
            # Save approvals are signed by Credimi; the harness route would let the UI settle a
            # review without them.
            response = JSONResponse(
                status_code=403,
                content={
                    "error": "review_through_credimi",
                    "message": "Save reviews are sent to Credimi.",
                },
            )
        else:
            try:
                response = await dispatch(request, call_next)
            except RPCError as exc:
                response = _temporal_error(exc, request.url.path)
        _frame_for_credimi(response)
        return response

    return harness


async def _harness_for(app, namespace: str) -> FastAPI:
    """The namespace's harness app, connecting on first use. Entries are never evicted: there is
    one per organization. A failed connect is not cached."""
    async with app.state.lock:
        harness = app.state.harnesses.get(namespace)
        if harness is None:
            harness = _create_harness(await app.state.connect(namespace))
            app.state.harnesses[namespace] = harness
        return harness


class _ScopeMiddleware:
    """Route each HTTP request to its namespace's harness app, scoped to the named chat."""

    def __init__(self, app: ASGIApp, owner: Starlette) -> None:
        self.app = app
        self.owner = owner

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return
        headers = {
            name.decode("latin-1"): value.decode("latin-1") for name, value in scope["headers"]
        }
        namespace = headers.get(NAMESPACE_HEADER, "")
        chat = headers.get(CHAT_HEADER, "")
        if not NAMESPACE_PATTERN.fullmatch(namespace) or not CHAT_PATTERN.fullmatch(chat):
            response = JSONResponse(status_code=400, content={"error": "missing_scope"})
            await response(scope, receive, send)
            return
        try:
            harness = await _harness_for(self.owner, namespace)
        except Exception:
            logger.exception("connect namespace %s", namespace)
            response = JSONResponse(
                status_code=503,
                content={"error": "temporal_unavailable"},
                headers={**NO_STORE, "Retry-After": TEMPORAL_RETRY_AFTER},
            )
            await response(scope, receive, send)
            return
        state = {**scope.get("state", {}), "chat": chat}
        await harness({**scope, "state": state}, receive, send)


def create_app(*, connect: Callable[[str], Awaitable[Client]] = connect_namespace) -> Starlette:
    app = Starlette()
    app.state.connect = connect
    app.state.harnesses = {}
    app.state.lock = asyncio.Lock()
    app.user_middleware = [Middleware(_ScopeMiddleware, owner=app)]
    return app


def main() -> None:
    logging.basicConfig(level=logging.INFO)
    uvicorn.run(create_app(), host="0.0.0.0", port=_port())


if __name__ == "__main__":
    main()
