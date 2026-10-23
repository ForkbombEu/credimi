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
from typing import Any

from temporal_agent_harness.harness.agent_protocol.events import AgentEventType
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio.client import Client
from temporalio.envconfig import ClientConfig

from .models import DEFAULT_MODEL, ScanRequest
from .scan_runner import ScanFailedError, run_scan


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
    p.add_argument("--max-tokens", type=int, default=400_000, help="model token budget")
    p.add_argument("--workflow-id", help="default: news-scan-<since>-<until>")
    args = p.parse_args()
    if args.since > args.until:
        p.error("--since must not be after --until")
    return args


def _short(value: object, limit: int = 160) -> str:
    text = value if isinstance(value, str) else json.dumps(value)
    text = " ".join(text.split())
    return text if len(text) <= limit else text[: limit - 1] + "…"


def _print_event(item: Any) -> None:
    ev = item.event
    if ev.type == AgentEventType.TOOL_START:
        print(f"  -> {ev.tool_name} {_short(ev.tool_input)}")
    elif ev.type == AgentEventType.TOOL_END:
        print(f"  <- {ev.tool_name} {_short(ev.tool_output)}")
    elif ev.type == AgentEventType.TOOL_ERROR:
        print(f"  !! {ev.tool_name} {_short(ev.message)}")


async def run() -> int:
    args = _args()
    request = ScanRequest(
        since=args.since,
        until=args.until,
        source_ids=args.source,
        max_drafts=args.max_drafts,
        max_turns=args.max_turns,
        model=args.model,
        max_tokens=args.max_tokens,
    )
    workflow_id = args.workflow_id or f"news-scan-{args.since}-{args.until}"
    client = await Client.connect(
        **ClientConfig.load_client_connect_config(), plugins=[AgentHarnessPlugin()]
    )
    ui = os.environ.get("TEMPORAL_UI_URL", "http://localhost:8280")
    print(f"workflow {workflow_id}: {ui}/namespaces/{client.namespace}/workflows/{workflow_id}")

    try:
        report = await run_scan(client, request, workflow_id, on_event=_print_event)
    except ScanFailedError as exc:
        print(f"scan failed: {exc}", file=sys.stderr)
        return 1
    print(json.dumps(report.model_dump(), indent=2))
    return 0 if not report.error else 2


def main() -> None:
    sys.exit(asyncio.run(run()))


if __name__ == "__main__":
    main()
