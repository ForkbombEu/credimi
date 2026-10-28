# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""ActionAuthor: a Temporal Agent Harness agent that writes Maestro wallet actions.

Credimi authenticates session start and binds a selected wallet version and organization.
``ask`` explores its verified, locally installed APK and authors an action. ``submit_action``
replays it twice from cleared app data before accepting and writing it to
``out/<appId>/<slot>.yaml``. The human-only ``save_wallet_action`` handler persists the exact
reviewed candidate; the model cannot save.

The OpenAI Agents SDK drives the reason/act loop (model calls and MCP calls are Temporal
activities). The conversation and the session state persist across messages.
"""

import dataclasses
import hashlib
from datetime import timedelta

from temporalio import workflow
from temporalio.common import RetryPolicy
from temporalio.contrib.workflow_streams import WorkflowStream
from temporalio.exceptions import ActivityError, ApplicationError

with workflow.unsafe.imports_passed_through():
    # Pydantic's payload converter loads this lazily when decoding workflow updates.
    # Preload it through the sandbox boundary, before any workflow activation.
    import annotated_types  # noqa: F401
    from agents import Agent as OpenAIAgent
    from agents import RunHooks, Runner, TResponseInputItem
    from agents.exceptions import AgentsException
    from temporal_agent_harness.ai_sdks.openai_agents.workflow import stateless_mcp_server
    from temporal_agent_harness.ai_sdks.openai_agents_harness import as_openai_agent_tools
    from temporal_agent_harness.harness import agent
    from temporal_agent_harness.harness.agent_protocol import (
        AgentConfig,
        TextMessage,
        TextReply,
        ToolApprovalPolicy,
        ToolErrorEvent,
    )
    from temporal_agent_harness.harness.agent_workflow import AgentWorkflowRunner
    from temporalio.workflow import ActivityConfig

    from .activities import DEVICE_INFO
    from .contracts import (
        GO_ACTIVITY_QUEUE,
        PREPARE_BUILD,
        RESOLVE_BUILD,
        SAVE_ACTION,
        AuthoringContext,
        InstalledBuild,
        PrepareBuildRequest,
        ResolvedBuild,
        SaveActionApproval,
    )
    from .models import (
        AGENT_DEVICE_MCP,
        CHAT_MAX_MESSAGES,
        DEFAULT_MODEL,
        MAX_SUBMISSIONS,
        SLOTS,
        WORKFLOW_TYPE,
        DeviceInfo,
        Usage,
    )
    from .tools import TOOLS, AuthorSession

INSTRUCTIONS = f"""\
You write Maestro wallet actions for Credimi, a conformance testing platform for EU Digital
Identity wallets. An action is a Maestro flow that brings a wallet app from the slot's
precondition to its outcome on a real Android phone.

How to work:
1. Call `set_target` with the app id, the slot and every parameter value the user gave. If
   something is missing or ambiguous, ask the user instead of guessing.
2. Explore the app step by step with the agent-device tools: `reset_app` (fresh state, the
   slot precondition), `open` the app, `snapshot`, then `press`/`click`/`fill`/`type`/`scroll`/
   `swipe`/`back`/`wait`/`find`/`get`/`is`. Follow each screen until the outcome is reached.
3. Close the agent-device session (`close` without arguments) before the Maestro tools; they
   close it anyway. `maestro_inspect_screen` shows the selectors Maestro will match on the
   current screen; `maestro_cheat_sheet` shows the syntax.
4. Write the action: a header with `appId` (and optionally name, tags), `---`, then commands.
   - Selectors: prefer `id:` (resource-id), then visible text; coordinates (`point:`) only as a
     last resort.
   - Embed the PIN the user supplied directly as a quoted string in each PIN `inputText`
     command (e.g. `inputText: "123456"`). Never use `${{PIN}}` or external substitution.
     Device-tool replies mask PIN values; use the original value from the user's request.
   - Wait with `extendedWaitUntil` (visible + timeout), not sleeps.
   - End with an assertion of the outcome (`assertVisible` or `extendedWaitUntil`), then
     `takeScreenshot`.
   - The replay starts from cleared app data: start with `launchApp`.
5. Call `check_action`, fix every problem, then `submit_action` with the same YAML.
   `submit_action` replays it twice on the phone; read its report and fix the action if a
   replay failed. You have {MAX_SUBMISSIONS} submissions per target. Stop after ACCEPTED or when
   they are spent, and tell the user the result in a few lines.

Slots:
""" + "\n".join(
    f"- `{s.name}`: precondition: {s.precondition}; outcome: {s.outcome}; "
    f"parameters: {', '.join(s.required_params)}."
    for s in SLOTS.values()
)


class TokenBudgetExceeded(AgentsException):
    """A turn used more model tokens than its budget allows."""


class TokenBudget(RunHooks):
    """Counts a run's model usage and stops it once input plus output tokens exceed ``limit``."""

    def __init__(self, limit: int) -> None:
        self.limit = limit
        self.usage = Usage()

    async def on_llm_end(self, context, agent, response) -> None:
        u = response.usage
        self.usage = self.usage.add(
            Usage(requests=1, input_tokens=u.input_tokens, output_tokens=u.output_tokens)
        )
        used = self.usage.input_tokens + self.usage.output_tokens
        if used > self.limit:
            raise TokenBudgetExceeded(f"token budget of {self.limit} tokens exceeded: {used} used")


