# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Pure helpers for Maestro output: compact view hierarchies and per-command test results."""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from typing import Any

# Maestro MCP prepends this notice to its replies; it is noise for the model.
_VIEWER_NOTICE = re.compile(r"^Important: Maestro Viewer is available at .*$", re.MULTILINE)
_RESULT_LINE = re.compile(r"^(?P<indent>\s*)(?P<command>.+?)\.\.\. (?P<status>[A-Z]+)\s*$")
_STATUSES = {"COMPLETED", "FAILED", "SKIPPED", "WARNED"}
# JVM warnings Maestro prints on every start.
_NOISE = re.compile(r"^(WARNING:|kotlin-logging:|SLF4J)")


def strip_viewer_notice(text: str) -> str:
    return _VIEWER_NOTICE.sub("", text).strip()


def _node_line(node: dict[str, Any], defaults: dict[str, Any]) -> str | None:
    """One line for a node a selector can target, or None."""

    def get(key: str) -> Any:
        return node.get(key, defaults.get(key, ""))

    text, rid, a11y, hint = get("txt"), get("rid"), get("a11y"), get("hint")
    clickable = bool(get("clickable"))
    if not (text or rid or a11y or hint or clickable):
        return None
    parts = [get("b") or "[?]"]
    if rid:
        parts.append(f"id={rid}")
    if text:
        parts.append(f"text={json.dumps(text, ensure_ascii=False)}")
    if a11y:
        parts.append(f"desc={json.dumps(a11y, ensure_ascii=False)}")
    if hint:
        parts.append(f"hint={json.dumps(hint, ensure_ascii=False)}")
    flags = [
        name
        for name in ("clickable", "focused", "selected", "checked", "scrollable")
        if bool(node.get("scroll" if name == "scrollable" else name, defaults.get(name)))
    ]
    if flags:
        parts.append(",".join(flags))
    if node.get("enabled", defaults.get("enabled", True)) is False:
        parts.append("disabled")
    return " ".join(parts)


def compact_hierarchy(raw: str, limit: int = 8_000) -> str:
    """The targetable nodes of a Maestro ``inspect_screen`` reply, one line each with bounds.

    Keeps nodes with text, resource-id, content-desc, hint or clickable; everything else
    (layout containers) is dropped. The result is capped at ``limit`` characters.
    """
    raw = strip_viewer_notice(raw)
    try:
        data = json.loads(raw)
    except json.JSONDecodeError:
        return raw[:limit]
    defaults = (data.get("ui_schema") or {}).get("defaults") or {}
    lines: list[str] = []

    def walk(node: Any) -> None:
        if not isinstance(node, dict):
            return
        if line := _node_line(node, defaults):
            lines.append(line)
        for child in node.get("c") or []:
            walk(child)

    for element in data.get("elements") or []:
        walk(element)
    text = "\n".join(lines) if lines else "(no targetable elements)"
    if len(text) > limit:
        text = text[:limit] + "\n[... truncated]"
    return text


@dataclass
class CommandResult:
    command: str
    status: str
    depth: int


@dataclass
class RunOutput:
    """Per-command results of one ``maestro test`` run."""

    results: list[CommandResult] = field(default_factory=list)

    @property
    def failed(self) -> CommandResult | None:
        return next((r for r in self.results if r.status == "FAILED"), None)

    @property
    def completed(self) -> int:
        return sum(r.status == "COMPLETED" for r in self.results)


def parse_run_output(output: str) -> RunOutput:
    """Collect the ``<command>... COMPLETED|FAILED|SKIPPED|WARNED`` lines of ``maestro test``."""
    out = RunOutput()
    for line in output.splitlines():
        m = _RESULT_LINE.match(line)
        if m and m.group("status") in _STATUSES:
            out.results.append(
                CommandResult(
                    command=m.group("command").strip(),
                    status=m.group("status"),
                    depth=len(m.group("indent")) // 2,
                )
            )
    return out


def output_tail(output: str, lines: int = 30) -> str:
    """The last ``lines`` lines of Maestro output, without JVM noise and blank lines."""
    kept = [line for line in output.splitlines() if line.strip() and not _NOISE.match(line)]
    return "\n".join(kept[-lines:])
