# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""ChatGPT-plan Responses requests, constrained to the public subscription route."""

from typing import Any

import httpx
from agents import Model, ModelProvider, ModelResponse, OpenAIProvider
from agents.models.openai_responses import OpenAIResponsesModel, _usage_from_response
from openai import APIStatusError, AsyncOpenAI
from openai.types.responses import (
    ResponseCompletedEvent,
    ResponseFailedEvent,
    ResponseIncompleteEvent,
)
from openai.types.shared import Reasoning

from .chatgpt_auth import chatgpt_access_token

# https://developers.openai.com/siwc/token-sharing-open-source/preview-limitations
_UNSUPPORTED = (
    "background",
    "conversation",
    "max_output_tokens",
    "max_tool_calls",
    "metadata",
    "moderation",
    "multi_agent",
    "prompt",
    "prompt_cache_retention",
    "safety_identifier",
    "temperature",
    "top_logprobs",
    "top_p",
    "truncation",
    "user",
    "previous_response_id",
)


async def _bearer() -> str:
    return await chatgpt_access_token()


class ChatGPTResponsesModel(OpenAIResponsesModel):
    """The SDK's Responses model with the subscription route's required request shape."""

    def _build_response_create_kwargs(self, **kwargs: Any) -> dict[str, Any]:
        request = super()._build_response_create_kwargs(**kwargs)
        for key in _UNSUPPORTED:
            request.pop(key, None)
        # The harness defaults GPT-5 requests to effort="none". After replacing
        # that model, let the selected subscription model choose its supported default.
        reasoning = request.get("reasoning")
        if isinstance(reasoning, Reasoning):
            reasoning = reasoning.model_dump(exclude_none=True)
        if isinstance(reasoning, dict) and reasoning.get("effort") == "none":
            reasoning = {key: value for key, value in reasoning.items() if key != "effort"}
            if reasoning:
                request["reasoning"] = reasoning
            else:
                request.pop("reasoning")
        request["store"] = False
        request["stream"] = True
        request["extra_headers"] = None
        request["extra_query"] = None
        request["extra_body"] = None
        tools = request.get("tools")
        if isinstance(tools, list) and tools:
            # Keep SDK tool identities flat. A wire-only namespace would make
            # returned calls undispatchable by the harness's bare-name tool registry.
            request["tools"] = []
            request["input"] = [
                {"type": "additional_tools", "role": "developer", "tools": tools},
                *request["input"],
            ]
        return request

    async def get_response(self, *args: Any, **kwargs: Any):
        # This route requires SSE even when the harness uses its non-streaming activity.
        kwargs.pop("tracing", None)  # SDK concern, not a Responses API argument.
        stream = await self._fetch_response(*args, stream=True, **kwargs)
        output_items = []
        try:
            async for event in stream:
                if event.type == "response.output_item.done":
                    output_items.append(event.item)
                if isinstance(event, ResponseCompletedEvent):
                    return ModelResponse(
                        output=event.response.output or output_items,
                        usage=_usage_from_response(event.response),
                        response_id=event.response.id,
                    )
                if isinstance(event, (ResponseFailedEvent, ResponseIncompleteEvent)):
                    error = event.response.error
                    code = error.code if error else event.type
                    status = 503 if code == "subscription_sharing_usage_unavailable" else 400
                    if code == "subscription_sharing_usage_limit_exceeded":
                        status = 429
                    elif code in {
                        "subscription_sharing_user_not_eligible",
                        "subscription_sharing_route_not_supported",
                        "chatpass_v2_scope_not_authorized",
                        "chatpass_v2_invalid_authorization_context",
                    }:
                        status = 403
                    elif code == "subscription_sharing_invalid_user":
                        status = 401
                    headers = {} if status == 503 else {"x-should-retry": "false"}
                    raise APIStatusError(
                        error.message if error else f"ChatGPT Responses stream ended: {event.type}",
                        response=httpx.Response(
                            status,
                            headers=headers,
                            request=httpx.Request("POST", "https://api.openai.com/v1/responses"),
                        ),
                        body=error.model_dump() if error else {"code": code},
                    )
            raise RuntimeError("ChatGPT Responses stream ended without response.completed.")
        finally:
            await stream.aclose()


class ChatGPTModelProvider(ModelProvider):
    def __init__(self, model: str) -> None:
        client = AsyncOpenAI(api_key=_bearer, max_retries=0)
        self.inner = OpenAIProvider(openai_client=client, use_responses=True)
        self.model = model

    def get_model(self, model_name: str | None) -> Model:
        name = self.model or model_name or ""
        # OpenAIProvider constructs this same SDK model; substituting the subclass
        # keeps all serialization, streaming and error handling in one implementation.
        base = self.inner.get_model(name)
        assert isinstance(base, OpenAIResponsesModel)
        return ChatGPTResponsesModel(model=name, openai_client=base._get_client())