def _adapt_tools(runner: AgentWorkflowRunner, tools: list, session: AuthorSession) -> list:
    """Harness tools for the SDK, with invalid arguments sent back to the model.

    The harness adapter raises a non-retryable `InvalidToolInput` when the model's arguments
    do not match the tool schema, which fails the whole turn. Models without strict schema
    enforcement (e.g. through Ollama) omit fields, so the error becomes the tool's reply and
    the model can call again. The error is also published as a TOOL_ERROR event, so the
    harness UI and CLI show it.
    """
    adapted = as_openai_agent_tools(runner, tools, injections={"session": session})
    return [dataclasses.replace(t, on_invoke_tool=_reply_invalid_input(runner, t)) for t in adapted]


def _reply_invalid_input(runner: AgentWorkflowRunner, tool):
    invoke = tool.on_invoke_tool

    async def on_invoke_tool(ctx, input: str):  # noqa: A002
        try:
            return await invoke(ctx, input)
        except ApplicationError as exc:
            if exc.type != "InvalidToolInput":
                raise
            try:
                runner.publish(
                    ToolErrorEvent(
                        tool_id=getattr(ctx, "tool_call_id", "") or f"invalid-{tool.name}",
                        tool_name=tool.name,
                        message=exc.message,
                    )
                )
            except RuntimeError:
                pass  # no active turn to publish against
            return (
                f"ERROR: invalid arguments for {tool.name}: {exc.message}\n"
                "Call it again with every required field."
            )

    return on_invoke_tool


