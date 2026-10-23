# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""NewsDailyScan: the workflow the daily Temporal Schedule starts.

Each run scans the days since the previous successful run, up to yesterday (UTC). The cursor is
the last completion result of the schedule (``DailyScanResult.until``): a failed run does not
replace it, so the next run scans the missed days again, at most MAX_WINDOW_DAYS of them.
The scan itself is one NewsAgent session, run by the `news.run_scan` activity.
"""

from datetime import date, timedelta

from temporalio import workflow
from temporalio.common import RetryPolicy
from temporalio.exceptions import ApplicationError

with workflow.unsafe.imports_passed_through():
    from .models import DailyScanInput, DailyScanResult, RunScanInput, ScanReport, ScanRequest
    from .validation import MAX_WINDOW_DAYS


def next_window(today: date, last_until: str | None) -> tuple[date, date] | None:
    """The [since, until] to scan today after a cursor at ``last_until``, or None if none."""
    until = today - timedelta(days=1)
    since = date.fromisoformat(last_until) + timedelta(days=1) if last_until else until
    if since > until:
        return None
    return max(since, until - timedelta(days=MAX_WINDOW_DAYS - 1)), until


@workflow.defn(name="NewsDailyScan")
class NewsDailyScanWorkflow:
    @workflow.run
    async def run(self, inp: DailyScanInput) -> DailyScanResult:
        last = workflow.get_last_completion_result(DailyScanResult)
        window = next_window(workflow.now().date(), last.until if last else None)
        if window is None:
            # Only possible with a cursor: today's window was already scanned.
            return last
        since, until = window
        report = await workflow.execute_activity(
            "news.run_scan",
            RunScanInput(
                request=ScanRequest(
                    since=since.isoformat(),
                    until=until.isoformat(),
                    source_ids=inp.source_ids,
                    max_drafts=inp.max_drafts,
                ),
                workflow_id=f"news-scan-{since}-{until}",
            ),
            result_type=ScanReport,
            start_to_close_timeout=timedelta(minutes=45),
            heartbeat_timeout=timedelta(minutes=2),
            retry_policy=RetryPolicy(initial_interval=timedelta(minutes=1), maximum_attempts=2),
        )
        if report.error:
            # Failing keeps the previous completion result, so the cursor does not advance.
            raise ApplicationError(
                f"scan {since}..{until} failed: {report.error}", non_retryable=True
            )
        return DailyScanResult(
            until=until.isoformat(),
            scan_workflow_id=report.workflow_id,
            stored=len(report.stored),
        )
