# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""ActionAuthor: a Temporal Agent Harness agent that writes Maestro wallet actions.

Credimi authenticates session start and binds a selected wallet version and organization.
``ask`` explores its verified, locally installed build and authors an action. ``submit_action``
replays it twice from cleared app data before accepting and writing it to
``out/<appId>/<slot>.yaml``. A workflow-owned Accept/Refuse callback confirms the exact
reviewed candidate before persisting it; the model cannot approve or save.

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
    from agents.exceptions import AgentsException, MaxTurnsExceeded
    from agents.items import ItemHelpers
    from pydantic import StrictBool, TypeAdapter
    from temporal_agent_harness.ai_sdks.openai_agents.workflow import stateless_mcp_server
    from temporal_agent_harness.ai_sdks.openai_agents_harness import as_openai_agent_tools
    from temporal_agent_harness.harness import agent
    from temporal_agent_harness.harness.agent_protocol import (
        AgentConfig,
        ReplyDelta,
        TextMessage,
        TextReply,
        ToolApprovalPolicy,
        ToolEndEvent,
        ToolErrorEvent,
        ToolStartEvent,
    )
    from temporal_agent_harness.harness.agent_workflow import AgentWorkflowRunner, CallbackToolError
    from temporalio.workflow import ActivityConfig

    from .activities import DEVICE_INFO
    from .contracts import (
        GO_ACTIVITY_QUEUE,
        PLATFORM_LABELS,
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
        REPLAY_RUNS,
        SLOTS,
        WORKFLOW_TYPE,
        DeviceInfo,
        Usage,
    )
    from .tools import TOOLS, AuthorSession

INSTRUCTIONS = f"""\
You write Maestro wallet actions for Credimi, a conformance testing platform for EU Digital
Identity wallets. An action is a Maestro flow that brings a wallet app from the slot's
precondition to its outcome on a test device: an Android phone or an iOS simulator.

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
   - Selectors: prefer `id:` (Android resource-id, iOS accessibility identifier), then visible
     text; coordinates (`point:`) only as a last resort.
   - Embed the PIN the user supplied directly as a quoted string in each PIN `inputText`
     command (e.g. `inputText: "123456"`). Never use `${{PIN}}` or external substitution.
   - Wait with `extendedWaitUntil` (visible + timeout), not sleeps.
   - End with an assertion of the outcome (`assertVisible` or `extendedWaitUntil`), then
     `takeScreenshot`.
   - The replay starts from cleared app data: start with `launchApp`.
5. Call `check_action`, fix every problem, then `submit_action` with the same YAML.
   `submit_action` replays it twice on the device; read its report and fix the action if a
   replay failed. You have {MAX_SUBMISSIONS} submissions per target. Stop after ACCEPTED or when
   they are spent, and tell the user the result in a few lines.

Slots:
""" + "\n".join(
    f"- `{s.name}`: precondition: {s.precondition}; outcome: {s.outcome}; "
    f"parameters: {', '.join(s.required_params)}."
    for s in SLOTS.values()
)

# Platform rules appended to the instructions after the device line.
PLATFORM_NOTES = {
    "android": "",
    "ios": (
        "iOS notes: every `reset_app` and every replay starts on a fresh clone of the simulator "
        "with the wallet freshly installed. Maestro's `back` does nothing on iOS: navigate back "
        "with `tapOn` on the app's own back button. System alerts (permissions, notifications) "
        'are separate elements: tap their button by text, e.g. `tapOn: "Allow"`, wrapped in '
        "`runFlow: when: visible:` if they do not always appear.\n"
    ),
}


def _device_line(session: AuthorSession) -> str:
    if session.platform == "ios":
        return f"Device: iOS simulator, cloned from {session.serial}."
    return f"Device: Android phone, serial {session.serial}."


class TokenBudgetExceeded(AgentsException):
    """A turn used more model tokens than its budget allows."""


