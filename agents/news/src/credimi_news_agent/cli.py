# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Start one scan, print the agent's tool calls live, then print the report.

    python -m credimi_news_agent.cli --since 2026-10-01 --until 2026-10-14 [--source openid4vp]

The session workflow is closed when the turn ends, so each scan is one short workflow run.
"""

import argparse
import asyncio
import json
import os
import sys
from datetime import date

from temporal_agent_harness.harness.agent_client import (
    AgentClient,
    AgentTurnError,
    AgentTurnTimeout,
)
from temporal_agent_harness.harness.agent_protocol import AgentConfig
from temporal_agent_harness.harness.agent_protocol.events import AgentEventType
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio.client import Client
from temporalio.common import WorkflowIDConflictPolicy
from temporalio.envconfig import ClientConfig

from .models import DEFAULT_MODEL, TASK_QUEUE, WORKFLOW_TYPE, ScanReport, ScanRequest

TURN_TIMEOUT = 30 * 60


def _day(value: str) -> str:
    try:
        return date.fromisoformat(value).isoformat()
    except ValueError as exc:
        raise argparse.ArgumentTypeError(f"expected YYYY-MM-DD, got {value!r}") from exc


def _args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawTextHelpFormatter)
    p.add_argument("--since", required=True, type=_day, help="YYYY-MM-DD")
    p.add_argument("--until", required=True, type=_day, help="YYYY-MM-DD")
    p.add_argument("--source", action="append", default=[], help="source id (repeatable)")
    p.add_argument("--max-drafts", type=int, default=3)
    p.add_argument("--max-turns", type=int, default=30)
    p.add_argument("--model", default=DEFAULT_MODEL)
    p.add_argument("--workflow-id", help="default: news-scan-<since>-<until>")
    args = p.parse_args()
    if args.since > args.until:
        p.error("--since must not be after --until")
    return args


def _short(value: object, limit: int = 160) -> str:
    text = value if isinstance(value, str) else json.dumps(value)
    text = " ".join(text.split())
    return text if len(text) <= limit else text[: limit - 1] + "…"


async def run() -> int:
    args = _args()
    request = ScanRequest(
        since=args.since,
        until=args.until,
        source_ids=args.source,
        max_drafts=args.max_drafts,
        max_turns=args.max_turns,
        model=args.model,
    )
    workflow_id = args.workflow_id or f"news-scan-{args.since}-{args.until}"
    client = await Client.connect(
        **ClientConfig.load_client_connect_config(), plugins=[AgentHarnessPlugin()]
    )
    handle = await client.start_workflow(
        WORKFLOW_TYPE,
        AgentConfig(),
        id=workflow_id,
        task_queue=TASK_QUEUE,
        id_conflict_policy=WorkflowIDConflictPolicy.USE_EXISTING,
    )
    ui = os.environ.get("TEMPORAL_UI_URL", "http://localhost:8280")
    print(f"workflow {workflow_id}: {ui}/namespaces/{client.namespace}/workflows/{workflow_id}")

    report: ScanReport | None = None
    failure = ""
    try:
        stream = await AgentClient(client, workflow_id).send_message(
            "scan", request.model_dump(), on_item=lambda item, _offset: item, timeout=TURN_TIMEOUT
        )
        async for item in stream:
            if isinstance(item, AgentTurnError | AgentTurnTimeout):
                failure = str(item)
                continue
            ev = item.event
            if ev.type == AgentEventType.TOOL_START:
                print(f"  -> {ev.tool_name} {_short(ev.tool_input)}")
            elif ev.type == AgentEventType.TOOL_END:
                print(f"  <- {ev.tool_name} {_short(ev.tool_output)}")
            elif ev.type == AgentEventType.TOOL_ERROR:
                print(f"  !! {ev.tool_name} {_short(ev.message)}")
            elif ev.type == AgentEventType.MESSAGE_HANDLER_END:
                report = ScanReport.model_validate(ev.output)
    finally:
        await handle.signal("close")

    if report is None:
        print(f"scan failed: {failure or 'no report'}", file=sys.stderr)
        return 1
    print(json.dumps(report.model_dump(), indent=2))
    return 0 if not report.error else 2


def main() -> None:
    sys.exit(asyncio.run(run()))


if __name__ == "__main__":
    main()
