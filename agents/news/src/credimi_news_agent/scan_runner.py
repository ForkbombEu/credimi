# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Run one scan as a NewsAgent session: start it, send `scan`, stream the turn, close it.

Shared by the CLI and by the `news.run_scan` activity of the daily workflow.

No ``from __future__ import annotations``: activity argument and return types cross Temporal's
data converter, which resolves them from these annotations.
"""

import asyncio
import contextlib
from collections.abc import Callable
from typing import Any

from temporal_agent_harness.harness.agent_client import (
    AgentClient,
    AgentTurnError,
    AgentTurnTimeout,
)
from temporal_agent_harness.harness.agent_protocol import AgentConfig
from temporal_agent_harness.harness.agent_protocol.events import AgentEventType
from temporalio import activity
from temporalio.client import Client
from temporalio.common import WorkflowIDConflictPolicy
from temporalio.exceptions import ApplicationError

from .models import TASK_QUEUE, WORKFLOW_TYPE, RunScanInput, ScanReport, ScanRequest

TURN_TIMEOUT = 30 * 60
HEARTBEAT_EVERY = 30


class ScanFailedError(Exception):
    """The scan turn ended without a report."""


async def run_scan(
    client: Client,
    request: ScanRequest,
    workflow_id: str,
    on_event: Callable[[Any], None] | None = None,
) -> ScanReport:
    """Run ``request`` in the session workflow ``workflow_id`` and return its report.

    The `scan` message is sent with a fixed update id, so a retried caller reattaches to the
    turn of a session that is still running instead of starting a second one.
    """
    handle = await client.start_workflow(
        WORKFLOW_TYPE,
        AgentConfig(),
        id=workflow_id,
        task_queue=TASK_QUEUE,
        id_conflict_policy=WorkflowIDConflictPolicy.USE_EXISTING,
    )
    report: ScanReport | None = None
    failure = ""
    try:
        stream = await AgentClient(client, workflow_id).send_message(
            "scan",
            request.model_dump(),
            on_item=lambda item, _offset: item,
            timeout=TURN_TIMEOUT,
            update_id=f"{workflow_id}/scan",
        )
        async for item in stream:
            if isinstance(item, AgentTurnError | AgentTurnTimeout):
                failure = str(item)
                continue
            if on_event is not None:
                on_event(item)
            if item.event.type == AgentEventType.MESSAGE_HANDLER_END:
                report = ScanReport.model_validate(item.event.output)
    finally:
        await handle.signal("close")

    if report is None:
        raise ScanFailedError(failure or "no report")
    return report


async def _heartbeat() -> None:
    while True:
        activity.heartbeat()
        await asyncio.sleep(HEARTBEAT_EVERY)


@activity.defn(name="news.run_scan")
async def run_scan_activity(inp: RunScanInput) -> ScanReport:
    """One scan for the daily workflow, run through the worker's own Temporal client."""
    heartbeat = asyncio.create_task(_heartbeat())
    try:
        return await run_scan(activity.client(), inp.request, inp.workflow_id)
    except ScanFailedError as exc:
        raise ApplicationError(str(exc), type="ScanFailed") from exc
    finally:
        heartbeat.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await heartbeat
