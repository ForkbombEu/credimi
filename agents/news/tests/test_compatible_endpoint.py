# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""The adapter between the SDK and non-OpenAI, OpenAI-compatible endpoints (Gemini, Ollama).

Each case pins a request shape that a real endpoint rejected or mishandled on 15/10/2026.
"""

from typing import Any

import pytest
from agents import AgentOutputSchema, ModelProvider, ModelResponse, ModelSettings, Usage
from openai.types.responses import ResponseOutputMessage, ResponseOutputText
from openai.types.shared import Reasoning

from credimi_news_agent.models import ScanDecisions
from credimi_news_agent.worker import _RenamedModels


class RecordingModel:
    """Stands in for the endpoint model: records the request, returns a fixed reply."""

    def __init__(self, name: str, reply: str) -> None:
        self.name = name
        self.reply = reply
        self.calls: list[dict[str, Any]] = []

    async def get_response(self, **request: Any) -> ModelResponse:
        self.calls.append(request)
        message = ResponseOutputMessage(
            id="m",
            content=[ResponseOutputText(text=self.reply, annotations=[], type="output_text")],
            role="assistant",
            status="completed",
            type="message",
        )
        return ModelResponse(output=[message], usage=Usage(), response_id=None)


class RecordingProvider(ModelProvider):
    def __init__(self, reply: str = "ok") -> None:
        self.reply = reply
        self.models: list[RecordingModel] = []

    def get_model(self, model_name: str | None) -> RecordingModel:
        model = RecordingModel(model_name or "", self.reply)
        self.models.append(model)
        return model


def function_call(call_id: str, provider_data: dict[str, Any] | None = None) -> dict[str, Any]:
    item: dict[str, Any] = {
        "type": "function_call",
        "call_id": call_id,
        "name": "list_changes",
        "arguments": "{}",
    }
    if provider_data is not None:
        item["provider_data"] = provider_data
    return item


async def send(
    configured: str,
    *,
    items: list[Any] | None = None,
    settings: ModelSettings | None = None,
    output_schema: Any = None,
    reply: str = "ok",
) -> tuple[RecordingModel, ModelResponse]:
    provider = RecordingProvider(reply)
    model = _RenamedModels(provider, configured).get_model("gpt-5.1")
    response = await model.get_response(
        system_instructions="Be factual.",
        input=items or [],
        model_settings=settings or ModelSettings(),
        tools=[],
        output_schema=output_schema,
        handoffs=[],
        tracing=None,
    )
    return provider.models[0], response


async def test_configured_model_replaces_the_requested_one():
    inner, _ = await send("qwen3:14b")
    assert inner.name == "qwen3:14b"


async def test_gpt5_only_settings_are_dropped_and_the_rest_kept():
    settings = ModelSettings(
        verbosity="low", reasoning=Reasoning(effort="low"), tool_choice="required"
    )
    inner, _ = await send("gemini-3.5-flash", settings=settings)
    sent = inner.calls[0]["model_settings"]
    assert sent.verbosity is None
    assert sent.reasoning is None
    assert sent.tool_choice == "required"


async def test_gemini_tool_calls_without_signature_get_the_placeholder():
    items = [
        {"role": "user", "content": "hi"},
        function_call("lost"),
        function_call("kept", {"thought_signature": "real-signature", "model": "gemini"}),
    ]
    inner, _ = await send("gemini-3.5-flash", items=items)
    sent = inner.calls[0]["input"]
    assert sent[0] == {"role": "user", "content": "hi"}
    assert sent[1]["provider_data"]["thought_signature"] == "skip_thought_signature_validator"
    assert sent[2]["provider_data"] == {"thought_signature": "real-signature", "model": "gemini"}


async def test_other_endpoints_get_tool_calls_unchanged():
    items = [function_call("lost")]
    inner, _ = await send("qwen3:14b", items=items)
    assert inner.calls[0]["input"] == items


async def test_typed_output_moves_into_instructions_except_for_gemini():
    schema = AgentOutputSchema(ScanDecisions)

    ollama, _ = await send("qwen3:14b", output_schema=schema)
    assert ollama.calls[0]["output_schema"] is None
    assert ollama.calls[0]["system_instructions"].startswith("Be factual.")
    assert '"decisions"' in ollama.calls[0]["system_instructions"]

    gemini, _ = await send("gemini-3.5-flash", output_schema=schema)
    assert gemini.calls[0]["output_schema"] is schema
    assert gemini.calls[0]["system_instructions"] == "Be factual."


@pytest.mark.parametrize(
    ("reply", "expected"),
    [
        ('```json\n{"a": 1}\n```', '{"a": 1}'),
        ('```\n{"a": 1}\n```', '{"a": 1}'),
        ('{"a": 1}', '{"a": 1}'),
        ("Plain answer with ``` inside", "Plain answer with ``` inside"),
    ],
)
async def test_code_fences_around_a_reply_are_removed(reply, expected):
    _, response = await send("qwen3:14b", reply=reply)
    assert response.output[0].content[0].text == expected
