# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Serve the author registry without reusing another agent's session manager."""

import base64
import json
import os
from pathlib import Path

import uvicorn
from fastapi.responses import (
    FileResponse,
    HTMLResponse,
    JSONResponse,
    RedirectResponse,
    StreamingResponse,
)
from temporal_agent_harness.harness.agent_protocol import AgentEvent
from temporal_agent_harness.web import create_agent_harness_app
from temporal_agent_harness.web.app import _sse_headers, _yield_item
from temporalio.api.common.v1 import Payload
from temporalio.client import WorkflowExecutionStatus
from temporalio.contrib.workflow_streams import WorkflowStreamClient

from .models import WORKFLOW_TYPE

START_PAGE = Path(__file__).with_name("start.html")
START_PAGE_HEADERS = {
    "Cache-Control": "no-store",
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


def create_app():
    app = create_agent_harness_app(
        registry_path=Path(__file__).resolve().parents[2] / "agents.toml",
        manager_workflow_id="action-author-session-manager",
    )

    @app.middleware("http")
    async def authenticated_start_only(request, call_next):
        if request.method == "GET" and request.url.path == "/action-save-review.js":
            return FileResponse(
                Path(__file__).with_name("save-review.js"),
                media_type="text/javascript",
                headers={"Cache-Control": "no-store"},
            )
        if request.method == "GET" and request.url.path.startswith("/api/agent-interface/"):
            response = await call_next(request)
            if response.status_code != 200:
                return response
            interface = json.loads(b"".join([chunk async for chunk in response.body_iterator]))
            return JSONResponse(
                [handler for handler in interface if handler["name"] != "save_wallet_action"],
                headers={"Cache-Control": "no-store"},
            )
        if request.method == "GET" and request.url.path == "/start":
            return FileResponse(START_PAGE, headers=START_PAGE_HEADERS)
        if request.method == "GET" and request.url.path == "/" and not request.query_params:
            return RedirectResponse("/start")
        if request.method == "GET" and request.url.path == "/":
            response = await call_next(request)
            if response.status_code == 200 and response.headers.get("content-type", "").startswith(
                "text/html"
            ):
                html = b"".join([chunk async for chunk in response.body_iterator]).decode()
                return HTMLResponse(
                    html.replace("</head>", AUTHENTICATED_START_SCRIPT + "</head>", 1),
                    headers={"Cache-Control": "no-store"},
                )
            return response
        if request.method == "POST" and request.url.path.rstrip("/") == "/api/sessions":
            return JSONResponse(
                status_code=403,
                content={
                    "error": "authenticated_start_required",
                    "message": "Start sessions through authenticated POST "
                    "/api/action-author/sessions on Credimi, then select the session here.",
                },
            )
        if request.method == "GET" and request.url.path == "/api/sessions":
            response = await call_next(request)
            if response.status_code != 200:
                return response
            sessions = json.loads(b"".join([chunk async for chunk in response.body_iterator]))
            known = {session["workflow_id"] for session in sessions}
            # Direct Credimi starts are not in the harness manager's registry.
            # Upstream discovers only running workflows, so add recent closed runs.
            async for execution in app.state.temporal.list_workflows(
                query=f"WorkflowType='{WORKFLOW_TYPE}' AND ExecutionStatus!='Running'",
                limit=200,
            ):
                if execution.id not in known:
                    sessions.append(
                        {
                            "workflow_id": execution.id,
                            "created_at": execution.start_time.timestamp(),
                            "label": "Credimi action author",
                            "agent_workflow_type": WORKFLOW_TYPE,
                            "is_discovered": True,
                            "execution_status": execution.status.name,
                            "closed": True,
                        }
                    )
            return JSONResponse(sessions, headers={"Cache-Control": "no-store"})
        session_id = None
        offset = 0
        if request.method == "GET" and request.url.path == "/api/attach":
            session_id = request.query_params.get("session_id")
            try:
                offset = int(request.query_params.get("from_offset", "0"))
            except ValueError:
                return await call_next(request)
            if session_id and offset >= 0:
                temporal = app.state.temporal
                state = await temporal.get_workflow_handle(session_id).describe()
                if state.status != WorkflowExecutionStatus.RUNNING:
                    return StreamingResponse(
                        _closed_history(temporal, session_id, offset),
                        media_type="text/event-stream",
                        headers=_sse_headers(),
                    )
        elif request.method == "POST" and request.url.path == "/api/chat":
            try:
                body = await request.json()
            except ValueError:
                return await call_next(request)
            if isinstance(body, dict) and isinstance(body.get("session_id"), str):
                session_id = body["session_id"]
                offset = await WorkflowStreamClient.create(
                    app.state.temporal, session_id
                ).get_offset()
        if session_id:
            response = await call_next(request)
            if response.status_code == 200 and response.headers.get("content-type", "").startswith(
                "text/event-stream"
            ):
                return StreamingResponse(
                    _complete_stream(response, app.state.temporal, session_id, offset),
                    media_type="text/event-stream",
                    headers=_sse_headers(),
                )
            return response
        return await call_next(request)

    return app


def main() -> None:
    app = create_app()
    uvicorn.run(app, host="127.0.0.1", port=int(os.environ.get("ACTION_AUTHOR_UI_PORT", "8001")))


if __name__ == "__main__":
    main()
