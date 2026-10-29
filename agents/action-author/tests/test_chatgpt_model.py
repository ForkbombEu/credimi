# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""ChatGPT-plan request constraints at the harness model-provider boundary."""

import json

import httpx
import pytest
from agents import Agent, ModelSettings, ModelTracing, RunConfig, Runner, function_tool
from openai import APIStatusError, AsyncOpenAI
from openai.types.shared import Reasoning

from credimi_action_author.chatgpt_model import ChatGPTModelProvider, ChatGPTResponsesModel


@function_tool
def inspect_wallet() -> str:
    """Inspect the test wallet."""
    return "ok"


def completed_stream(item=None):
    response = {
        "id": "resp_test",
        "created_at": 1,
        "model": "gpt-test",
        "object": "response",
        "output": [],
        "parallel_tool_calls": True,
        "status": "completed",
        "tool_choice": "auto",
        "tools": [],
        "usage": {"input_tokens": 3, "output_tokens": 1, "total_tokens": 4},
    }
    message = item or {
        "id": "msg_test",
        "type": "message",
        "status": "completed",
        "role": "assistant",
        "content": [{"type": "output_text", "text": "harness-ok", "annotations": []}],
    }
    return (
        "event: response.output_item.done\n"
        "data: "
        + json.dumps({"type": "response.output_item.done", "output_index": 0, "item": message})
        + "\n\n"
        "event: response.completed\n"
        f"data: {json.dumps({'type': 'response.completed', 'response': response})}\n\n"
    ).encode()


@pytest.mark.parametrize("effort", ["none", "low", "high"])
async def test_nonstreaming_harness_call_still_uses_required_subscription_request(effort):
    requests = []

    def respond(request: httpx.Request) -> httpx.Response:
        requests.append(request)
        return httpx.Response(
            200,
            headers={"Content-Type": "text/event-stream"},
            content=completed_stream(),
            request=request,
        )

    http = httpx.AsyncClient(transport=httpx.MockTransport(respond))
    client = AsyncOpenAI(api_key="oauth-token", http_client=http, max_retries=0)
    model = ChatGPTResponsesModel(model="gpt-test", openai_client=client)
    result = await model.get_response(
        system_instructions="Author an action.",
        input=[{"role": "user", "content": "Inspect it"}],
        model_settings=ModelSettings(
            temperature=0.4, max_tokens=50, store=True, reasoning=Reasoning(effort=effort)
        ),
        tools=[inspect_wallet],
        output_schema=None,
        handoffs=[],
        tracing=ModelTracing.DISABLED,
        previous_response_id="must-not-be-sent",
    )
    await http.aclose()
    assert result.usage.input_tokens == 3
    assert result.output[0].content[0].text == "harness-ok"
    body = json.loads(requests[0].content)
    assert body["store"] is False
    assert body["stream"] is True
    assert "temperature" not in body
    assert "max_output_tokens" not in body
    assert "previous_response_id" not in body
    if effort == "none":
        assert "reasoning" not in body
    else:
        assert body["reasoning"]["effort"] == effort


async def test_subscription_tool_executes_and_its_result_reaches_the_next_model_call():
    selected = []

    @function_tool
    def select_wallet(app_id: str) -> str:
        """Select the wallet under test."""
        selected.append(app_id)
        return "wallet-selected"

    def respond(request: httpx.Request) -> httpx.Response:
        body = json.loads(request.content)
        outputs = [item for item in body["input"] if item.get("type") == "function_call_output"]
        if outputs:
            assert outputs[0]["call_id"] == "call_select"
            assert outputs[0]["output"] == "wallet-selected"
            content = completed_stream()
        else:
            call = {
                "type": "function_call",
                "id": "fc_select",
                "call_id": "call_select",
                "name": "select_wallet",
                "arguments": '{"app_id":"eu.europa.ec.euidi"}',
                "status": "completed",
            }
            # A namespace advertised on the wire is part of the returned tool identity.
            for tool in body.get("tools", []):
                if tool["type"] == "namespace":
                    call["namespace"] = tool["name"]
            content = completed_stream(call)
        return httpx.Response(
            200, headers={"Content-Type": "text/event-stream"}, content=content, request=request
        )

    async with httpx.AsyncClient(transport=httpx.MockTransport(respond)) as http:
        model = ChatGPTResponsesModel(
            model="gpt-test", openai_client=AsyncOpenAI(api_key="oauth", http_client=http)
        )
        result = await Runner.run(
            Agent(name="CredimiActionAuthor", model=model, tools=[select_wallet]),
            input="Select the wallet.",
            run_config=RunConfig(tracing_disabled=True),
        )
    assert selected == ["eu.europa.ec.euidi"]
    assert result.final_output == "harness-ok"


@pytest.mark.parametrize(
    ("code", "status", "should_retry"),
    [
        ("subscription_sharing_usage_limit_exceeded", 429, "false"),
        ("server_error", 500, None),
    ],
)
async def test_stream_failure_keeps_harness_retry_classification(code, status, should_retry):
    response = {
        "id": "resp_failed",
        "created_at": 1,
        "model": "gpt-test",
        "object": "response",
        "output": [],
        "parallel_tool_calls": True,
        "status": "failed",
        "tool_choice": "auto",
        "tools": [],
        "error": {"code": code, "message": "The request failed."},
    }
    stream = (
        "event: response.failed\n"
        f"data: {json.dumps({'type': 'response.failed', 'response': response})}\n\n"
    ).encode()

    def respond(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            200,
            headers={"Content-Type": "text/event-stream"},
            content=stream,
            request=request,
        )

    http = httpx.AsyncClient(transport=httpx.MockTransport(respond))
    model = ChatGPTResponsesModel(
        model="gpt-test",
        openai_client=AsyncOpenAI(api_key="oauth-token", http_client=http, max_retries=0),
    )
    with pytest.raises(APIStatusError) as caught:
        await model.get_response(
            system_instructions=None,
            input=[],
            model_settings=ModelSettings(),
            tools=[],
            output_schema=None,
            handoffs=[],
            tracing=ModelTracing.DISABLED,
        )
    await http.aclose()
    assert caught.value.status_code == status
    assert caught.value.response.headers.get("x-should-retry") == should_retry


def test_provider_uses_selected_subscription_model():
    model = ChatGPTModelProvider("gpt-test").get_model("ignored")
    assert isinstance(model, ChatGPTResponsesModel)
    assert model.model == "gpt-test"
