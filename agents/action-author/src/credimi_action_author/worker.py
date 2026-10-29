# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Worker for the ActionAuthor workflow, its activities and the agent-device MCP server.

Environment (model options are checked in this order; the first enabled one is used):
    TEMPORAL_ADDRESS / TEMPORAL_NAMESPACE   Temporal connection; use the wallet owner's namespace
    ACTION_AUTHOR_DEVICE_SERIAL             adb serial of the Android phone, or
    ACTION_AUTHOR_IOS_SIMULATOR_UDID        UDID of a shut-down iOS simulator the worker clones
                                            (macOS); one of the two is required unless
                                            ACTION_AUTHOR_FAKE_MODEL=1 (then, without either, the
                                            device side is faked too)
    ACTION_AUTHOR_FAKE_MODEL=1              scripted model (no API key, no cost)
    ACTION_AUTHOR_CHATGPT=1                 local ChatGPT subscription through browser OAuth;
                                            no Codex CLI, API key or OpenAI-billed requests
    ACTION_AUTHOR_CHATGPT_MODEL             subscription model (default gpt-6.1-sol)
    ACTION_AUTHOR_CHATGPT_CREDENTIALS       optional local credential path (default
                                            ~/.config/credimi-action-author/chatgpt.json)
    ACTION_AUTHOR_ANTHROPIC=1               native Claude Messages (also selected by the official
                                            Anthropic MODEL_BASE_URL); uses MODEL_API_KEY/MODEL
    ACTION_AUTHOR_ANTHROPIC_CACHE_TTL       5m (default), 1h, or off
    ACTION_AUTHOR_ANTHROPIC_CONTEXT_TOKENS  stale-observation clearing threshold (30000; 0 off)
    ACTION_AUTHOR_MODEL_BASE_URL            any OpenAI-compatible Chat Completions endpoint
                                            (Ollama, freellmapi, Gemini's OpenAI endpoint, ...)
    ACTION_AUTHOR_MODEL_API_KEY             key for that endpoint or native Claude
                                            (Ollama accepts any value)
    ACTION_AUTHOR_MODEL                     model name there, e.g. qwen3:14b; replaces the model
                                            name the workflow asks for
    OPENAI_API_KEY                          plain OpenAI, when no other option is set
    ACTION_AUTHOR_TOKEN_BUDGET              tokens per `ask` turn (default 600000)
    ACTION_AUTHOR_MAX_MODEL_CALLS           model calls per `ask` turn (default 80)
    ACTION_AUTHOR_OUT_DIR                   where accepted actions are written
                                            (default: agents/action-author/out)
    ACTION_AUTHOR_CREDIMI_URL               trusted Credimi origin (default http://localhost:8090)
"""

import asyncio
import dataclasses
import logging
import os
import sys
from datetime import timedelta
from urllib.parse import urlparse

from agents import Model, ModelProvider, ModelSettings, OpenAIProvider, set_tracing_disabled
from openai import AsyncOpenAI
from temporal_agent_harness.ai_sdks.openai_agents import (
    ModelActivityParameters,
    OpenAIAgentsPlugin,
    StatelessMCPServerProvider,
)
from temporal_agent_harness.plugin import AgentHarnessPlugin
from temporalio.client import Client
from temporalio.common import RetryPolicy
from temporalio.envconfig import ClientConfig
from temporalio.worker import Worker

from .activities import ActionAuthorActivities
from .device import AndroidDevice, Device, agent_device_factory
from .fake_device import FakeDevice
from .fake_model import ScriptedModelProvider
from .ios import IOSSimulatorDevice
from .models import AGENT_DEVICE_MCP, TASK_QUEUE
from .workflow import ActionAuthorWorkflow

AGENT_DEVICE_COMMAND = ["agent-device", "mcp"]
FAKE_AGENT_DEVICE_COMMAND = [sys.executable, "-m", "credimi_action_author.fake_device"]
DEFAULT_CHATGPT_MODEL = "gpt-6.1-sol"
DEFAULT_CLAUDE_MODEL = "claude-sonnet-5-5"


def _env(name: str, default: str = "") -> str:
    return os.environ.get(name, "").strip() or default


def _flag(name: str) -> bool:
    return _env(name).lower() in {"1", "true", "yes"}


def fake_model_enabled() -> bool:
    return _flag("ACTION_AUTHOR_FAKE_MODEL")


# Gemini's documented placeholder for a tool call whose thought signature was not kept.
_GEMINI_PLACEHOLDER_SIGNATURE = "skip_thought_signature_validator"


class _PortableModel(Model):
    """Adapts the SDK's requests to non-OpenAI, OpenAI-compatible endpoints.

    * Drops OpenAI-only settings the SDK adds for gpt-5 model names (verbosity, reasoning).
    * Gemini 3 rejects a tool call sent back without its thought signature. The SDK keeps the
      signature in the item's `provider_data`, but the harness decodes the model activity's
      input into typed items that have no such field, so it is lost. Tool calls without one get
      Gemini's placeholder, as the SDK's own LiteLLM model does.
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

    async def get_response(self, system_instructions, input, model_settings, **kwargs):  # noqa: A002
        return await self._inner.get_response(
            system_instructions=system_instructions,
            input=self._items(input),
            model_settings=self._portable(model_settings),
            **kwargs,
        )

    def stream_response(self, system_instructions, input, model_settings, **kwargs):  # noqa: A002
        return self._inner.stream_response(
            system_instructions=system_instructions,
            input=self._items(input),
            model_settings=self._portable(model_settings),
            **kwargs,
        )


class _RenamedModels(ModelProvider):
    """Serves every model request with one configured model of a compatible endpoint."""

    def __init__(self, inner: ModelProvider, model: str) -> None:
        self._inner = inner
        self._model = model

    def get_model(self, model_name: str | None) -> Model:
        name = self._model or model_name or ""
        return _PortableModel(self._inner.get_model(name), name)


def provider_kind() -> str:
    """The configured model option; the first enabled one wins, in this order."""
    if fake_model_enabled():
        return "scripted"
    if _flag("ACTION_AUTHOR_CHATGPT"):
        return "chatgpt"
    base_url = _env("ACTION_AUTHOR_MODEL_BASE_URL")
    if _flag("ACTION_AUTHOR_ANTHROPIC") or urlparse(base_url).hostname == "api.anthropic.com":
        return "anthropic"
    if base_url:
        return "compatible"
    return "openai"


def model_provider() -> ModelProvider | None:
    """Pick scripted, ChatGPT, native Claude, another compatible endpoint, or OpenAI."""
    kind = provider_kind()
    if kind == "scripted":
        return ScriptedModelProvider()
    if kind == "chatgpt":
        from .chatgpt_model import ChatGPTModelProvider

        return ChatGPTModelProvider(_env("ACTION_AUTHOR_CHATGPT_MODEL", DEFAULT_CHATGPT_MODEL))
    if kind == "anthropic":
        from .anthropic_model import AnthropicModelProvider

        key = _env("ACTION_AUTHOR_MODEL_API_KEY")
        if not key:
            raise ValueError("set ACTION_AUTHOR_MODEL_API_KEY for native Claude")
        return AnthropicModelProvider(
            _env("ACTION_AUTHOR_MODEL", DEFAULT_CLAUDE_MODEL),
            key,
            cache_ttl=_env("ACTION_AUTHOR_ANTHROPIC_CACHE_TTL", "5m"),
            context_tokens=int(_env("ACTION_AUTHOR_ANTHROPIC_CONTEXT_TOKENS", "30000")),
        )
    if kind == "compatible":
        client = AsyncOpenAI(
            base_url=_env("ACTION_AUTHOR_MODEL_BASE_URL"),
            api_key=_env("ACTION_AUTHOR_MODEL_API_KEY", "unused"),
            # Activity retries own the retry policy.
            max_retries=0,
        )
        # Compatible endpoints implement Chat Completions, not OpenAI's Responses API.
        provider = OpenAIProvider(openai_client=client, use_responses=False)
        return _RenamedModels(provider, _env("ACTION_AUTHOR_MODEL"))
    return None


def model_label() -> str:
    kind = provider_kind()
    if kind == "chatgpt":
        return f"ChatGPT subscription {_env('ACTION_AUTHOR_CHATGPT_MODEL', DEFAULT_CHATGPT_MODEL)}"
    if kind == "anthropic":
        return f"native Claude {_env('ACTION_AUTHOR_MODEL', DEFAULT_CLAUDE_MODEL)}"
    if kind == "compatible":
        model = _env("ACTION_AUTHOR_MODEL", "as requested")
        return f"{model} at {_env('ACTION_AUTHOR_MODEL_BASE_URL')}"
    return kind


def openai_plugin(device: Device, agent_device_command: list[str]) -> OpenAIAgentsPlugin:
    kind = provider_kind()
    retry_policy = None
    if kind == "chatgpt":
        retry_policy = RetryPolicy(maximum_attempts=3)
    elif kind == "anthropic":
        # Bad input and truncated replies cannot recover by issuing the same paid request.
        retry_policy = RetryPolicy(non_retryable_error_types=["UserError", "ModelBehaviorError"])
    return OpenAIAgentsPlugin(
        model_params=ModelActivityParameters(
            # Local models are slow; a turn's model call may take minutes.
            start_to_close_timeout=timedelta(
                minutes=10 if kind in {"anthropic", "compatible"} else 3
            ),
            retry_policy=retry_policy,
        ),
        model_provider=model_provider(),
        mcp_server_providers=[
            StatelessMCPServerProvider(
                AGENT_DEVICE_MCP, agent_device_factory(agent_device_command, device)
            )
        ],
    )


def device_from_env() -> tuple[Device, list[str]]:
    """The Android phone or iOS simulator and the agent-device MCP command, or offline fakes for
    the scripted model."""
    serial = os.environ.get("ACTION_AUTHOR_DEVICE_SERIAL", "").strip()
    simulator = os.environ.get("ACTION_AUTHOR_IOS_SIMULATOR_UDID", "").strip()
    if serial and simulator:
        sys.exit(
            "error: set only one of ACTION_AUTHOR_DEVICE_SERIAL and "
            "ACTION_AUTHOR_IOS_SIMULATOR_UDID: a worker drives one device"
        )
    if serial:
        return AndroidDevice(serial), AGENT_DEVICE_COMMAND
    if simulator:
        return IOSSimulatorDevice(simulator), AGENT_DEVICE_COMMAND
    if fake_model_enabled():
        return FakeDevice(), FAKE_AGENT_DEVICE_COMMAND
    sys.exit(
        "error: set ACTION_AUTHOR_DEVICE_SERIAL (adb devices), ACTION_AUTHOR_IOS_SIMULATOR_UDID "
        "(xcrun simctl list devices), or ACTION_AUTHOR_FAKE_MODEL=1"
    )


async def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
    if provider_kind() != "openai":
        # No API key to export traces with; keep OAuth traffic out of tracing.
        set_tracing_disabled(True)
    elif not os.environ.get("OPENAI_API_KEY"):
        sys.exit(
            "error: set OPENAI_API_KEY, ACTION_AUTHOR_CHATGPT=1, ACTION_AUTHOR_ANTHROPIC=1, "
            "ACTION_AUTHOR_MODEL_BASE_URL, or ACTION_AUTHOR_FAKE_MODEL=1"
        )
    device, command = device_from_env()
    simulator = device if isinstance(device, IOSSimulatorDevice) else None
    if simulator is not None:
        # The iOS simulator: refuse a running base, and drop clones a killed worker left.
        await simulator.check_base()
        for udid in await simulator.remove_stale_clones():
            print(f"action author worker: deleted stale simulator clone {udid}", flush=True)
    activities = ActionAuthorActivities(device)
    client = await Client.connect(
        **ClientConfig.load_client_connect_config(),
        plugins=[openai_plugin(device, command), AgentHarnessPlugin()],
    )
    if client.namespace == "default" and not fake_model_enabled():
        sys.exit(
            "error: set TEMPORAL_NAMESPACE to the selected wallet's organization namespace; "
            "Credimi starts authoring sessions there, not in the default namespace"
        )
    worker = Worker(
        client,
        task_queue=TASK_QUEUE,
        workflows=[ActionAuthorWorkflow],
        activities=activities.all(),
    )
    print(
        f"action author worker: namespace={client.namespace} task_queue={TASK_QUEUE} "
        f"model={model_label()} device={device.platform} {device.configured_device} "
        f"agent-device={' '.join(command)}",
        flush=True,
    )
    try:
        await worker.run()
    finally:
        if simulator is not None:
            await simulator.cleanup()


if __name__ == "__main__":
    asyncio.run(main())
