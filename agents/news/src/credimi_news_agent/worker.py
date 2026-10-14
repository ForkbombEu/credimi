# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Worker for the NewsAgent workflow and its Python activities.

Environment:
    TEMPORAL_ADDRESS / TEMPORAL_NAMESPACE   Temporal connection (defaults: localhost:7233, default)
    OPENAI_API_KEY                          required unless NEWS_AGENT_FAKE_MODEL=1
    NEWS_AGENT_FAKE_MODEL=1                 use the scripted model (no API key, no cost)
    NEWS_AGENT_FIXTURES_DIR / NEWS_AGENT_RECORD_DIR / GITHUB_TOKEN   see http.py
    NEWS_AGENT_SOURCES                      path of sources.toml (default: agents/news/sources.toml)

The Go activities this workflow calls run in `credimi serve` (Temporal workers enabled).
"""

import asyncio
import logging
import os
import sys
from datetime import timedelta

from agents import set_tracing_disabled
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


def openai_plugin() -> OpenAIAgentsPlugin:
    return OpenAIAgentsPlugin(
        model_params=ModelActivityParameters(
            start_to_close_timeout=timedelta(minutes=3),
        ),
        model_provider=ScriptedModelProvider() if fake_model_enabled() else None,
    )


async def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(levelname)s %(message)s")
    if fake_model_enabled():
        # Nothing to export to OpenAI without a key.
        set_tracing_disabled(True)
    elif not os.environ.get("OPENAI_API_KEY"):
        sys.exit("error: set OPENAI_API_KEY, or NEWS_AGENT_FAKE_MODEL=1 for the scripted model")

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
    print(
        f"news agent worker: namespace={client.namespace} task_queue={TASK_QUEUE} "
        f"model={'scripted' if fake_model_enabled() else 'openai'} "
        f"fixtures={os.environ.get('NEWS_AGENT_FIXTURES_DIR') or 'off'}",
        flush=True,
    )
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())
