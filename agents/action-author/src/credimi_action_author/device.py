# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""The phone, as seen from the worker: agent-device over MCP for exploration, and the commands
the Maestro tools and the replay run (agent-device close, adb pm clear, maestro).

Every command is pinned to the worker's device serial. The model never chooses the device:
``AgentDeviceServer`` drops the device-selection fields from the tool schemas and overwrites them
on every call, and the agent-device process only sees that serial in its Android allowlist.

Device turn-taking: Maestro's Android driver cannot start while an agent-device session holds
the device, so every Maestro command first closes the agent-device session (the app stays open).
"""

from __future__ import annotations

import asyncio
import contextlib
import json
import os
import shutil
import signal
from collections.abc import Callable
from pathlib import Path
from typing import Any, Protocol

from agents.mcp import MCPServer, MCPServerStdio
from mcp import Tool as MCPTool
from mcp.types import CallToolResult, TextContent

from .contracts import InstalledBuild, PrepareBuildRequest
from .maestro import strip_viewer_notice
from .models import AGENT_DEVICE_SESSION
from .validation import redact

# agent-device MCP tools the model may use; the others (install, settings, shell-like tools,
# other devices) are hidden and refused.
AGENT_DEVICE_TOOLS = (
    "open",
    "snapshot",
    "press",
    "click",
    "fill",
    "type",
    "scroll",
    "swipe",
    "back",
    "wait",
    "find",
    "get",
    "is",
    "screenshot",
    "close",
)
# Tool input fields the model never sees and that are dropped from its calls:
# * device, tenant and session selection, set by the worker only;
# * fields reaching beyond the app under test (URLs, Metro, host paths, recordings, shutdown);
# * diagnostics and output-format knobs, which only make the schemas longer.
_HIDDEN_FIELDS = frozenset(
    {
        "session",
        "platform",
        "deviceTarget",
        "device",
        "udid",
        "serial",
        "androidDeviceAllowlist",
        "tenant",
        "runId",
        "leaseId",
        "iosSimulatorDeviceSet",
        "stateDir",
        "daemonBaseUrl",
        "daemonAuthToken",
        "url",
        "launchUrl",
        "bundleUrl",
        "metroHost",
        "metroPort",
        "launchConsole",
        "launchArgs",
        "surface",
        "saveScript",
        "force",
        "shutdown",
        "path",
        "record",
        "recordAs",
        "noRecord",
        "testIme",
        "debug",
        "includeCost",
        "mcpOutputFormat",
        "responseLevel",
        "customActions",
    }
)
# On commands without a UI target, `target` is the device form (mobile, tv, desktop).
_DEVICE_FORMS = {"mobile", "tv", "desktop"}
TOOL_OUTPUT_LIMIT = 20_000
MAESTRO_MCP_TIMEOUT = 180.0


def agent_device_env(serial: str) -> dict[str, str]:
    return {
        **os.environ,
        "AGENT_DEVICE_ANDROID_DEVICE_ALLOWLIST": serial,
        "AGENT_DEVICE_PLATFORM": "android",
        "AGENT_DEVICE_SESSION": AGENT_DEVICE_SESSION,
    }


def _is_device_form(schema: Any) -> bool:
    return isinstance(schema, dict) and set(schema.get("enum") or ()) == _DEVICE_FORMS


def _visible_args(arguments: dict[str, Any]) -> dict[str, Any]:
    return {
        k: v
        for k, v in arguments.items()
        if k not in _HIDDEN_FIELDS
        and not (k == "target" and isinstance(v, str) and v in _DEVICE_FORMS)
    }


def _hide_fields(tool: MCPTool) -> MCPTool:
    schema = dict(tool.input_schema or {})
    props = {
        k: v
        for k, v in (schema.get("properties") or {}).items()
        if k not in _HIDDEN_FIELDS and not (k == "target" and _is_device_form(v))
    }
    schema["properties"] = props
    if "required" in schema:
        schema["required"] = [r for r in schema["required"] if r in props]
    return tool.model_copy(update={"input_schema": schema})


def _error(text: str) -> CallToolResult:
    return CallToolResult(content=[TextContent(type="text", text=f"ERROR: {text}")], is_error=True)


class AgentDeviceServer(MCPServerStdio):
    """``agent-device mcp`` restricted to the exploration tools and pinned to one device.

    ``app_id`` (when set) is the only app `open` may start; ``params`` are masked in outputs.
    """

    def __init__(
        self,
        command: list[str],
        serial: str,
        app_id: str = "",
        params: dict[str, str] | None = None,
    ) -> None:
        super().__init__(
            name="agent-device",
            params={"command": command[0], "args": command[1:], "env": agent_device_env(serial)},
            client_session_timeout_seconds=120,
        )
        self._serial = serial
        self._app_id = app_id
        self._params = params or {}

    async def list_tools(self, run_context=None, agent=None) -> list[MCPTool]:
        tools = await super().list_tools(run_context, agent)
        return [_hide_fields(t) for t in tools if t.name in AGENT_DEVICE_TOOLS]

    async def call_tool(
        self,
        tool_name: str,
        arguments: dict[str, Any] | None,
        meta: dict[str, Any] | None = None,
    ) -> CallToolResult:
        if tool_name not in AGENT_DEVICE_TOOLS:
            return _error(f"tool {tool_name!r} is not available; use {list(AGENT_DEVICE_TOOLS)}")
        if not self._app_id:
            return _error("call set_target before interacting with the device")
        args = _visible_args(arguments or {})
        app = args.get("app")
        if self._app_id and (
            (tool_name == "open" and app != self._app_id)
            or (tool_name == "close" and app not in (None, "", self._app_id))
        ):
            return _error(f"{tool_name} only the app under test: app={self._app_id!r}")
        args |= {"serial": self._serial, "platform": "android", "session": AGENT_DEVICE_SESSION}
        result = await super().call_tool(tool_name, args, meta)
        # The model reads the text blocks; drop the structured copy so it is not stored unmasked.
        result.structured_content = None
        for block in result.content:
            if isinstance(block, TextContent):
                text = redact(block.text, self._params)
                if len(text) > TOOL_OUTPUT_LIMIT:
                    text = text[:TOOL_OUTPUT_LIMIT] + "\n[... truncated]"
                block.text = text
        return result


def agent_device_factory(
    command: list[str], serial: str
) -> Callable[[dict[str, Any] | None], MCPServer]:
    """The StatelessMCPServerProvider factory. The workflow's factory argument carries the app
    under test and the parameter values to mask; the serial is the worker's."""

    def factory(arg: dict[str, Any] | None) -> MCPServer:
        arg = arg or {}
        return AgentDeviceServer(command, serial, arg.get("app_id", ""), arg.get("params") or {})

    return factory