class TokenBudget(RunHooks):
    """Counts a run's model usage and stops it once input plus output tokens exceed ``limit``."""

    def __init__(self, limit: int) -> None:
        self.limit = limit
        self.usage = Usage()

    async def on_llm_end(self, context, agent, response) -> None:
        u = response.usage
        details = u.input_tokens_details
        self.usage = self.usage.add(
            Usage(
                requests=1,
                input_tokens=u.input_tokens,
                output_tokens=u.output_tokens,
                cache_read_tokens=getattr(details, "cached_tokens", 0) or 0,
                cache_write_tokens=getattr(details, "cache_write_tokens", 0) or 0,
                context_cleared_tokens=getattr(details, "context_cleared_tokens", 0) or 0,
            )
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
        self._stream = WorkflowStream()
        self._runner = AgentWorkflowRunner(
            config,
            stream=self._stream,
            # Exploration/replay are safe only on the dedicated test device. Saving
            # is a separate human handler, never an automatically approved model tool.
            approval_policy_default=ToolApprovalPolicy.allow_inherently_safe(),
        )
        self._session: AuthorSession | None = None
        self._conversation: list[TResponseInputItem] = []
        self._asks = 0
        self._context = context
        self._build: ResolvedBuild | None = None
        self._automatic_review = False

    @agent.teardown
    async def _drain_stream(self) -> None:
        if workflow.patched("action-author-drain-stream-on-close-v1"):
            self._stream.detach_pollers()
            await workflow.wait_condition(workflow.all_handlers_finished)

    @workflow.query
    def authoring_events(self, from_offset: int = 0) -> dict:
        """Read recorded events after completion, when stream polling updates are unavailable."""
        state = self._stream.get_state()
        start = max(0, from_offset - state.base_offset)
        page = state.log[start : start + 25]
        return {
            "items": [
                {"data": item.data, "offset": state.base_offset + start + index}
                for index, item in enumerate(page)
            ],
            "next_offset": state.base_offset + start + len(page),
            "end_offset": state.base_offset + len(state.log),
        }

    async def _load_session(self) -> AuthorSession:
        info: DeviceInfo = await workflow.execute_activity(
            DEVICE_INFO, start_to_close_timeout=timedelta(seconds=10), result_type=DeviceInfo
        )
        if info.platform != self._context.platform:
            raise ValueError(
                f"this worker drives an {PLATFORM_LABELS[info.platform]}, but the session "
                f"selected the {self._context.platform} build; start a new session for "
                f"platform {info.platform}"
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
        if build.platform != self._context.platform:
            raise ValueError("Credimi resolved a build for another platform")
        installed: InstalledBuild = await workflow.execute_activity(
            PREPARE_BUILD,
            PrepareBuildRequest(build=build, expected_app_id=self._context.app_id),
            start_to_close_timeout=timedelta(minutes=10),
            schedule_to_close_timeout=timedelta(minutes=11),
            heartbeat_timeout=timedelta(minutes=1),
            retry_policy=RetryPolicy(maximum_attempts=1),
            result_type=InstalledBuild,
        )
        if installed.installer_sha256 != build.installer_sha256:
            raise ValueError("prepared installer does not match the selected Credimi build")
        self._build = build.model_copy(update={"download_grant": ""})
        return AuthorSession(
            serial=info.serial,
            token_budget=info.token_budget,
            max_model_calls=info.max_model_calls,
            platform=info.platform,
            bound_app_id=installed.package_id,
            bound_slot=self._context.slot,
            installer_sha256=installed.installer_sha256,
        )

    def _agent_device(self, session: AuthorSession):
        # The worker pins the serial; the factory argument scopes `open` to the app under test.
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
        self._automatic_review = workflow.patched("action-author-automatic-save-review-v1")
        if self._automatic_review and self._session and self._session.accepted_yaml:
            return await self._review_save("The validated candidate is ready to save.")
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
                return TextReply(text=f"Cannot prepare the selected Credimi build: {exc.cause}")
            except ValueError as exc:
                # Includes Pydantic validation of the resolved build; a bare exception would
                # fail the workflow task and retry this update forever.
                return TextReply(text=f"Cannot prepare the selected Credimi build: {exc}")
        session = self._session
        accepted_before = session.accepted_yaml
        state = session.describe()
        user_text = message.text
        # Signed thinking binds to the prefix, including system instructions.
        # Put mutable authoring state in the new user turn instead.
        if workflow.patched("action-author-stable-model-instructions-v1"):
            user_text += f"\n\nCurrent authoring state: {state}"
            state = ""
        if self._automatic_review:
            user_text += (
                "\n\nIf the user explicitly requested an action name (for example, "
                "'write action \"First setup\"' or 'store it as \"Demo onboarding\"'), pass "
                "that exact name as set_target.action_name. Otherwise omit action_name: "
                "the default is Onboarding. After acceptance the workflow presents the "
                "human Accept/Refuse save confirmation automatically. Do not tell the "
                "user to copy a hash, choose a handler, or fill a save form."
            )
        sdk_agent = OpenAIAgent(
            name="CredimiActionAuthor",
            instructions=f"{INSTRUCTIONS}\n\n{_device_line(session)}\n"
            f"{PLATFORM_NOTES[session.platform]}"
            f"{state}"
            f"\nSelected build: {self._context.wallet_version_identifier}; "
            f"installer SHA256 {session.installer_sha256}; app {session.bound_app_id}. "
            + (
                "Never target another build. After two successful replays, the workflow "
                "will ask the user to Accept or Refuse saving. Do not request manual save fields."
                if self._automatic_review
                else "Never target another build. "
                "Saving requires the human save_wallet_action handler."
            ),
            model=DEFAULT_MODEL,
            tools=_adapt_tools(self._runner, TOOLS, session),
            mcp_servers=[self._agent_device(session)],
        )
        budget = TokenBudget(session.token_budget)
        error = ""
        try:
            result = await Runner.run(
                sdk_agent,
                input=[*self._conversation, {"role": "user", "content": user_text}],
                max_turns=session.max_model_calls,
                hooks=budget,
            )
        except AgentsException as exc:
            # Budget hooks stop before the new response's tools execute, and the turn limit
            # stops before the next model call. Keep only the SDK's completed items so a
            # human can continue from the actual device state.
            resumable = (
                isinstance(exc, TokenBudgetExceeded | MaxTurnsExceeded) and exc.run_data is not None
            )
            if resumable:
                self._conversation = ItemHelpers.input_to_new_input_list(exc.run_data.input)
                self._conversation.extend(item.to_input_item() for item in exc.run_data.new_items)
            error = f"{type(exc).__name__}: {exc}"
            text = f"I could not finish: {error}"
            if resumable:
                text += (
                    "\nCompleted exploration is retained. Send Continue to resume; "
                    "the next message has its own budgets."
                )
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
        if session.accepted_yaml and session.accepted_yaml != accepted_before:
            digest = hashlib.sha256(session.accepted_yaml.encode()).hexdigest()
            text += (
                f"\n\n**Accepted action** (`{session.accepted_path}`):\n\n"
                f"```yaml\n{session.accepted_yaml.rstrip()}\n```"
                f"\n\nCandidate SHA256: `{digest}`."
                f"\nValidated build: `{self._context.wallet_version_identifier}`; "
                f"installer SHA256: `{session.installer_sha256}`. "
                + (
                    "Review the YAML, then Accept or Refuse saving in the confirmation below."
                    if self._automatic_review
                    else "Review the YAML and replays, then select the save_wallet_action handler "
                    "and confirm this candidate hash to save into Credimi."
                )
            )
        text += (
            f"\n\n_Message {self._asks} of {CHAT_MAX_MESSAGES}. This answer: {usage.text()}. "
            f"Session: {session.usage.text()}._"
        )
        if self._automatic_review and session.accepted_yaml:
            return await self._review_save(text)
        return TextReply(text=text)

    async def _review_save(self, text: str) -> TextReply:
        session = self._session
        build = self._build
        if session is None or build is None or session.accepted_replays != REPLAY_RUNS:
            return TextReply(text="Not saved: no candidate has passed both replays.")
        name = session.action_name
        digest = hashlib.sha256(session.accepted_yaml.encode()).hexdigest()
        proposal = {
            "name": name,
            "wallet": build.wallet_id,
            "build": build.version_identifier,
            "category": session.bound_slot,
            "candidate_sha256": digest,
            "replay_count": session.accepted_replays,
            "published": False,
        }
        self._runner.publish(ReplyDelta(text=text + "\n\nConfirm whether to save this action."))
        while True:
            tool_id = f"save-review-{workflow.uuid4()}"
            self._runner.publish(
                ToolStartEvent(
                    tool_id=tool_id,
                    tool_name="confirm_wallet_action_save",
                    tool_input=proposal,
                )
            )
            try:
                accepted = await self._runner.await_callback_result(
                    tool_id=tool_id,
                    tool_name="confirm_wallet_action_save",
                    tool_input=proposal,
                    output_adapter=TypeAdapter(StrictBool),
                    output_schema={"type": "boolean"},
                    timeout=None,
                )
            except CallbackToolError:
                self._runner.publish(
                    ToolErrorEvent(
                        tool_id=tool_id,
                        tool_name="confirm_wallet_action_save",
                        message="Save confirmation was canceled.",
                    )
                )
                return TextReply(text=text + "\n\nNot saved: confirmation was canceled.")
            self._runner.publish(
                ToolEndEvent(
                    tool_id=tool_id,
                    tool_name="confirm_wallet_action_save",
                    tool_output="Accepted" if accepted else "Refused",
                )
            )
            if not accepted:
                session.discard_candidate()
                if workflow.patched("action-author-refusal-restores-submissions-v1"):
                    # A refusal on the last submission must not leave the chat unable to revise.
                    session.attempts = 0
                return TextReply(text=text + "\n\nNot saved. You can request changes in this chat.")
            saved, receipt = await self._persist_candidate(
                SaveActionApproval(name=name, candidate_sha256=digest, confirm=True)
            )
            if saved:
                return TextReply(text=f"{text}\n\n{receipt}")
            self._runner.publish(
                ReplyDelta(
                    text=f"\n\n{receipt}\nAccept to retry saving, or Refuse to keep editing."
                )
            )

    @agent.accepts
    async def save_wallet_action(self, approval: SaveActionApproval) -> TextReply:
        """Replay-compatible legacy save handler; new turns use the Accept/Refuse callback."""
        if self._automatic_review:
            return TextReply(text="Use the candidate's Accept/Refuse confirmation to save.")
        _, receipt = await self._persist_candidate(approval)
        return TextReply(text=receipt)

    async def _persist_candidate(self, approval: SaveActionApproval) -> tuple[bool, str]:
        """Save the accepted candidate; (saved, receipt or reason it was not saved)."""
        session, build = self._session, self._build
        if not approval.confirm:
            return False, "Not saved: human confirmation was not given."
        if (
            session is None
            or build is None
            or not session.accepted_yaml
            or session.accepted_replays != REPLAY_RUNS
        ):
            return False, "Not saved: no candidate has passed both replays."
        digest = hashlib.sha256(session.accepted_yaml.encode()).hexdigest()
        if approval.candidate_sha256 != digest:
            return False, "Not saved: candidate hash changed; review the current accepted YAML."
        if bool(approval.replace_action_id) != bool(approval.expected_existing_sha256):
            return (
                False,
                "Not saved: replacement requires the record ID and its current code SHA256.",
            )
        payload = {
            **self._context.build_payload(),
            "name": approval.name,
            "code": session.accepted_yaml,
            "candidate_sha256": digest,
            "installer_sha256": build.installer_sha256,
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
            return False, f"Not saved: {exc.cause}"
        receipt = result["output"]
        status = "published" if receipt["published"] else "unpublished"
        if workflow.patched("action-author-close-after-save-v1"):
            # Invoke the harness's registered close handler locally. Signaling our own
            # execution through SignalExternalWorkflow is rejected by the real server.
            close = workflow.get_signal_handler("close")
            if close is None:
                raise RuntimeError("the harness close handler is not registered")
            close()
        return True, (
            f"Saved wallet action `{receipt['record_id']}` ({status}). "
            f"Validated build `{build.version_identifier}`, candidate SHA256 `{digest}`."
        )
