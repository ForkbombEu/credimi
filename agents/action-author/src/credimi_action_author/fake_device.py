# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""An offline stand-in for the phone, for tests and for the worker without a device serial.

* ``FakeDevice`` implements ``device.Device``: it records the commands it gets and answers like
  a wallet on its first screen whose Maestro runs pass.
* ``python -m credimi_action_author.fake_device`` is a tiny stdio MCP server with a few
  agent-device tool names (plus ``install``, which the worker must hide). It appends every call
  to the JSON-lines file named by ``ACTION_AUTHOR_FAKE_MCP_LOG``, if set.
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from typing import Literal

from mcp.server.mcpserver import MCPServer
from pydantic import BaseModel

from .contracts import InstalledBuild, PrepareBuildRequest

FAKE_SERIAL = "fake-serial"

SCREEN = {
    "ui_schema": {
        "platform": "android",
        "abbreviations": {"b": "bounds", "txt": "text", "rid": "resource-id", "c": "children"},
        "defaults": {"enabled": True, "clickable": False, "txt": "", "rid": "", "a11y": ""},
    },
    "elements": [
        {
            "b": "[0,0][1080,2400]",
            "cls": "android.widget.FrameLayout",
            "c": [
                {
                    "b": "[63,360][1017,446]",
                    "txt": "Welcome to your Wallet",
                    "rid": "eu.europa.ec.euidi:id/pin_screen_title",
                },
                {"b": "[63,500][1017,560]", "txt": "Type a PIN"},
                {"b": "[63,900][1017,1100]", "cls": "android.widget.EditText", "clickable": True},
            ],
        }
    ],
}


class RefTarget(BaseModel):
    kind: Literal["ref"]
    ref: str


class SelectorTarget(BaseModel):
    kind: Literal["selector"]
    selector: str


class PointTarget(BaseModel):
    kind: Literal["point"]
    x: float
    y: float


UITarget = RefTarget | SelectorTarget | PointTarget


class FakeDevice:
    """Records every command; Maestro runs pass unless ``fail_runs`` says otherwise."""

    def __init__(self, serial: str = FAKE_SERIAL, fail_runs: int = 0) -> None:
        self.serial = serial
        self.fail_runs = fail_runs
        self.calls: list[tuple] = []
        self.installed_sha256 = ""

    async def prepare_build(self, req: PrepareBuildRequest) -> InstalledBuild:
        self.installed_sha256 = req.build.apk_sha256
        return InstalledBuild(
            package_id="eu.europa.ec.euidi",
            version_code="1",
            version_name="test",
            apk_sha256=self.installed_sha256,
        )

    async def verify_build(self, app_id: str, apk_sha256: str) -> None:
        if app_id != "eu.europa.ec.euidi" or apk_sha256 != self.installed_sha256:
            raise ValueError("installed wallet changed since the Credimi build was selected")

    async def close_session(self) -> str:
        self.calls.append(("close_session",))
        return "Closed: action-author"

    async def clear_app(self, app_id: str) -> str:
        self.calls.append(("clear_app", app_id))
        return "Success"

    async def maestro_test(self, flow: Path, timeout: float) -> tuple[int, str]:
        self.calls.append(("maestro_test", flow.read_text()))
        if self.fail_runs > 0:
            self.fail_runs -= 1
            return 1, (
                "Running on fake\n"
                'Launch app "eu.europa.ec.euidi"... COMPLETED\n'
                'Assert that "Welcome to your Wallet" is visible... FAILED\n\n'
                'Assertion is false: "Welcome to your Wallet" is visible\n'
            )
        return 0, (
            "Running on fake\n"
            'Launch app "eu.europa.ec.euidi"... COMPLETED\n'
            "Input text... COMPLETED\n"
            "Take screenshot onboarding-complete... COMPLETED\n"
        )

    async def inspect_screen(self) -> str:
        self.calls.append(("inspect_screen",))
        notice = "Important: Maestro Viewer is available at http://127.0.0.1:10000/."
        return f"{notice}\n{json.dumps(SCREEN)}"

    async def cheat_sheet(self) -> str:
        self.calls.append(("cheat_sheet",))
        return "Important: Maestro Viewer is available at x.\n# Maestro cheat sheet\n- tapOn: Text"


def _server() -> MCPServer:
    server = MCPServer("fake-agent-device")
    log = os.environ.get("ACTION_AUTHOR_FAKE_MCP_LOG", "")

    def record(tool: str, **args: object) -> None:
        if log:
            with open(log, "a") as f:
                f.write(json.dumps({"tool": tool, "args": args}) + "\n")

    @server.tool(name="open")
    def open_app(
        app: str = "",
        session: str = "",
        platform: str = "",
        serial: str = "",
    ) -> str:
        """Open an app in a session."""
        record("open", app=app, session=session, platform=platform, serial=serial)
        return f"Opened: {app}"

    @server.tool()
    def snapshot(session: str = "", platform: str = "", serial: str = "") -> str:
        """Capture the accessibility tree."""
        record("snapshot", session=session, platform=platform, serial=serial)
        return '@e1 [text] "Welcome to your Wallet"\n@e2 [text-field] "PIN 123456"'

    @server.tool()
    def click(target: UITarget, session: str = "", platform: str = "", serial: str = "") -> str:
        """Click a structured UI target."""
        record(
            "click", target=target.model_dump(), session=session, platform=platform, serial=serial
        )
        return "Clicked the wallet control"

    @server.tool()
    def fill(
        target: UITarget, text: str, session: str = "", platform: str = "", serial: str = ""
    ) -> str:
        """Fill a structured input target."""
        record(
            "fill",
            target=target.model_dump(),
            text=text,
            session=session,
            platform=platform,
            serial=serial,
        )
        return f"Filled PIN {text}"

    @server.tool()
    def close(session: str = "", platform: str = "", serial: str = "") -> str:
        """Close the session."""
        record("close", session=session, platform=platform, serial=serial)
        return "Closed"

    @server.tool()
    def install(app: str = "", serial: str = "") -> str:
        """Install an app (must never reach the model)."""
        record("install", app=app, serial=serial)
        return "Installed"

    return server


if __name__ == "__main__":
    sys.exit(_server().run("stdio"))
