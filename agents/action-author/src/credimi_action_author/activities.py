# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Worker activities: everything that touches the phone or the filesystem.

``ActionAuthorActivities`` wraps a ``Device`` (the real ``LocalDevice``, or a fake in tests and
offline runs).
"""

from __future__ import annotations

import asyncio
import contextlib
import os
import tempfile
import time
from pathlib import Path

from temporalio import activity
from temporalio.exceptions import ApplicationError

from .contracts import PREPARE_BUILD, InstalledBuild, PrepareBuildRequest
from .device import Device
from .maestro import compact_hierarchy, output_tail, parse_run_output, strip_viewer_notice
from .models import (
    DEFAULT_MAX_MODEL_CALLS,
    DEFAULT_TOKEN_BUDGET,
    SLOTS,
    DeviceInfo,
    ReplayRequest,
    ReplayResult,
    RunOutcome,
    ScreenRequest,
)
from .validation import APP_ID

DEVICE_INFO = "action_author.device_info"
CHEAT_SHEET = "action_author.maestro_cheat_sheet"
INSPECT_SCREEN = "action_author.maestro_inspect_screen"
RESET_APP = "action_author.reset_app"
REPLAY_ACTION = "action_author.replay_action"

MAESTRO_TEST_TIMEOUT = 300.0
HEARTBEAT_EVERY = 10.0
CHEAT_SHEET_LIMIT = 40_000
SCREEN_LIMIT = 8_000

PACKAGE_ROOT = Path(__file__).resolve().parents[2]


def _env_int(name: str, default: int) -> int:
    value = os.environ.get(name, "").strip()
    try:
        return int(value) if value else default
    except ValueError:
        # The workflow does not bound this activity's retries: fail it instead of looping.
        raise ApplicationError(f"{name} must be an integer", non_retryable=True) from None


async def _with_heartbeats[T](coro, every: float = HEARTBEAT_EVERY) -> T:
    """Await ``coro`` while heartbeating, so a stuck worker is noticed."""
    task = asyncio.ensure_future(coro)
    try:
        while True:
            done, _ = await asyncio.wait({task}, timeout=every)
            if done:
                return task.result()
            activity.heartbeat()
    finally:
        if not task.done():
            task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await task


class ActionAuthorActivities:
    def __init__(self, device: Device, out_dir: Path | None = None) -> None:
        self._device = device
        self._out_dir = out_dir or Path(
            os.environ.get("ACTION_AUTHOR_OUT_DIR") or PACKAGE_ROOT / "out"
        )
        self._cheat_sheet: str | None = None

    def all(self) -> list:
        return [
            self.device_info,
            self.prepare_build,
            self.maestro_cheat_sheet,
            self.maestro_inspect_screen,
            self.reset_app,
            self.replay_action,
        ]

    @activity.defn(name=DEVICE_INFO)
    async def device_info(self) -> DeviceInfo:
        return DeviceInfo(
            serial=self._device.serial,
            token_budget=_env_int("ACTION_AUTHOR_TOKEN_BUDGET", DEFAULT_TOKEN_BUDGET),
            max_model_calls=_env_int("ACTION_AUTHOR_MAX_MODEL_CALLS", DEFAULT_MAX_MODEL_CALLS),
        )

    @activity.defn(name=PREPARE_BUILD)
    async def prepare_build(self, req: PrepareBuildRequest) -> InstalledBuild:
        return await _with_heartbeats(self._device.prepare_build(req))

    @activity.defn(name=CHEAT_SHEET)
    async def maestro_cheat_sheet(self) -> str:
        """Maestro's syntax reference, fetched once per worker from Maestro MCP `cheat_sheet`."""
        if self._cheat_sheet is None:
            text = strip_viewer_notice(await self._device.cheat_sheet())
            self._cheat_sheet = text[:CHEAT_SHEET_LIMIT]
        return self._cheat_sheet

    async def _screen(self) -> str:
        await self._device.close_session()
        raw = await self._device.inspect_screen()
        return compact_hierarchy(raw, SCREEN_LIMIT)

    @activity.defn(name=INSPECT_SCREEN)
    async def maestro_inspect_screen(self, req: ScreenRequest) -> str:
        """The current screen as Maestro sees it (closes the agent-device session first)."""
        return await _with_heartbeats(self._screen())

    @activity.defn(name=RESET_APP)
    async def reset_app(self, req: ScreenRequest) -> str:
        """Close the agent-device session and clear the app's data (slot precondition)."""
        if not APP_ID.fullmatch(req.app_id):
            raise ValueError(f"invalid app id {req.app_id!r}")
        await self._device.close_session()
        return await self._device.clear_app(req.app_id)

    async def _run_once(self, req: ReplayRequest, flow: Path) -> RunOutcome:
        await self._device.close_session()
        await self._device.verify_build(req.app_id, req.apk_sha256)
        await self._device.clear_app(req.app_id)
        start = time.monotonic()
        code, output = await self._device.maestro_test(flow, MAESTRO_TEST_TIMEOUT)
        seconds = round(time.monotonic() - start, 1)
        parsed = parse_run_output(output)
        failed = parsed.failed
        if code == 0 and failed is None:
            return RunOutcome(passed=True, seconds=seconds, completed=parsed.completed)
        try:
            screen = await self._screen()
        except Exception as exc:  # noqa: BLE001 - the replay verdict matters more
            screen = f"(could not inspect the screen: {exc})"
        return RunOutcome(
            passed=False,
            seconds=seconds,
            completed=parsed.completed,
            failed_command=failed.command if failed else f"exit code {code}",
            tail=output_tail(output),
            screen=screen,
        )

    async def _replay(self, req: ReplayRequest) -> ReplayResult:
        runs: list[RunOutcome] = []
        with tempfile.TemporaryDirectory(prefix="action-author-") as tmp:
            flow = Path(tmp) / f"{req.slot}.yaml"
            flow.write_text(req.yaml)
            for _ in range(req.runs):
                outcome = await self._run_once(req, flow)
                runs.append(outcome)
                if not outcome.passed:
                    return ReplayResult(accepted=False, runs=runs)
        path = self._out_dir / req.app_id / f"{req.slot}.yaml"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(req.yaml)
        return ReplayResult(accepted=True, runs=runs, path=str(path))

    @activity.defn(name=REPLAY_ACTION)
    async def replay_action(self, req: ReplayRequest) -> ReplayResult:
        """Replay a validated action ``req.runs`` times, each from a cleared app; on success write
        it to ``out/<appId>/<slot>.yaml``. The YAML embeds its input values; no ``-e`` is used."""
        if not APP_ID.fullmatch(req.app_id) or req.slot not in SLOTS:
            raise ValueError(f"invalid app id {req.app_id!r} or slot {req.slot!r}")
        return await _with_heartbeats(self._replay(req))