class Device(Protocol):
    """What the activities need from the phone. ``LocalDevice`` is the real one."""

    serial: str

    async def prepare_build(self, req: PrepareBuildRequest) -> InstalledBuild: ...

    async def verify_build(self, app_id: str, apk_sha256: str) -> None: ...

    async def close_session(self) -> str: ...

    async def clear_app(self, app_id: str) -> str: ...

    async def maestro_test(self, flow: Path, timeout: float) -> tuple[int, str]: ...

    async def inspect_screen(self) -> str: ...

    async def cheat_sheet(self) -> str: ...


async def _run(cmd: list[str], timeout: float, cwd: Path | None = None) -> tuple[int, str]:
    """Run a command; (exit code, combined output). Exit code 124 on timeout."""
    proc = await asyncio.create_subprocess_exec(
        *cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.STDOUT,
        cwd=cwd,
        start_new_session=True,
        env={
            **os.environ,
            "MAESTRO_CLI_NO_ANALYTICS": "1",
            "MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED": "true",
        },
    )
    # Shield the pipe reader so timeout/cancellation does not discard buffered output.
    output = asyncio.create_task(proc.communicate())
    try:
        out, _ = await asyncio.wait_for(asyncio.shield(output), timeout)
    except TimeoutError:
        with contextlib.suppress(ProcessLookupError):
            os.killpg(proc.pid, signal.SIGKILL)
        out, _ = await output
        return 124, (out or b"").decode(errors="replace") + f"\n[timed out after {timeout:.0f}s]"
    except asyncio.CancelledError:
        with contextlib.suppress(ProcessLookupError):
            os.killpg(proc.pid, signal.SIGKILL)
        await asyncio.shield(output)
        raise
    return proc.returncode or 0, (out or b"").decode(errors="replace")


async def maestro_mcp_call(tool: str, arguments: dict[str, Any]) -> str:
    """One-shot Maestro MCP client: start ``maestro mcp``, initialize, call one tool, exit."""
    server = MCPServerStdio(
        name="maestro",
        params={"command": "maestro", "args": ["mcp"]},
        client_session_timeout_seconds=MAESTRO_MCP_TIMEOUT,
    )
    await server.connect()
    try:
        result = await server.call_tool(tool, arguments)
    finally:
        await server.cleanup()
    text = "\n".join(b.text for b in result.content if isinstance(b, TextContent))
    if result.is_error:
        raise RuntimeError(f"maestro {tool} failed: {strip_viewer_notice(text)[:2000]}")
    return text


class LocalDevice:
    """An Android device on this host's adb, driven by agent-device, adb and maestro."""

    def __init__(self, serial: str) -> None:
        self.serial = serial
        for tool in ("agent-device", "adb", "maestro", "apkanalyzer"):
            if shutil.which(tool) is None:
                raise RuntimeError(f"{tool} is not on PATH")

    async def close_session(self) -> str:
        # Ends the agent-device session (and its UI automation) without stopping the app.
        code, out = await _run(
            [
                "agent-device",
                "close",
                "--serial",
                self.serial,
                "--session",
                AGENT_DEVICE_SESSION,
                "--json",
            ],
            timeout=60,
        )
        if code != 0:
            try:
                error = json.loads(out).get("error", {})
            except (ValueError, AttributeError):
                error = {}
            if not isinstance(error, dict) or error.get("code") != "SESSION_NOT_FOUND":
                raise RuntimeError(f"agent-device close failed (exit {code}): {out.strip()}")
        return out.strip()

    async def clear_app(self, app_id: str) -> str:
        code, out = await _run(["adb", "-s", self.serial, "shell", "pm", "clear", app_id], 60)
        if code != 0 or "Success" not in out:
            raise RuntimeError(f"pm clear {app_id} failed: {out.strip()}")
        return out.strip()

    async def maestro_test(self, flow: Path, timeout: float) -> tuple[int, str]:
        cmd = ["maestro", "--device", self.serial, "test", str(flow)]
        # Screenshots land in the flow's (temporary) directory.
        return await _run(cmd, timeout, cwd=flow.parent)

    async def inspect_screen(self) -> str:
        return await maestro_mcp_call("inspect_screen", {"device_id": self.serial})

    async def cheat_sheet(self) -> str:
        return await maestro_mcp_call("cheat_sheet", {})

    async def prepare_build(self, req: PrepareBuildRequest) -> InstalledBuild:
        from .apk import prepare_apk

        return await prepare_apk(self, req)

    async def verify_build(self, app_id: str, apk_sha256: str) -> None:
        from .apk import installed_digest

        if await installed_digest(self.serial, app_id) != apk_sha256:
            raise ValueError("installed wallet changed since the Credimi build was selected")
