# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Worker for the NewsAgent workflow and its Python activities.

Environment:
    TEMPORAL_ADDRESS / TEMPORAL_NAMESPACE   Temporal connection (defaults: localhost:7233, default)
    OPENAI_API_KEY                          required unless NEWS_AGENT_FAKE_MODEL=1 or a
                                            compatible endpoint is set
    NEWS_AGENT_FAKE_MODEL=1                 use the scripted model (no API key, no cost)
    NEWS_AGENT_MODEL_BASE_URL               any OpenAI-compatible Chat Completions endpoint
                                            (Ollama, Gemini's OpenAI endpoint, ...)
    NEWS_AGENT_MODEL_API_KEY                key for that endpoint (Ollama accepts any value)
    NEWS_AGENT_MODEL                        model name to use there, e.g. qwen3:8b; replaces
                                            the model name the workflow asks for
    NEWS_AGENT_FIXTURES_DIR / NEWS_AGENT_RECORD_DIR / GITHUB_TOKEN   see http.py
    NEWS_AGENT_SOURCES                      path of sources.toml (default: agents/news/sources.toml)

The Go activities this workflow calls run in `credimi serve` (Temporal workers enabled).
"""

import asyncio
import dataclasses
import json
import logging
import os
import re
import sys
from datetime import timedelta

from agents import (
    Model,
    ModelProvider,
    ModelResponse,
    ModelSettings,
    OpenAIProvider,
    set_tracing_disabled,
)
from openai import AsyncOpenAI
from temporal_agent_harness.ai_sdks.openai_agents import (
    ModelActivityParameters,
    OpenAIAgentsPlugin,
)
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio.client import Client
from temporalio.envconfig import ClientConfig
from temporalio.worker import Worker

from .activities import ALL_ACTIVITIES
from .fake_model import ScriptedModelProvider
from .models import TASK_QUEUE
from .workflow import NewsAgentWorkflow


def fake_model_enabled() -> bool:
    return os.environ.get("NEWS_AGENT_FAKE_MODEL", "").lower() in {"1", "true", "yes"}


# Gemini's documented placeholder for a tool call whose thought signature was not kept.
_GEMINI_PLACEHOLDER_SIGNATURE = "skip_thought_signature_validator"


class _PortableModel(Model):
    """Adapts the SDK's requests to non-OpenAI, OpenAI-compatible endpoints.

    * Drops OpenAI-only settings the SDK adds for gpt-5 model names (verbosity, reasoning).
    * Gemini 3 rejects a tool call sent back without its thought signature. The SDK keeps the
      signature in the item's `provider_data`, but the harness decodes the model activity's
      input into typed items that have no such field, so it is lost. Tool calls without one get
      Gemini's placeholder, as the SDK's own LiteLLM model does.
    * Ollama constrains the whole reply to a requested JSON schema, so with a typed output the
      model can never call a tool. For non-Gemini endpoints the schema moves from the request
      into the instructions; the SDK still validates the final reply against it.
    """

    def __init__(self, inner: Model, model_name: str) -> None:
        self._inner = inner
        self._gemini = "gemini" in model_name.lower()

    @staticmethod
    def _portable(settings: ModelSettings) -> ModelSettings:
        return dataclasses.replace(settings, verbosity=None, reasoning=None)

    def _items(self, items):
        if not self._gemini or isinstance(items, str):
            return items
        out = []
        for item in items:
            if (
                isinstance(item, dict)
                and item.get("type") == "function_call"
                and not (item.get("provider_data") or {}).get("thought_signature")
            ):
                provider_data = {
                    **(item.get("provider_data") or {}),
                    "thought_signature": _GEMINI_PLACEHOLDER_SIGNATURE,
                }
                item = {**item, "provider_data": provider_data}
            out.append(item)
        return out

    def _adapt(self, system_instructions, input, model_settings, kwargs):  # noqa: A002
        schema = kwargs.get("output_schema")
        if not self._gemini and schema is not None and not schema.is_plain_text():
            kwargs = {**kwargs, "output_schema": None}
            system_instructions = (
                f"{system_instructions or ''}\n\nWhen you have finished using tools, reply with "
                "only a JSON object, without code fences, that matches this JSON schema:\n"
                f"{json.dumps(schema.json_schema())}"
            )
        return system_instructions, self._items(input), self._portable(model_settings), kwargs

    async def get_response(self, system_instructions, input, model_settings, **kwargs):  # noqa: A002
        instructions, items, settings, kwargs = self._adapt(
            system_instructions, input, model_settings, kwargs
        )
        response = await self._inner.get_response(
            system_instructions=instructions, input=items, model_settings=settings, **kwargs
        )
        return _strip_code_fences(response)

    def stream_response(self, system_instructions, input, model_settings, **kwargs):  # noqa: A002
        instructions, items, settings, kwargs = self._adapt(
            system_instructions, input, model_settings, kwargs
        )
        return self._inner.stream_response(
            system_instructions=instructions, input=items, model_settings=settings, **kwargs
        )


def _strip_code_fences(response: ModelResponse) -> ModelResponse:
    """Small models often wrap a JSON reply in ```json fences, which JSON validation rejects."""
    for item in response.output:
        for part in getattr(item, "content", None) or []:
            text = getattr(part, "text", None)
            if isinstance(text, str) and text.strip().startswith("```"):
                part.text = re.sub(r"^```[a-zA-Z]*\s*|\s*```$", "", text.strip())
    return response


class _RenamedModels(ModelProvider):
    """Serves every model request with one configured model of a compatible endpoint."""

    def __init__(self, inner: ModelProvider, model: str) -> None:
        self._inner = inner
        self._model = model

    def get_model(self, model_name: str | None) -> Model:
        name = self._model or model_name or ""
        return _PortableModel(self._inner.get_model(name), name)


def compatible_endpoint() -> str:
    return os.environ.get("NEWS_AGENT_MODEL_BASE_URL", "")


def model_provider() -> ModelProvider | None:
    if fake_model_enabled():
        return ScriptedModelProvider()
    if base_url := compatible_endpoint():
        client = AsyncOpenAI(
            base_url=base_url,
            api_key=os.environ.get("NEWS_AGENT_MODEL_API_KEY", "unused"),
            # Activity retries own the retry policy.
            max_retries=0,
        )
        # Compatible endpoints implement Chat Completions, not OpenAI's Responses API.
        provider = OpenAIProvider(openai_client=client, use_responses=False)
        return _RenamedModels(provider, os.environ.get("NEWS_AGENT_MODEL", ""))
    return None


def openai_plugin() -> OpenAIAgentsPlugin:
    return OpenAIAgentsPlugin(
        model_params=ModelActivityParameters(
            # Local models are slow; a turn's model call may take minutes.
            start_to_close_timeout=timedelta(minutes=10 if compatible_endpoint() else 3),
        ),
        model_provider=model_provider(),
    )


async def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
    if fake_model_enabled() or compatible_endpoint():
        # No OpenAI key to export traces with.
        set_tracing_disabled(True)
    elif not os.environ.get("OPENAI_API_KEY"):
        sys.exit("error: set OPENAI_API_KEY, NEWS_AGENT_MODEL_BASE_URL, or NEWS_AGENT_FAKE_MODEL=1")

    client = await Client.connect(
        **ClientConfig.load_client_connect_config(),
        plugins=[openai_plugin(), AgentHarnessPlugin()],
    )
    worker = Worker(
        client,
        task_queue=TASK_QUEUE,
        workflows=[NewsAgentWorkflow],
        activities=ALL_ACTIVITIES,
    )
    if fake_model_enabled():
        model = "scripted"
    elif compatible_endpoint():
        model = f"{os.environ.get('NEWS_AGENT_MODEL') or 'as requested'} at {compatible_endpoint()}"
    else:
        model = "openai"
    print(
        f"news agent worker: namespace={client.namespace} task_queue={TASK_QUEUE} "
        f"model={model} fixtures={os.environ.get('NEWS_AGENT_FIXTURES_DIR') or 'off'}",
        flush=True,
    )
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
