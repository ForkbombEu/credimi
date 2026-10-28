# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""The agent's own tools (the agent-device tools come from MCP, see device.py).

They run inline in the workflow, read and update the workflow-owned ``AuthorSession`` (injected
per call, hidden from the model) and delegate all device I/O to activities.

All of them are inherently safe for this PoC, so none waits for a human approval:
`reset_app` and `submit_action` wipe only the app under test, on a dedicated test device.

No ``from __future__ import annotations``: the harness reads tool annotations concretely to
build the model's schema and to find ``Injected`` parameters.
"""

from dataclasses import dataclass, field
from datetime import timedelta
from typing import Any

from temporalio import workflow
from temporalio.common import RetryPolicy
from temporalio.exceptions import ActivityError

with workflow.unsafe.imports_passed_through():
    from temporal_agent_harness.harness import agent

    from .activities import CHEAT_SHEET, INSPECT_SCREEN, REPLAY_ACTION, RESET_APP
    from .models import (
        MAX_SUBMISSIONS,
        SLOTS,
        Param,
        ReplayRequest,
        ReplayResult,
        ScreenRequest,
        Usage,
    )
    from .validation import APP_ID, redact, validate_action

DEVICE_TIMEOUT = timedelta(minutes=4)
# Two `maestro test` runs of at most 5 minutes each, plus clearing and inspecting.
REPLAY_TIMEOUT = timedelta(minutes=15)
HEARTBEAT_TIMEOUT = timedelta(minutes=1)
NO_RETRY = RetryPolicy(maximum_attempts=1)
RETRY = RetryPolicy(initial_interval=timedelta(seconds=2), maximum_attempts=2)


@dataclass
class AuthorSession:
    """Workflow-owned state of one chat; rebuilt deterministically on replay."""

    serial: str
    token_budget: int
    max_model_calls: int
    bound_app_id: str = ""
    bound_slot: str = ""
    apk_sha256: str = ""
    accepted_replays: int = 0
    app_id: str = ""
    slot: str = ""
    params: dict[str, str] = field(default_factory=dict)
    attempts: int = 0
    accepted_yaml: str = ""
    accepted_path: str = ""
    usage: Usage = field(default_factory=Usage)
    # Shared with the MCP reference: update in place so target changes take effect
    # within the same model/tool turn, not only on the next user message.
    device_context: dict[str, Any] = field(default_factory=dict)

    def mask(self, text: str) -> str:
        return redact(text, self.params)

    def describe(self) -> str:
        if not self.app_id:
            return "No target yet: call `set_target` first."
        names = ", ".join(self.params) or "none"
        state = (
            f"accepted, saved to {self.accepted_path}"
            if self.accepted_yaml
            else f"{MAX_SUBMISSIONS - self.attempts} of {MAX_SUBMISSIONS} submit_action calls left"
        )
        return (
            f"Target: app `{self.app_id}`, slot `{self.slot}`, "
            f"input values provided: {names}; {state}."
        )


def _failure(exc: ActivityError) -> str:
    cause = exc.cause
    return f"ERROR: {cause.message if cause is not None and hasattr(cause, 'message') else exc}"


def _contract(slot: str) -> str:
    s = SLOTS[slot]
    return (
        f"Slot `{s.name}` (category {s.category}). Precondition: {s.precondition}. "
        f"Outcome: {s.outcome}. Required input values: {', '.join(s.required_params)}."
    )


@agent.tool_defn(inherently_safe=True)
async def set_target(
    app_id: str, slot: str, parameters: list[Param], session: agent.Injected[AuthorSession]
) -> str:
    """Set the app, the action slot and the parameter values the user gave. Call it first.

    Args:
        app_id: Android package of the wallet, e.g. "eu.europa.ec.euidi".
        slot: Action slot, e.g. "onboarding".
        parameters: Every parameter value the user gave, e.g. [{"name": "PIN", "value": "..."}].
    """
    app_id, slot = app_id.strip(), slot.strip()
    if session.bound_app_id and app_id != session.bound_app_id:
        return f"ERROR: selected Credimi APK is {session.bound_app_id}; cannot target another app"
    if session.bound_slot and slot != session.bound_slot:
        return f"ERROR: this session is bound to slot {session.bound_slot}"
    if not APP_ID.match(app_id):
        return f"ERROR: {app_id!r} is not an Android package name"
    if slot not in SLOTS:
        return f"ERROR: unknown slot {slot!r}; known slots: {sorted(SLOTS)}"
    values = {p.name.strip(): p.value for p in parameters}
    contract = SLOTS[slot]
    if missing := [n for n in contract.required_params if not values.get(n)]:
        return f"ERROR: missing values for {missing}; ask the user for them"
    if unknown := sorted(set(values) - set(contract.required_params)):
        return f"ERROR: unknown parameters {unknown}; slot {slot} takes {contract.required_params}"
    if (app_id, slot, values) != (session.app_id, session.slot, session.params):
        session.app_id, session.slot, session.params = app_id, slot, values
        session.attempts, session.accepted_yaml, session.accepted_path = 0, "", ""
        session.accepted_replays = 0
    session.device_context.update(app_id=session.app_id, params=session.params)
    return f"OK. {session.describe()}\n{_contract(slot)}"


@agent.tool_defn(inherently_safe=True)
async def maestro_cheat_sheet() -> str:
    """Maestro flow syntax reference: every command with its parameters. Read it before writing
    an action."""
    try:
        return await workflow.execute_activity(
            CHEAT_SHEET,
            start_to_close_timeout=DEVICE_TIMEOUT,
            retry_policy=RETRY,
            result_type=str,
        )
    except ActivityError as exc:
        return _failure(exc)


@agent.tool_defn(inherently_safe=True)
async def maestro_inspect_screen(session: agent.Injected[AuthorSession]) -> str:
    """The current screen as Maestro sees it: one line per element it can match, with bounds,
    id (resource-id), text, desc (content-desc) and hint. Use these for selectors.

    It closes the agent-device session first; call `open` again before more agent-device tools.
    """
    try:
        return await workflow.execute_activity(
            INSPECT_SCREEN,
            ScreenRequest(app_id=session.app_id, params=session.params),
            start_to_close_timeout=DEVICE_TIMEOUT,
            heartbeat_timeout=HEARTBEAT_TIMEOUT,
            retry_policy=RETRY,
            result_type=str,
        )
    except ActivityError as exc:
        return _failure(exc)


@agent.tool_defn(inherently_safe=True)
async def reset_app(session: agent.Injected[AuthorSession]) -> str:
    """Clear the app's data so it starts as freshly installed (the slot precondition).

    It closes the agent-device session; call `open` again to explore from the first screen.
    """
    if not session.app_id:
        return "ERROR: call set_target first"
    try:
        out = await workflow.execute_activity(
            RESET_APP,
            ScreenRequest(app_id=session.app_id),
            start_to_close_timeout=DEVICE_TIMEOUT,
            retry_policy=RETRY,
            result_type=str,
        )
    except ActivityError as exc:
        return _failure(exc)
    return f"OK: {session.app_id} data cleared ({out}). Call `open` to start it."


def _problems(yaml: str, session: AuthorSession) -> list[str]:
    if not session.app_id:
        return ["call set_target first"]
    return validate_action(yaml, session.app_id, session.slot, session.params)


@agent.tool_defn(inherently_safe=True)
async def check_action(yaml: str, session: agent.Injected[AuthorSession]) -> str:
    """Check a Maestro action without running it. Replies OK or lists every problem to fix.

    Args:
        yaml: The whole flow: header (appId, name, tags), `---`, then the command list.
    """
    if problems := _problems(yaml, session):
        return session.mask("PROBLEMS:\n" + "\n".join(f"- {p}" for p in problems))
    return "OK: the action passes every check. Call submit_action with the same YAML."


def _replay_report(result: ReplayResult) -> str:
    lines = []
    for i, run in enumerate(result.runs, 1):
        if run.passed:
            lines.append(f"Replay {i}: passed in {run.seconds}s ({run.completed} commands).")
            continue
        lines += [
            f"Replay {i}: FAILED after {run.seconds}s at: {run.failed_command}",
            "Last output lines:",
            run.tail,
            "Screen at the failure (Maestro view):",
            run.screen,
        ]
    return "\n".join(lines)


@agent.tool_defn(inherently_safe=True)
async def submit_action(yaml: str, session: agent.Injected[AuthorSession]) -> str:
    """Submit the action: it is checked, then replayed twice on the device, each time from
    cleared app data, with the real parameter values. Both replays must pass.

    Args:
        yaml: The same YAML that check_action accepted.
    """
    if not session.app_id:
        return "REJECTED: call set_target first"
    if session.accepted_yaml:
        return f"REJECTED: an action was already accepted ({session.accepted_path})"
    if session.attempts >= MAX_SUBMISSIONS:
        return f"REJECTED: the budget of {MAX_SUBMISSIONS} submissions is spent; stop here"
    session.attempts += 1
    attempt = f"(attempt {session.attempts} of {MAX_SUBMISSIONS})"
    if problems := _problems(yaml, session):
        return session.mask(
            f"REJECTED {attempt} before replay:\n" + "\n".join(f"- {p}" for p in problems)
        )
    try:
        result: ReplayResult = await workflow.execute_activity(
            REPLAY_ACTION,
            ReplayRequest(
                app_id=session.app_id,
                slot=session.slot,
                yaml=yaml,
                params=session.params,
                apk_sha256=session.apk_sha256,
            ),
            start_to_close_timeout=REPLAY_TIMEOUT,
            heartbeat_timeout=HEARTBEAT_TIMEOUT,
            retry_policy=NO_RETRY,
            result_type=ReplayResult,
        )
    except ActivityError as exc:
        return f"REJECTED {attempt}: replay could not run: {_failure(exc)}"
    report = session.mask(_replay_report(result))
    if not result.accepted or len(result.runs) != 2 or not all(run.passed for run in result.runs):
        return f"REJECTED {attempt}: the replay failed.\n{report}"
    session.accepted_yaml, session.accepted_path = yaml, result.path
    session.accepted_replays = 2
    return f"ACCEPTED {attempt}: saved to {result.path}.\n{report}"


TOOLS = [
    set_target,
    maestro_cheat_sheet,
    maestro_inspect_screen,
    reset_app,
    check_action,
    submit_action,
]
