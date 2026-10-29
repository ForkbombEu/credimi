# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Typed contracts shared by the workflow, its tools, the activities and the scripted model.

Models the LLM fills in (``Param``) declare no defaults, so the OpenAI strict JSON schema keeps
every field required.
"""

from __future__ import annotations

from dataclasses import dataclass

from pydantic import BaseModel

# The worker polls this queue in the selected wallet owner's organization namespace.
TASK_QUEUE = "action-author"
WORKFLOW_TYPE = "ActionAuthor"
DEFAULT_MODEL = "gpt-5.1"

# Name of the agent-device MCP provider registered on the worker (StatelessMCPServerProvider).
AGENT_DEVICE_MCP = "agent-device"
# agent-device session used by the model's MCP calls and closed by every Maestro tool.
AGENT_DEVICE_SESSION = "action-author"

# Chat (`ask`) limits. A session takes at most CHAT_MAX_MESSAGES messages, which also bounds its
# Temporal history (the harness does not continue-as-new).
CHAT_MAX_MESSAGES = 20
# Defaults of the per-`ask` budgets; the worker environment can override them
# (ACTION_AUTHOR_TOKEN_BUDGET, ACTION_AUTHOR_MAX_MODEL_CALLS).
DEFAULT_TOKEN_BUDGET = 600_000
DEFAULT_MAX_MODEL_CALLS = 80
# submit_action calls allowed per target (app id, slot and parameter values) or after a refusal.
MAX_SUBMISSIONS = 3
# Successful replays from a cleared app needed to accept an action.
REPLAY_RUNS = 2


@dataclass(frozen=True)
class Slot:
    """The contract of one wallet action slot: what the action starts from and must reach."""

    name: str
    category: str
    precondition: str
    outcome: str
    required_params: tuple[str, ...]


SLOTS: dict[str, Slot] = {
    "onboarding": Slot(
        name="onboarding",
        category="onboarding",
        precondition="app data cleared (`pm clear`): the app starts as freshly installed",
        outcome="the wallet home screen is visible, with the PIN set",
        required_params=("PIN",),
    ),
}


class Param(BaseModel):
    """One action parameter, e.g. name "PIN", value "123456"."""

    name: str
    value: str


class Usage(BaseModel):
    requests: int = 0
    input_tokens: int = 0
    output_tokens: int = 0
    cache_read_tokens: int = 0
    cache_write_tokens: int = 0
    context_cleared_tokens: int = 0

    def add(self, other: Usage) -> Usage:
        return Usage(
            requests=self.requests + other.requests,
            input_tokens=self.input_tokens + other.input_tokens,
            output_tokens=self.output_tokens + other.output_tokens,
            cache_read_tokens=self.cache_read_tokens + other.cache_read_tokens,
            cache_write_tokens=self.cache_write_tokens + other.cache_write_tokens,
            context_cleared_tokens=self.context_cleared_tokens + other.context_cleared_tokens,
        )

    def text(self) -> str:
        text = (
            f"{self.requests} model calls, {self.input_tokens}/{self.output_tokens} tokens in/out"
        )
        if self.cache_read_tokens or self.cache_write_tokens:
            text += f", cache read/write {self.cache_read_tokens}/{self.cache_write_tokens}"
        if self.context_cleared_tokens:
            text += f", context cleared {self.context_cleared_tokens} tokens"
        return text


class DeviceInfo(BaseModel):
    """The worker's device and budgets, read once per session from the worker's environment."""

    serial: str
    token_budget: int = DEFAULT_TOKEN_BUDGET
    max_model_calls: int = DEFAULT_MAX_MODEL_CALLS


class ScreenRequest(BaseModel):
    """Input of the activities that touch the device."""

    app_id: str = ""


class ReplayRequest(BaseModel):
    """Self-contained YAML replayed on the device against the installed build."""

    app_id: str
    slot: str
    yaml: str
    apk_sha256: str
    runs: int = REPLAY_RUNS


class RunOutcome(BaseModel):
    """One `maestro test` run of a candidate action."""

    passed: bool
    seconds: float
    completed: int = 0
    failed_command: str = ""
    tail: str = ""
    screen: str = ""


class ReplayResult(BaseModel):
    accepted: bool
    runs: list[RunOutcome]
    path: str = ""
