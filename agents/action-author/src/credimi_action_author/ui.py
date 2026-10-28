# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Serve the author registry without reusing another agent's session manager."""

import os
from pathlib import Path

import uvicorn
from fastapi.responses import JSONResponse
from temporal_agent_harness.web import create_agent_harness_app


def create_app():
    app = create_agent_harness_app(
        registry_path=Path(__file__).resolve().parents[2] / "agents.toml",
        manager_workflow_id="action-author-session-manager",
    )

    @app.middleware("http")
    async def authenticated_start_only(request, call_next):
        if request.method == "POST" and request.url.path.rstrip("/") == "/api/sessions":
            return JSONResponse(
                status_code=403,
                content={
                    "error": "authenticated_start_required",
                    "message": "Start sessions through authenticated POST "
                    "/api/action-author/sessions on Credimi, then select the session here.",
                },
            )
        return await call_next(request)

    return app


def main() -> None:
    app = create_app()
    uvicorn.run(app, host="127.0.0.1", port=int(os.environ.get("ACTION_AUTHOR_UI_PORT", "8001")))


if __name__ == "__main__":
    main()
