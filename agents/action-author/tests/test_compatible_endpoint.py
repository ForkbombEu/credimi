# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""The adapter between the SDK and non-OpenAI, OpenAI-compatible endpoints (Gemini, Ollama).

Each case pins a request shape that a real endpoint rejected or mishandled on 15/10/2026
(found with the news agent, whose adapter this is).
"""

from typing import Any

from agents import ModelProvider, ModelResponse, ModelSettings, Usage
from openai.types.responses import ResponseOutputMessage, ResponseOutputText
from openai.types.shared import Reasoning

from credimi_action_author.worker import _RenamedModels


class RecordingModel:
    """Stands in for the endpoint model: records the request, returns a fixed reply."""

    def __init__(self, name: str) -> None:
        self.name = name
        self.calls: list[dict[str, Any]] = []

    async def get_response(self, **request: Any) -> ModelResponse:
        self.calls.append(request)
        message = ResponseOutputMessage(
            id="m",
            content=[ResponseOutputText(text="ok", annotations=[], type="output_text")],
            role="assistant",
            status="completed",
            type="message",
        )
        return ModelResponse(output=[message], usage=Usage(), response_id=None)


class RecordingProvider(ModelProvider):
    def __init__(self) -> None:
        self.models: list[RecordingModel] = []

    def get_model(self, model_name: str | None) -> RecordingModel:
        model = RecordingModel(model_name or "")
        self.models.append(model)
        return model


def function_call(call_id: str, provider_data: dict[str, Any] | None = None) -> dict[str, Any]:
    item: dict[str, Any] = {
        "type": "function_call",
        "call_id": call_id,
        "name": "snapshot",
        "arguments": "{}",
    }
    if provider_data is not None:
        item["provider_data"] = provider_data
    return item


async def send(
    configured: str, *, items: list[Any] | None = None, settings: ModelSettings | None = None
) -> RecordingModel:
    provider = RecordingProvider()
    model = _RenamedModels(provider, configured).get_model("gpt-5.1")
    await model.get_response(
        system_instructions="Be precise.",
        input=items or [],
        model_settings=settings or ModelSettings(),
        tools=[],
        output_schema=None,
        handoffs=[],
        tracing=None,
    )
    return provider.models[0]


async def test_configured_model_replaces_the_requested_one():
    inner = await send("qwen3:14b")
    assert inner.name == "qwen3:14b"


async def test_gpt5_only_settings_are_dropped_and_the_rest_kept():
    settings = ModelSettings(
        verbosity="low", reasoning=Reasoning(effort="low"), tool_choice="required"
    )
    sent = (await send("gemini-3.5-flash", settings=settings)).calls[0]["model_settings"]
    assert sent.verbosity is None
    assert sent.reasoning is None
    assert sent.tool_choice == "required"


async def test_gemini_tool_calls_without_signature_get_the_placeholder():
    items = [
        {"role": "user", "content": "hi"},
        function_call("lost"),
        function_call("kept", {"thought_signature": "real-signature", "model": "gemini"}),
    ]
    sent = (await send("gemini-3.5-flash", items=items)).calls[0]["input"]
    assert sent[0] == {"role": "user", "content": "hi"}
    assert sent[1]["provider_data"]["thought_signature"] == "skip_thought_signature_validator"
    assert sent[2]["provider_data"] == {"thought_signature": "real-signature", "model": "gemini"}


async def test_other_endpoints_get_tool_calls_unchanged():
    items = [function_call("lost")]
    assert (await send("qwen3:14b", items=items)).calls[0]["input"] == items
