# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""AgentDeviceServer against the fake agent-device MCP server (a real stdio subprocess)."""

import asyncio
import contextlib
import json
import os
import signal
import sys
from unittest.mock import AsyncMock

import pytest

from credimi_action_author import device as device_module
from credimi_action_author.activities import ActionAuthorActivities
from credimi_action_author.device import AgentDeviceServer, LocalDevice, _run, _visible_args
from credimi_action_author.models import AGENT_DEVICE_SESSION

FAKE_MCP = [sys.executable, "-m", "credimi_action_author.fake_device"]


@pytest.fixture
async def server(monkeypatch, tmp_path):
    log = tmp_path / "mcp.jsonl"
    monkeypatch.setenv("ACTION_AUTHOR_FAKE_MCP_LOG", str(log))
    server = AgentDeviceServer(FAKE_MCP, "pinned", "eu.europa.ec.euidi", {"PIN": "123456"})
    await server.connect()
    yield server, log
    await server.cleanup()


def text(result) -> str:
    return result.content[0].text


@pytest.mark.parametrize(
    "target",
    [
        {"kind": "ref", "ref": "@e25"},
        {"kind": "selector", "selector": "text=Continue"},
        {"kind": "point", "x": 540, "y": 1800},
    ],
)
def test_ui_targets_survive_device_selector_filtering(target):
    assert _visible_args({"target": target, "serial": "other", "platform": "ios"}) == {
        "target": target
    }


@pytest.mark.parametrize("target", ["mobile", "tv", "desktop"])
def test_device_form_targets_cannot_override_pinned_device(target):
    assert _visible_args({"target": target, "serial": "other", "text": "Continue"}) == {
        "text": "Continue"
    }


async def test_only_exploration_tools_without_device_fields_are_listed(server):
    srv, _ = server
    tools = {t.name: t for t in await srv.list_tools()}
    assert "install" not in tools
    assert set(tools["open"].input_schema["properties"]) == {"app"}


async def test_calls_are_pinned_scoped_and_masked(server):
    srv, log = server
    opened = await srv.call_tool("open", {"app": "eu.europa.ec.euidi", "serial": "other"})
    snap = await srv.call_tool("snapshot", {"session": "mine", "debug": True, "target": "tv"})
    other_app = await srv.call_tool("open", {"app": "com.android.settings"})
    install = await srv.call_tool("install", {"app": "x"})
    close_other = await srv.call_tool("close", {"app": "com.android.settings"})

    assert text(opened) == "Opened: eu.europa.ec.euidi"
    assert "PIN ${PIN}" in text(snap) and "123456" not in text(snap)
    assert other_app.is_error and "only the app under test" in text(other_app)
    assert install.is_error and "not available" in text(install)
    assert close_other.is_error and "only the app under test" in text(close_other)
    calls = [json.loads(line) for line in log.read_text().splitlines()]
    assert [c["tool"] for c in calls] == ["open", "snapshot"]
    for call in calls:
        assert call["args"]["serial"] == "pinned"
        assert call["args"]["session"] == AGENT_DEVICE_SESSION
        assert call["args"]["platform"] == "android"


async def test_device_operations_require_target_selection(tmp_path, monkeypatch):
    log = tmp_path / "mcp.jsonl"
    monkeypatch.setenv("ACTION_AUTHOR_FAKE_MCP_LOG", str(log))
    srv = AgentDeviceServer(FAKE_MCP, "pinned")
    async with srv:
        result = await srv.call_tool("open", {"app": "com.android.settings"})
    assert result.is_error
    assert "set_target" in text(result)
    assert not log.exists()


@pytest.mark.parametrize(
    ("code", "output"),
    [
        (1, '{"error":{"code":"DEVICE_BUSY","message":"still attached"}}'),
        (124, "[timed out after 60s]"),
        (1, "not a structured SESSION_NOT_FOUND error"),
    ],
)
async def test_failed_close_prevents_maestro_handoff(monkeypatch, code, output):
    monkeypatch.setattr(device_module.shutil, "which", lambda name: "/unused")
    monkeypatch.setattr(device_module, "_run", AsyncMock(return_value=(code, output)))
    device = LocalDevice("test")
    inspect = AsyncMock(return_value="wallet screen")
    monkeypatch.setattr(device, "inspect_screen", inspect)
    with pytest.raises(RuntimeError, match="agent-device close failed"):
        await ActionAuthorActivities(device)._screen({})
    inspect.assert_not_awaited()


async def test_already_closed_session_allows_maestro_handoff(monkeypatch):
    monkeypatch.setattr(device_module.shutil, "which", lambda name: "/unused")
    monkeypatch.setattr(
        device_module,
        "_run",
        AsyncMock(return_value=(1, '{"error":{"code":"SESSION_NOT_FOUND"}}')),
    )
    device = LocalDevice("test")
    monkeypatch.setattr(device, "inspect_screen", AsyncMock(return_value="wallet screen"))
    assert await ActionAuthorActivities(device)._screen({}) == "wallet screen"


@pytest.mark.parametrize("cancel", [False, True])
async def test_timeout_and_cancellation_reap_process_tree(monkeypatch, cancel):
    original_spawn = asyncio.create_subprocess_exec
    ready = asyncio.Event()
    processes = []
    child_pids = []

    async def spawn(*args, **kwargs):
        proc = await original_spawn(*args, **kwargs)
        processes.append(proc)
        child_pids.append(int(await asyncio.wait_for(proc.stdout.readline(), 5)))
        ready.set()
        return proc

    monkeypatch.setattr(asyncio, "create_subprocess_exec", spawn)
    code = (
        "import subprocess,sys,time; "
        "child=subprocess.Popen([sys.executable,'-c','import time; time.sleep(30)']); "
        "print(child.pid,flush=True); print('tree-ready',flush=True); time.sleep(30)"
    )
    task = asyncio.create_task(_run([sys.executable, "-c", code], 30 if cancel else 0.1))
    try:
        await asyncio.wait_for(ready.wait(), 5)
        if cancel:
            task.cancel()
            with pytest.raises(asyncio.CancelledError):
                await asyncio.wait_for(task, 2)
        else:
            status, output = await asyncio.wait_for(task, 2)
            assert status == 124
            assert "tree-ready" in output
            assert "timed out" in output
        assert processes[0].returncode == -signal.SIGKILL
    finally:
        for pid in child_pids:
            with contextlib.suppress(ProcessLookupError):
                os.kill(pid, signal.SIGKILL)
        for proc in processes:
            with contextlib.suppress(ProcessLookupError):
                proc.kill()
        task.cancel()
        with contextlib.suppress(Exception, asyncio.CancelledError):
            await task
