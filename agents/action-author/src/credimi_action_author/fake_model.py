# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""A scripted stand-in for the LLM, so the whole stack runs without an API key.

It runs where the real model would (inside the model activity) and decides each step only from
the conversation it is given, so retries and crash-resume behave as with a real model. It
exercises the plumbing; it does not explore.

Script, for the latest user message (it needs an app id and "PIN <digits>" in it):

1. ``set_target`` with that app, slot ``onboarding`` and the PIN;
2. ``open`` the app and ``snapshot`` it (agent-device over MCP);
3. ``maestro_cheat_sheet``, then ``maestro_inspect_screen``;
4. ``check_action`` with an external PIN reference (the check must refuse it);
5. ``check_action`` again with the supplied PIN embedded, then ``submit_action``;
6. an answer quoting the submit_action reply; ordinary prose is masked, while the accepted
   action is shown verbatim with its literal test PIN.

The action is the FCAF onboarding flow, with the supplied PIN embedded.
"""

from __future__ import annotations

import json
import re
from collections.abc import AsyncIterator
from typing import Any

from agents import Model, ModelProvider, ModelResponse, Usage
from openai.types.responses import (
    ResponseFunctionToolCall,
    ResponseOutputMessage,
    ResponseOutputText,
)

ONBOARDING_ACTION = """\
appId: {app_id}
name: Onboarding with a new PIN
tags:
  - onboarding
---
- launchApp:
    clearState: false
    permissions:
      camera: allow
      notifications: allow
      all: unset
- extendedWaitUntil:
    visible: "Welcome to your Wallet"
    timeout: 30000
- tapOn:
    point: "11%,42%"
- inputText: "{pin}"
- hideKeyboard
- extendedWaitUntil:
    visible: "Confirm PIN"
    timeout: 15000
- tapOn:
    point: "11%,42%"
- inputText: "{pin}"
- hideKeyboard
- extendedWaitUntil:
    visible:
      id: "{app_id}:id/dashboard_screen_bottom_navigation_item_home"
    timeout: 30000
- takeScreenshot: onboarding-complete
"""

_APP_ID = re.compile(r"\b([a-z][a-z0-9_]*(?:\.[a-z0-9_]+){2,})\b")
_PIN = re.compile(r"\bPIN\s*(?:is\s*|=\s*|:\s*)?(\d{4,})", re.IGNORECASE)


def onboarding_action(app_id: str, pin: str) -> str:
    return ONBOARDING_ACTION.format(app_id=app_id, pin=pin)


def _get(item: Any, key: str) -> Any:
    return item.get(key) if isinstance(item, dict) else getattr(item, key, None)


def _response(output: Any) -> ModelResponse:
    usage = Usage(requests=1, input_tokens=1000, output_tokens=100, total_tokens=1100)
    return ModelResponse(output=[output], usage=usage, response_id=None)


def _text(text: str) -> ModelResponse:
    return _response(
        ResponseOutputMessage(
            id="msg_final",
            content=[ResponseOutputText(text=text, annotations=[], type="output_text")],
            role="assistant",
            status="completed",
            type="message",
        )
    )


def _content(item: Any) -> str:
    content = _get(item, "content")
    if isinstance(content, str):
        return content
    return " ".join(str(_get(c, "text") or "") for c in content or [])


class _Conversation:
    """The calls made since the latest user message, with their outputs."""

    def __init__(self, items: list[Any]) -> None:
        users = [i for i, item in enumerate(items) if _get(item, "role") == "user"]
        start = users[-1] if users else 0
        self.user = _content(items[start]) if users else ""
        outputs = {
            _get(i, "call_id"): str(_get(i, "output") or "")
            for i in items
            if _get(i, "type") == "function_call_output"
        }
        # Call ids must stay unique across the whole conversation.
        self.next_id = sum(1 for i in items if _get(i, "type") == "function_call") + 1
        self.calls = [
            (_get(c, "name"), outputs.get(_get(c, "call_id"), ""))
            for c in items[start:]
            if _get(c, "type") == "function_call"
        ]

    def call(self, name: str, args: dict[str, Any]) -> ModelResponse:
        n = self.next_id
        return _response(
            ResponseFunctionToolCall(
                arguments=json.dumps(args),
                call_id=f"call_{n}",
                name=name,
                type="function_call",
                id=f"fc_{n}",
                status="completed",
            )
        )

    @property
    def last(self) -> tuple[str, str]:
        return self.calls[-1] if self.calls else ("", "")


def next_response(items: list[Any]) -> ModelResponse:
    conv = _Conversation(items)
    app = _APP_ID.search(conv.user)
    pin = _PIN.search(conv.user)
    if not app or not pin:
        return _text(
            "Scripted model: name the app id and the PIN, e.g. `write the onboarding action "
            "for eu.europa.ec.euidi with PIN 123456`."
        )
    app_id, pin_value = app.group(1), pin.group(1)
    name, out = conv.last
    request = conv.user.split("\n\nCurrent authoring state:", 1)[0]
    requested_name = re.search(
        r'(?:write\s+action|(?:store|save)\s+(?:it\s+)?as)\s+["\']([^"\']+)["\']',
        request,
        re.IGNORECASE,
    )
    if not conv.calls:
        return conv.call(
            "set_target",
            {
                "app_id": app_id,
                "slot": "onboarding",
                "parameters": [{"name": "PIN", "value": pin_value}],
                **({"action_name": requested_name.group(1)} if requested_name else {}),
            },
        )
    if out.startswith("ERROR"):
        return _text(f"Scripted model stops: {name} failed: {out[:500]}")
    steps = {
        "set_target": ("open", {"app": app_id}),
        "open": ("snapshot", {}),
        "snapshot": ("maestro_cheat_sheet", {}),
        "maestro_cheat_sheet": ("maestro_inspect_screen", {}),
        "maestro_inspect_screen": ("check_action", {"yaml": onboarding_action(app_id, "${PIN}")}),
    }
    if name in steps:
        return conv.call(*steps[name])
    if name == "check_action" and out.startswith("PROBLEMS"):
        return conv.call("check_action", {"yaml": onboarding_action(app_id, pin_value)})
    if name == "check_action" and out.startswith("OK"):
        return conv.call("submit_action", {"yaml": onboarding_action(app_id, pin_value)})
    return _text(f"Done with PIN {pin_value}. {name} replied:\n{out[:1500]}")


class ScriptedModel(Model):
    async def get_response(
        self,
        system_instructions: str | None,
        input: str | list[Any],  # noqa: A002
        model_settings: Any,
        tools: list[Any],
        *args: Any,
        **kwargs: Any,
    ) -> ModelResponse:
        items = input if isinstance(input, list) else [{"role": "user", "content": input}]
        return next_response(items)

    def stream_response(self, *args: Any, **kwargs: Any) -> AsyncIterator[Any]:
        raise NotImplementedError("the action author runs the model without streaming")


class ScriptedModelProvider(ModelProvider):
    def get_model(self, model_name: str | None) -> Model:
        return ScriptedModel()
