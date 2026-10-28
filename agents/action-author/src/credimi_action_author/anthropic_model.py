# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Native Claude Messages behind the harness's existing Agents SDK model boundary.

Signed thinking travels in the SDK's opaque reasoning field, not provider_data: the
harness's typed activity input preserves the former but can discard the latter.
Context edits happen server-side so signed conversation prefixes remain unchanged.
"""

import base64
import json
import logging
from typing import Any

import anthropic
from agents import FunctionTool, Model, ModelProvider, ModelResponse, ModelSettings, Usage
from agents.exceptions import ModelBehaviorError, UserError
from openai import APIStatusError
from openai.types.responses import (
    ResponseFunctionToolCall,
    ResponseOutputMessage,
    ResponseOutputText,
    ResponseReasoningItem,
)
from openai.types.responses.response_usage import InputTokensDetails

logger = logging.getLogger(__name__)
_CONTEXT_BETA = "context-management-2025-06-27"
_THINKING_PREFIX = "anthropic:"
# Only disposable observations may be cleared. Interaction results, parameters,
# generated YAML, validation and replay reports remain available to the model.
OBSERVATION_TOOLS = frozenset(
    {"snapshot", "screenshot", "maestro_inspect_screen", "find", "get", "is"}
)


def context_policy(tool_names: list[str], trigger_tokens: int) -> dict[str, Any] | None:
    if trigger_tokens == 0 or not OBSERVATION_TOOLS.intersection(tool_names):
        return None
    return {
        "edits": [
            {
                "type": "clear_tool_uses_20250919",
                "trigger": {"type": "input_tokens", "value": trigger_tokens},
                "keep": {"type": "tool_uses", "value": 3},
                "clear_at_least": {"type": "input_tokens", "value": 5000},
                "clear_tool_inputs": False,
                "exclude_tools": sorted(set(tool_names) - OBSERVATION_TOOLS),
            }
        ]
    }


def _text_blocks(content: Any) -> list[dict[str, Any]]:
    if isinstance(content, str):
        return [{"type": "text", "text": content}] if content else []
    blocks = []
    for part in content or []:
        kind = part.get("type")
        if kind in {"input_text", "output_text", "text"}:
            blocks.append({"type": "text", "text": part["text"]})
        elif kind == "refusal":
            blocks.append({"type": "text", "text": part["refusal"]})
        else:
            raise UserError(f"Native action author does not support content block {kind!r}")
    return blocks


def messages_from_items(items: str | list[dict[str, Any]], model: str) -> list[dict[str, Any]]:
    if isinstance(items, str):
        items = [{"role": "user", "content": items}]
    messages: list[dict[str, Any]] = []
    for item in items:
        kind = item.get("type", "message")
        if kind == "message":
            role = item["role"]
            if role not in {"user", "assistant"}:
                raise UserError(
                    f"Unexpected conversation role {role!r}; system instructions are separate"
                )
            blocks = _text_blocks(item.get("content"))
        elif kind == "function_call":
            role = "assistant"
            blocks = [
                {
                    "type": "tool_use",
                    "id": item["call_id"],
                    "name": item["name"],
                    "input": json.loads(item["arguments"]),
                }
            ]
        elif kind == "function_call_output":
            role = "user"
            output = item.get("output", "")
            blocks = [
                {
                    "type": "tool_result",
                    "tool_use_id": item["call_id"],
                    "content": output if isinstance(output, str) else _text_blocks(output),
                }
            ]
        elif kind == "reasoning":
            opaque = item.get("encrypted_content") or ""
            if not opaque.startswith(_THINKING_PREFIX):
                raise UserError("Start a fresh session when switching to native Claude reasoning")
            envelope = json.loads(base64.b64decode(opaque[len(_THINKING_PREFIX) :]))
            if envelope["model"] != model:
                raise UserError(
                    "Start a fresh session when switching Claude models with signed thinking"
                )
            role, blocks = "assistant", [envelope["block"]]
        else:
            raise UserError(f"Native action author does not support input item {kind!r}")
        if not blocks:
            continue
        if messages and messages[-1]["role"] == role:
            messages[-1]["content"].extend(blocks)
        else:
            messages.append({"role": role, "content": blocks})
    return messages


def response_from_message(message: Any, model: str) -> ModelResponse:
    output = []
    for index, block in enumerate(message.content):
        item_id = f"{message.id}_{index}"
        if block.type == "text":
            output.append(
                ResponseOutputMessage(
                    id=item_id,
                    type="message",
                    role="assistant",
                    status="completed",
                    content=[
                        ResponseOutputText(type="output_text", text=block.text, annotations=[])
                    ],
                )
            )
        elif block.type == "tool_use":
            output.append(
                ResponseFunctionToolCall(
                    id=item_id,
                    type="function_call",
                    call_id=block.id,
                    name=block.name,
                    arguments=json.dumps(block.input),
                    status="completed",
                )
            )
        elif block.type in {"thinking", "redacted_thinking"}:
            envelope = {"model": model, "block": block.model_dump(mode="json", exclude_none=True)}
            encoded = base64.b64encode(json.dumps(envelope).encode()).decode()
            output.append(
                ResponseReasoningItem(
                    id=item_id,
                    type="reasoning",
                    summary=[],
                    encrypted_content=_THINKING_PREFIX + encoded,
                )
            )
        else:
            raise ModelBehaviorError(f"Unexpected Claude response block {block.type!r}")
    usage = message.usage
    reads = usage.cache_read_input_tokens or 0
    writes = usage.cache_creation_input_tokens or 0
    # Native input_tokens excludes cache reads/writes; SDK budgets must include all three.
    input_tokens = usage.input_tokens + reads + writes
    edits = getattr(message, "context_management", None)
    cleared = sum(
        getattr(edit, "cleared_input_tokens", 0) for edit in (edits.applied_edits if edits else [])
    )
    details = InputTokensDetails.model_validate(
        {
            "cached_tokens": reads,
            "cache_write_tokens": writes,
            "context_cleared_tokens": cleared,
        }
    )
    logger.info(
        "Claude usage: input=%d output=%d cache_read=%d cache_write=%d context_cleared=%d",
        input_tokens,
        usage.output_tokens,
        reads,
        writes,
        cleared,
    )
    if message.stop_reason in {"max_tokens", "pause_turn"}:
        raise ModelBehaviorError(
            f"Claude stopped with {message.stop_reason}; no partial tools executed"
        )
    return ModelResponse(
        output=output,
        usage=Usage(
            requests=1,
            input_tokens=input_tokens,
            output_tokens=usage.output_tokens,
            total_tokens=input_tokens + usage.output_tokens,
            input_tokens_details=details,
        ),
        response_id=None,
    )


class AnthropicModel(Model):
    def __init__(
        self,
        client: anthropic.AsyncAnthropic,
        model: str,
        *,
        cache_ttl: str = "5m",
        context_tokens: int = 30_000,
    ):
        if cache_ttl not in {"5m", "1h", "off"}:
            raise ValueError("ACTION_AUTHOR_ANTHROPIC_CACHE_TTL must be 5m, 1h or off")
        if context_tokens < 0:
            raise ValueError("ACTION_AUTHOR_ANTHROPIC_CONTEXT_TOKENS cannot be negative")
        self.client, self.model = client, model
        self.cache_ttl, self.context_tokens = cache_ttl, context_tokens

    async def get_response(
        self,
        system_instructions,
        input,
        model_settings: ModelSettings,
        tools,
        output_schema=None,
        handoffs=None,
        tracing=None,
        *,
        previous_response_id=None,
        conversation_id=None,
        prompt=None,
    ):  # noqa: A002
        if previous_response_id or conversation_id or prompt or handoffs:
            raise UserError("Native action author uses local conversation history and no handoffs")
        if output_schema is not None and not output_schema.is_plain_text():
            raise UserError(
                "Native action author returns text and function tools, not structured output"
            )
        if any(not isinstance(tool, FunctionTool) for tool in tools):
            raise UserError("Native action author supports only client-side function tools")
        if model_settings.tool_choice is not None and model_settings.tool_choice != "auto":
            raise UserError("Native action author requires automatic tool selection")
        native_tools = [
            {
                "name": tool.name,
                "description": tool.description,
                "input_schema": tool.params_json_schema,
            }
            for tool in tools
        ]
        request: dict[str, Any] = {
            "model": self.model,
            "max_tokens": model_settings.max_tokens or 8192,
            "messages": messages_from_items(input, self.model),
        }
        if system_instructions:
            request["system"] = system_instructions
        if native_tools:
            request["tools"] = native_tools
        if self.cache_ttl != "off":
            request["cache_control"] = {"type": "ephemeral", "ttl": self.cache_ttl}
        policy = context_policy([tool.name for tool in tools], self.context_tokens)
        if policy:
            request["context_management"] = policy
        try:
            if policy:
                message = await self.client.beta.messages.create(betas=[_CONTEXT_BETA], **request)
            else:
                message = await self.client.messages.create(**request)
        except anthropic.APIStatusError as exc:
            # Reuse the harness's status/retry classification; SDK retries are disabled.
            raise APIStatusError(exc.message, response=exc.response, body=exc.body) from exc
        return response_from_message(message, self.model)

    def stream_response(self, *args, **kwargs):
        raise UserError("The action-author harness invokes native Claude without model streaming")


class AnthropicModelProvider(ModelProvider):
    def __init__(
        self, model: str, api_key: str, *, cache_ttl: str = "5m", context_tokens: int = 30_000
    ):
        self.client = anthropic.AsyncAnthropic(api_key=api_key, max_retries=0)
        self.model = AnthropicModel(
            self.client, model, cache_ttl=cache_ttl, context_tokens=context_tokens
        )

    def get_model(self, model_name: str | None) -> Model:
        return self.model

    async def aclose(self) -> None:
        await self.client.close()
