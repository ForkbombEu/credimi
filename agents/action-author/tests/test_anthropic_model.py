# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later


import anthropic
import httpx2 as httpx
import pytest
from agents import ModelResponse, ModelSettings
from agents.exceptions import ModelBehaviorError, UserError
from openai import APIStatusError
from temporalio.contrib.pydantic import pydantic_data_converter

from credimi_action_author.anthropic_model import (
    AnthropicModel,
    context_policy,
    messages_from_items,
    response_from_message,
)
from credimi_action_author.workflow import TokenBudget, TokenBudgetExceeded

MODEL = "claude-sonnet-5-5"


def native_message(*, stop_reason="tool_use"):
    return anthropic.types.beta.BetaMessage.model_validate(
        {
            "id": "msg_test",
            "type": "message",
            "role": "assistant",
            "model": MODEL,
            "content": [
                {"type": "thinking", "thinking": "", "signature": "opaque-signed-thinking"},
                {"type": "tool_use", "id": "call_a", "name": "snapshot", "input": {}},
                {"type": "tool_use", "id": "call_b", "name": "get", "input": {"kind": "text"}},
            ],
            "stop_reason": stop_reason,
            "stop_sequence": None,
            "usage": {
                "input_tokens": 100,
                "output_tokens": 25,
                "cache_creation_input_tokens": 300,
                "cache_read_input_tokens": 600,
            },
            "context_management": {
                "applied_edits": [
                    {
                        "type": "clear_tool_uses_20250919",
                        "cleared_tool_uses": 4,
                        "cleared_input_tokens": 2000,
                    }
                ]
            },
        }
    )


async def test_signed_thinking_and_parallel_tool_ids_survive_temporal_roundtrip():
    response = response_from_message(native_message(), MODEL)
    payloads = await pydantic_data_converter.encode([response])
    (decoded,) = await pydantic_data_converter.decode(payloads, [ModelResponse])
    items = [
        {"role": "user", "content": "Inspect this wallet."},
        *decoded.to_input_items(),
        {"type": "function_call_output", "call_id": "call_b", "output": "Current text"},
        {"type": "function_call_output", "call_id": "call_a", "output": "Current screen"},
    ]
    messages = messages_from_items(items, MODEL)
    assert messages[1]["content"] == [
        {"type": "thinking", "thinking": "", "signature": "opaque-signed-thinking"},
        {"type": "tool_use", "id": "call_a", "name": "snapshot", "input": {}},
        {"type": "tool_use", "id": "call_b", "name": "get", "input": {"kind": "text"}},
    ]
    assert messages[2]["content"] == [
        {"type": "tool_result", "tool_use_id": "call_b", "content": "Current text"},
        {"type": "tool_result", "tool_use_id": "call_a", "content": "Current screen"},
    ]
    with pytest.raises(UserError, match="switching Claude models"):
        messages_from_items(items, "claude-opus-5-5")


async def test_cached_input_still_counts_toward_budget_and_reports_clearing():
    response = response_from_message(native_message(), MODEL)
    payloads = await pydantic_data_converter.encode([response])
    (decoded,) = await pydantic_data_converter.decode(payloads, [ModelResponse])
    budget = TokenBudget(1024)
    with pytest.raises(TokenBudgetExceeded, match="1025 used"):
        await budget.on_llm_end(None, None, decoded)
    assert budget.usage.input_tokens == 1000
    assert budget.usage.output_tokens == 25
    assert budget.usage.cache_read_tokens == 600
    assert budget.usage.cache_write_tokens == 300
    assert budget.usage.context_cleared_tokens == 2000


@pytest.mark.parametrize("reason", ["max_tokens", "pause_turn"])
def test_incomplete_response_never_dispatches_partial_tools(reason):
    with pytest.raises(ModelBehaviorError, match=reason):
        response_from_message(native_message(stop_reason=reason), MODEL)


def test_context_clearing_preserves_interactions_yaml_and_replay_evidence():
    tools = [
        "snapshot",
        "get",
        "maestro_inspect_screen",
        "set_target",
        "click",
        "fill",
        "check_action",
        "submit_action",
        "reset_app",
        "maestro_cheat_sheet",
    ]
    edit = context_policy(tools, 30000)["edits"][0]
    cleared_tools = set(tools) - set(edit["exclude_tools"])
    assert cleared_tools == {"snapshot", "get", "maestro_inspect_screen"}
    assert edit["clear_tool_inputs"] is False
    assert edit["keep"] == {"type": "tool_uses", "value": 3}
    assert context_policy(tools, 0) is None


async def test_native_http_errors_keep_harness_retry_classification():
    def deny(request):
        return httpx.Response(
            401,
            json={
                "type": "error",
                "error": {
                    "type": "authentication_error",
                    "message": "invalid API key",
                },
            },
        )

    async with anthropic.AsyncAnthropic(
        api_key="test-only",
        max_retries=0,
        http_client=httpx.AsyncClient(transport=httpx.MockTransport(deny)),
    ) as client:
        model = AnthropicModel(client, MODEL)
        with pytest.raises(APIStatusError) as caught:
            await model.get_response("Instructions", "Hello", ModelSettings(), [])
    assert caught.value.status_code == 401


@pytest.mark.parametrize(
    "item",
    [
        {"type": "reasoning", "encrypted_content": "another-provider-state"},
        {"role": "user", "content": [{"type": "input_image", "image_url": "unhandled"}]},
    ],
)
def test_unsupported_history_is_rejected_instead_of_silently_dropped(item):
    with pytest.raises(UserError):
        messages_from_items([item], MODEL)