@agent.defn(name=WORKFLOW_TYPE)
class ActionAuthorWorkflow:
    @agent.init
    def __init__(self, config: AgentConfig, context: AuthoringContext) -> None:
        self._runner = AgentWorkflowRunner(
            config,
            stream=WorkflowStream(),
            # Exploration/replay are safe only on the dedicated test device. Saving
            # is a separate human handler, never an automatically approved model tool.
            approval_policy_default=ToolApprovalPolicy.allow_inherently_safe(),
        )
        self._session: AuthorSession | None = None
        self._conversation: list[TResponseInputItem] = []
        self._asks = 0
        self._context = context
        self._build: ResolvedBuild | None = None

    async def _load_session(self) -> AuthorSession:
        info: DeviceInfo = await workflow.execute_activity(
            DEVICE_INFO, start_to_close_timeout=timedelta(seconds=10), result_type=DeviceInfo
        )
        result = await workflow.execute_activity(
            RESOLVE_BUILD,
            {"config": {}, "payload": self._context.build_payload()},
            task_queue=GO_ACTIVITY_QUEUE,
            start_to_close_timeout=timedelta(minutes=2),
            schedule_to_close_timeout=timedelta(minutes=3),
            retry_policy=RetryPolicy(maximum_attempts=2),
        )
        build = ResolvedBuild.model_validate(result["output"])
        if build.owner_namespace != workflow.info().namespace:
            raise ValueError("selected build does not belong to the authoring namespace")
        installed: InstalledBuild = await workflow.execute_activity(
            PREPARE_BUILD,
            PrepareBuildRequest(build=build, expected_app_id=self._context.app_id),
            start_to_close_timeout=timedelta(minutes=5),
            schedule_to_close_timeout=timedelta(minutes=6),
            heartbeat_timeout=timedelta(minutes=1),
            retry_policy=RetryPolicy(maximum_attempts=1),
            result_type=InstalledBuild,
        )
        if installed.apk_sha256 != build.apk_sha256:
            raise ValueError("prepared APK does not match the selected Credimi build")
        self._build = build.model_copy(update={"download_grant": ""})
        return AuthorSession(
            serial=info.serial,
            token_budget=info.token_budget,
            max_model_calls=info.max_model_calls,
            bound_app_id=installed.package_id,
            bound_slot=self._context.slot,
            apk_sha256=installed.apk_sha256,
        )

    def _agent_device(self, session: AuthorSession):
        # The worker pins the serial; the factory argument scopes `open` to the app under test
        # and masks the parameter values in the outputs.
        return stateless_mcp_server(
            AGENT_DEVICE_MCP,
            config=ActivityConfig(
                start_to_close_timeout=timedelta(minutes=3),
                schedule_to_close_timeout=timedelta(minutes=3),
                retry_policy=RetryPolicy(maximum_attempts=1),
            ),
            cache_tools_list=True,
            factory_argument=session.device_context,
            runner=self._runner,
            inherently_safe=True,
        )

    @agent.accepts
    async def ask(self, message: TextMessage) -> TextReply:
        """Ask for a wallet action, e.g. "Write the onboarding action for eu.europa.ec.euidi with
        PIN 123456". The agent explores the app on the phone, then replays its action twice."""
        if self._asks >= CHAT_MAX_MESSAGES:
            return TextReply(
                text=f"This session reached its limit of {CHAT_MAX_MESSAGES} messages. "
                "Start a new session to continue."
            )
        self._asks += 1
        if self._session is None:
            try:
                self._session = await self._load_session()
            except ActivityError as exc:
                return TextReply(text=f"Cannot prepare the selected Credimi APK: {exc.cause}")
        session = self._session
        accepted_before = session.accepted_yaml
        sdk_agent = OpenAIAgent(
            name="CredimiActionAuthor",
            instructions=f"{INSTRUCTIONS}\n\nDevice: Android, serial {session.serial}.\n"
            f"{session.describe()}"
            f"\nSelected build: {self._context.wallet_version_identifier}; "
            f"APK SHA256 {session.apk_sha256}; package {session.bound_app_id}. "
            "Never target another build. Saving requires the human save_wallet_action handler.",
            model=DEFAULT_MODEL,
            tools=_adapt_tools(self._runner, TOOLS, session),
            mcp_servers=[self._agent_device(session)],
        )
        budget = TokenBudget(session.token_budget)
        error = ""
        try:
            result = await Runner.run(
                sdk_agent,
                input=[*self._conversation, {"role": "user", "content": message.text}],
                max_turns=session.max_model_calls,
                hooks=budget,
            )
        except AgentsException as exc:
            # e.g. MaxTurnsExceeded, TokenBudgetExceeded: the conversation is left as it was;
            # an action accepted before the failure stays accepted.
            error = f"{type(exc).__name__}: {exc}"
            text = f"I could not finish: {error}"
        else:
            self._conversation = result.to_input_list()
            text = str(result.final_output or "").strip() or "(The model gave no answer.)"
        usage = budget.usage
        session.usage = session.usage.add(usage)
        workflow.logger.info(
            "ask %d: %s; session total %s%s",
            self._asks,
            usage.text(),
            session.usage.text(),
            f"; {error}" if error else "",
        )
        text = session.mask(text)
        if session.accepted_yaml and session.accepted_yaml != accepted_before:
            digest = hashlib.sha256(session.accepted_yaml.encode()).hexdigest()
            text += (
                f"\n\n**Accepted action** (`{session.accepted_path}`):\n\n"
                f"```yaml\n{session.accepted_yaml.rstrip()}\n```"
                f"\n\nCandidate SHA256: `{digest}`."
                f"\nValidated build: `{self._context.wallet_version_identifier}`; "
                f"APK SHA256: `{session.apk_sha256}`. "
                "Review the YAML and replays, then select the save_wallet_action handler "
                "and confirm this candidate hash to save into Credimi."
            )
        text += (
            f"\n\n_Message {self._asks} of {CHAT_MAX_MESSAGES}. This answer: {usage.text()}. "
            f"Session: {session.usage.text()}._"
        )
        return TextReply(text=text)

    @agent.accepts
    async def save_wallet_action(self, approval: SaveActionApproval) -> TextReply:
        """Human approval: save the reviewed, replayed candidate. Not callable by the model.

        Copy Candidate SHA256 from the accepted reply. A replacement also requires
        the existing record ID and its current code SHA256; it is never selected by name.
        """
        session, build = self._session, self._build
        if not approval.confirm:
            return TextReply(text="Not saved: human confirmation was not given.")
        if (
            session is None
            or build is None
            or not session.accepted_yaml
            or session.accepted_replays != 2
        ):
            return TextReply(text="Not saved: no candidate has passed both replays.")
        digest = hashlib.sha256(session.accepted_yaml.encode()).hexdigest()
        if approval.candidate_sha256 != digest:
            return TextReply(
                text="Not saved: candidate hash changed; review the current accepted YAML."
            )
        if bool(approval.replace_action_id) != bool(approval.expected_existing_sha256):
            return TextReply(
                text="Not saved: replacement requires the record ID and its current code SHA256."
            )
        payload = {
            **self._context.build_payload(),
            "name": approval.name,
            "code": session.accepted_yaml,
            "candidate_sha256": digest,
            "apk_sha256": build.apk_sha256,
            "package_id": session.bound_app_id,
            "slot": session.bound_slot,
            "replay_count": session.accepted_replays,
            "replace_action_id": approval.replace_action_id,
            "expected_existing_sha256": approval.expected_existing_sha256,
        }
        try:
            result = await workflow.execute_activity(
                SAVE_ACTION,
                {"config": {}, "payload": payload},
                task_queue=GO_ACTIVITY_QUEUE,
                start_to_close_timeout=timedelta(minutes=2),
                schedule_to_close_timeout=timedelta(minutes=3),
                retry_policy=RetryPolicy(maximum_attempts=3),
            )
        except ActivityError as exc:
            return TextReply(text=f"Not saved: {exc.cause}")
        receipt = result["output"]
        status = "published" if receipt["published"] else "unpublished"
        return TextReply(
            text=f"Saved wallet action `{receipt['record_id']}` ({status}). "
            f"Validated build `{build.version_identifier}`, candidate SHA256 `{digest}`."
        )
