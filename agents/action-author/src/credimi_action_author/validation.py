# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Checks a candidate Maestro wallet action without running it, and masks parameter values.

``validate_action`` returns every problem it finds (empty list: valid):

1. the YAML is exactly two documents: a header mapping and a command list;
2. the header keys are a subset of {appId, name, tags, env} and ``appId`` is the session's app;
3. every command, nested ones included (runFlow, retry, repeat), is on the allowlist; runFlow
   and retry carry inline ``commands`` only, openLink never opens an app store, launchApp and
   setPermissions never target another app;
4. at least one ``takeScreenshot``;
5. every required input value is embedded as a quoted literal ``inputText`` value, not an
   external parameter or a mention in comments/display metadata;
6. the last top-level command other than ``takeScreenshot`` asserts the outcome
   (``assertVisible``, or ``extendedWaitUntil`` with ``visible``).
"""

from __future__ import annotations

import re
from collections.abc import Iterator
from typing import Any

import yaml

from .models import SLOTS

HEADER_KEYS = {"appId", "name", "tags", "env"}
ALLOWED_COMMANDS = {
    "launchApp",
    "tapOn",
    "doubleTapOn",
    "longPressOn",
    "inputText",
    "eraseText",
    "hideKeyboard",
    "pressKey",
    "back",
    "swipe",
    "scroll",
    "scrollUntilVisible",
    "assertVisible",
    "assertNotVisible",
    "extendedWaitUntil",
    "waitForAnimationToEnd",
    "takeScreenshot",
    "runFlow",
    "retry",
    "repeat",
    "setPermissions",
    "openLink",
    "copyTextFrom",
    "pasteText",
}
# Commands whose nested `commands` are walked.
NESTED = {"runFlow", "retry", "repeat"}
STORE_LINK = re.compile(
    r"^\s*(market:|itms-apps:|itms:|https?://(play\.google\.com|apps\.apple\.com|itunes\.apple\.com))",
    re.IGNORECASE,
)
APP_ID = re.compile(r"^[A-Za-z][A-Za-z0-9_]*(\.[A-Za-z0-9_]+)+$")


def _command(item: Any) -> tuple[str, Any] | None:
    """(name, argument) of one list item: `- back` or `- tapOn: ...`."""
    if isinstance(item, str):
        return item, None
    if isinstance(item, dict) and len(item) == 1:
        name, arg = next(iter(item.items()))
        if isinstance(name, str):
            return name, arg
    return None


def _walk(commands: list[Any], path: str, problems: list[str]) -> Iterator[tuple[str, Any, str]]:
    """Every command, nested ones included, as (name, argument, location)."""
    for i, item in enumerate(commands):
        where = f"{path}[{i}]"
        cmd = _command(item)
        if cmd is None:
            problems.append(f"{where}: not a command (use `- name` or `- name: argument`)")
            continue
        name, arg = cmd
        yield name, arg, where
        if name not in NESTED:
            continue
        if not isinstance(arg, dict):
            problems.append(f"{where} {name}: needs a mapping with inline `commands`")
            continue
        if "file" in arg:
            problems.append(f"{where} {name}: `file` is not allowed; use inline `commands`")
        nested = arg.get("commands")
        if not isinstance(nested, list) or not nested:
            problems.append(f"{where} {name}: needs a non-empty inline `commands` list")
            continue
        yield from _walk(nested, f"{where}.{name}.commands", problems)


def _link(arg: Any) -> str:
    if isinstance(arg, dict):
        arg = arg.get("link", "")
    return arg if isinstance(arg, str) else ""


def _check_command(name: str, arg: Any, where: str, app_id: str, problems: list[str]) -> None:
    if name not in ALLOWED_COMMANDS:
        problems.append(f"{where}: command `{name}` is not allowed")
        return
    if name == "openLink" and STORE_LINK.match(_link(arg)):
        problems.append(f"{where}: openLink must not open an app store ({_link(arg)!r})")
    if name in ("launchApp", "setPermissions") and isinstance(arg, dict):
        other = arg.get("appId")
        if other is not None and other != app_id:
            problems.append(f"{where}: {name} targets {other!r}, not the app under test")
    if name == "launchApp" and isinstance(arg, str) and arg != app_id:
        problems.append(f"{where}: launchApp targets {arg!r}, not the app under test")


def _is_outcome_assertion(name: str, arg: Any) -> bool:
    if name == "assertVisible":
        return True
    return name == "extendedWaitUntil" and isinstance(arg, dict) and "visible" in arg


def _parameter_refs(value: Any) -> set[str]:
    """References in executable values, not YAML comments, metadata or mapping keys."""
    if isinstance(value, str):
        return set(re.findall(r"\$\{([A-Za-z_][A-Za-z0-9_]*)\}", value))
    children = (
        value.values() if isinstance(value, dict) else value if isinstance(value, list) else ()
    )
    refs: set[str] = set()
    for child in children:
        refs.update(_parameter_refs(child))
    return refs


def validate_action(text: str, app_id: str, slot: str, params: dict[str, str]) -> list[str]:
    """Every reason the action cannot be accepted (empty list: valid)."""
    try:
        docs = list(yaml.safe_load_all(text))
    except yaml.YAMLError as exc:
        return [f"invalid YAML: {exc}"]
    if len(docs) != 2:
        return [f"expected two YAML documents (header, `---`, command list), found {len(docs)}"]
    header, commands = docs
    problems: list[str] = []
    if not isinstance(header, dict):
        problems.append("the first document must be the header mapping (appId, ...)")
        header = {}
    if not isinstance(commands, list) or not commands:
        problems.append("the second document must be a non-empty list of commands")
        commands = []

    if any(not isinstance(key, str) for key in header):
        problems.append("header keys must be strings")
    if extra := sorted(key for key in header if isinstance(key, str) and key not in HEADER_KEYS):
        problems.append(f"header keys not allowed: {extra}; allowed: {sorted(HEADER_KEYS)}")
    if header.get("appId") != app_id:
        problems.append(f"header appId must be {app_id!r}, found {header.get('appId')!r}")

    screenshots = 0
    literal_inputs: list[Any] = []
    for name, arg, where in _walk(commands, "commands", problems):
        _check_command(name, arg, where, app_id, problems)
        screenshots += name == "takeScreenshot"
        if name == "inputText":
            literal_inputs.append(arg.get("text") if isinstance(arg, dict) else arg)
    if screenshots == 0:
        problems.append("add at least one `takeScreenshot` (after the final assertion)")

    contract = SLOTS.get(slot)
    references = _parameter_refs([commands, header.get("env", {})])
    for name in contract.required_params if contract else ():
        if not params.get(name) or params[name] not in literal_inputs:
            problems.append(f"embed the supplied {name} as a quoted literal inputText value")
        if name in references:
            problems.append(f"{name} must be embedded in the action, not an external parameter")

    last = [c for c in (_command(item) for item in commands) if c and c[0] != "takeScreenshot"]
    if not last or not _is_outcome_assertion(*last[-1]):
        problems.append(
            "the last command before the final takeScreenshot must assert the outcome: "
            "`assertVisible`, or `extendedWaitUntil` with `visible`"
        )
    return problems


def redact(text: str, params: dict[str, str]) -> str:
    """Replace every parameter value with ``${NAME}``, longest values first."""
    for name, value in sorted(params.items(), key=lambda p: -len(p[1])):
        if value:
            text = text.replace(value, "${" + name + "}")
    return text
