# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""A scripted stand-in for the LLM, so the whole stack runs without an API key.

It runs where the real model would (inside the model activity) and decides each step only from
the conversation it is given, so retries and crash-resume behave as with a real model. It
exercises the plumbing; it does not judge newsworthiness.

Scan script (the tools offered include ``submit_draft``):

1. ``list_changes`` on the first source of the request, over the scan window;
2. ``get_pull_request`` on the first merged pull request listed;
3. ``search_news`` for its title;
4. ``submit_draft`` citing an evidence id that does not exist (the workflow must reject it);
5. ``submit_draft`` again, correctly cited, with a verbatim quote;
6. the final ``ScanDecisions``.

Chat script (the tools offered include ``save_draft``), for the latest user message. It needs a
source id and two YYYY-MM-DD dates in the message, e.g. "openid4vp 2026-10-01 2026-10-14":

1. ``list_changes`` on that source and window, then ``get_pull_request`` on its first merged
   pull request, and an answer listing what it found;
2. only if the message contains "draft": ``search_news``; ``check_draft`` without the required
   ``quotes`` field (as small local models do; the error must come back as the tool's reply),
   then ``check_draft`` again with the full draft; ``save_draft`` (which waits for a human
   approval in the harness UI); and an answer saying how it went.

When a step has nothing to work on it answers straight away.
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


def _get(item: Any, key: str) -> Any:
    return item.get(key) if isinstance(item, dict) else getattr(item, key, None)


def _response(output: Any) -> ModelResponse:
    return ModelResponse(output=[output], usage=Usage(requests=1), response_id=None)


def _call(n: int, name: str, args: dict[str, Any]) -> ModelResponse:
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


def _final(decisions: list[dict[str, str]], summary: str) -> ModelResponse:
    return _text(json.dumps({"decisions": decisions, "summary": summary}))


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
        all_calls = [i for i in items if _get(i, "type") == "function_call"]
        # Call ids must stay unique across the whole conversation.
        self.next_id = len(all_calls) + 1
        self.calls = [
            (_get(c, "name"), outputs.get(_get(c, "call_id"), ""))
            for c in items[start:]
            if _get(c, "type") == "function_call"
        ]

    def call(self, name: str, args: dict[str, Any]) -> ModelResponse:
        return _call(self.next_id, name, args)

    @property
    def last(self) -> tuple[str, str]:
        return self.calls[-1] if self.calls else ("", "")

    def output_of(self, name: str) -> str:
        return next((o for nm, o in self.calls if nm == name), "")


def _first_pull_request(listed_out: str) -> tuple[str, dict[str, Any]] | None:
    listed = json.loads(listed_out)
    pr = next((i for i in listed.get("items", []) if i["kind"] == "pull_request"), None)
    return (listed["repo"], pr) if pr else None


def _draft(pr_ev: dict[str, Any], evidence_id: str) -> dict[str, Any]:
    title = pr_ev["title"].split(": ", 1)[-1]
    return {
        "draft": {
            "title": f"Merged: {title}"[:120],
            "summary": f"A pull request was merged: **{title}**.",
            "body": f"The pull request [{pr_ev['title']}]({pr_ev['url']}) was merged.\n\n"
            "<script>alert('raw html must be escaped')</script>",
            "key_points": [
                {"text": f"{pr_ev['title']} was merged.", "evidence_ids": [evidence_id]}
            ],
            "quotes": [{"text": title[:60], "evidence_id": pr_ev["evidence_id"]}],
            "tags": ["scripted", "poc"],
            "refer": pr_ev["url"].split("/pull/")[0],
            "diff": pr_ev["url"],
            "rationale": "Scripted run: exercises the pipeline, not a news judgement.",
        }
    }


def _scan(conv: _Conversation) -> ModelResponse:
    name, out = conv.last
    if not conv.calls:
        source = re.search(r"source `([^`]+)`", conv.user)
        window = re.search(r"Scan window: (\S+) to (\S+)", conv.user)
        if not source or not window:
            return _final([], "No sources in the request.")
        return conv.call(
            "list_changes",
            {"source_id": source.group(1), "since": window.group(1), "until": window.group(2)},
        )
    if name == "list_changes":
        try:
            found = _first_pull_request(out)
        except (json.JSONDecodeError, KeyError):
            return _final([], f"list_changes failed: {out[:200]}")
        if found is None:
            return _final([], "No merged pull request in the window; nothing to draft.")
        repo, pr = found
        return conv.call("get_pull_request", {"repo": repo, "number": int(pr["ref"])})

    pr_out = conv.output_of("get_pull_request")
    try:
        pr_ev = json.loads(pr_out)
    except json.JSONDecodeError:
        return _final([], f"get_pull_request failed: {pr_out[:200]}")
    if name == "get_pull_request":
        return conv.call("search_news", {"query": pr_ev["title"].split(": ", 1)[-1][:60]})
    submits = [o for nm, o in conv.calls if nm == "submit_draft"]
    if name == "search_news":
        return conv.call("submit_draft", _draft(pr_ev, "ev_does_not_exist"))
    if name == "submit_draft" and len(submits) == 1 and "already covers" not in out:
        return conv.call("submit_draft", _draft(pr_ev, pr_ev["evidence_id"]))

    ref = pr_ev["evidence_id"]
    if "already covers" in out:
        return _final(
            [{"ref": ref, "decision": "duplicate", "rationale": out}],
            "The only candidate is already covered by an existing news record.",
        )
    if out.startswith("ACCEPTED"):
        return _final(
            [{"ref": ref, "decision": "draft", "rationale": "Scripted draft."}],
            "Scripted scan: one draft accepted after one rejected submission.",
        )
    return _final([{"ref": ref, "decision": "skip", "rationale": out}], "Draft was rejected.")


def _chat(conv: _Conversation, instructions: str) -> ModelResponse:
    name, out = conv.last
    if not conv.calls:
        known = re.findall(r"source `([^`]+)`", instructions)
        source = next((s for s in known if s in conv.user.lower()), None)
        dates = re.findall(r"\d{4}-\d{2}-\d{2}", conv.user)
        if source is None or len(dates) < 2:
            return _text(
                "Scripted model: name a source and two dates, e.g. "
                f"`{known[0] if known else 'openid4vp'} 2026-10-01 2026-10-14`, and add "
                "`draft` to save a news draft."
            )
        return conv.call(
            "list_changes", {"source_id": source, "since": dates[0], "until": dates[1]}
        )
    if name == "list_changes":
        try:
            found = _first_pull_request(out)
        except (json.JSONDecodeError, KeyError):
            return _text(f"I could not list the changes: {out[:300]}")
        if found is None:
            return _text("No merged pull request in that window.")
        repo, pr = found
        return conv.call("get_pull_request", {"repo": repo, "number": int(pr["ref"])})

    pr_ev = json.loads(conv.output_of("get_pull_request"))
    wants_draft = "draft" in conv.user.lower()
    if name == "get_pull_request":
        if not wants_draft:
            return _text(
                f"The latest merged change is [{pr_ev['title']}]({pr_ev['url']}). "
                "Ask me to draft news about it if you want a draft."
            )
        return conv.call("search_news", {"query": pr_ev["title"].split(": ", 1)[-1][:60]})
    if name == "search_news":
        malformed = _draft(pr_ev, pr_ev["evidence_id"])
        del malformed["draft"]["quotes"]
        return conv.call("check_draft", malformed)
    if name == "check_draft":
        if out.startswith("ERROR: invalid arguments"):
            return conv.call("check_draft", _draft(pr_ev, pr_ev["evidence_id"]))
        if not out.startswith("OK"):
            return _text(f"I did not save a draft. The check found:\n{out}")
        return conv.call("save_draft", _draft(pr_ev, pr_ev["evidence_id"]))
    if out.startswith("ACCEPTED"):
        return _text(f"Draft saved for [{pr_ev['title']}]({pr_ev['url']}): {out}")
    return _text(f"The draft was not saved: {out}")


def next_response(items: list[Any], tool_names: set[str], instructions: str) -> ModelResponse:
    conv = _Conversation(items)
    if "save_draft" in tool_names:
        return _chat(conv, instructions)
    return _scan(conv)


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
        return next_response(
            items, {getattr(t, "name", "") for t in tools}, system_instructions or ""
        )

    def stream_response(self, *args: Any, **kwargs: Any) -> AsyncIterator[Any]:
        raise NotImplementedError("the news agent runs the model without streaming")


class ScriptedModelProvider(ModelProvider):
    def get_model(self, model_name: str | None) -> Model:
        return ScriptedModel()
