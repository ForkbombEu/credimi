<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Action author agent (phase 1, local execution)

An agent that writes Maestro **wallet actions** for Credimi. From a chat message such as

> Write the onboarding action for eu.europa.ec.euidi with PIN 123456

it resolves a selected Credimi wallet version, verifies and installs its Android APK on the
worker's test phone, explores the wallet, writes a Maestro flow, and replays it twice from
cleared app data. The accepted YAML is shown in the reply and written to
`out/<appId>/<slot>.yaml` (gitignored). A separate human confirmation saves the exact
reviewed candidate into Credimi as an unpublished `wallet_actions` record.

An OpenAI Agents SDK agent, hosted by the
[Temporal Agent Harness](https://github.com/temporal-community/temporal-agent-harness)
(`temporal-agent-harness==0.8.0`, pre-alpha). The initial PoC pinned 0.6.0; the
action-author dependency is now upgraded independently of the news agent.

Credimi starts the Python `ActionAuthor` workflow on task queue `action-author` in the
selected wallet owner's organization namespace. Go activities on that namespace's
`PipelineTaskQueue` hold `core.App`: they authorize the actor, resolve the selected build
and persist approved actions. The Python worker does not receive a Credimi API key.
The harness owns the model/tool loop; device calls remain separate Temporal activities.
Device execution is local in this phase, not on Credimi runners.

The UI uses its own `action-author-session-manager` workflow, so an existing news-agent
session manager cannot replace its registry. Both managers use the harness's session-manager
task queue.

The 0.8 registry reads `action-author.schema.json`, generated from the workflow by
`make action-author.schema`, instead of importing the Python class in the web server.
The UI and test targets regenerate this file automatically. The packaged 0.8 UI supplies
the session picker, structured forms, timeline and reload-resilient session rendering;
we retain the custom launcher to keep this agent's session manager separate.

After upgrading, restart both `make action-author.worker` and `make action-author.ui`
to load the new package and browser assets. The existing ChatGPT subscription provider,
device pinning, validation and replay rules are unchanged.

The workflow preloads Pydantic's `annotated_types` metadata dependency through its
sandbox pass-through imports before updates execute. To detect late-import regressions:

```sh
cd agents/action-author
uv run pytest -q tests/test_workflow.py -W 'error:Module .* was imported after initial workflow load.:UserWarning'
```

## Setup

Requirements on the worker host: `uv`, `adb`, `apkanalyzer` (Android SDK command-line tools),
`agent-device` (0.21.x, `npm i -g agent-device`), `maestro` CLI, a dedicated phone with USB
debugging, Credimi with its organization Go workers enabled, and Temporal
(`make dev`, or infrastructure through `scripts/worktree-compose.sh up`).

```sh
cp agents/action-author/.env.example agents/action-author/.env
# set ACTION_AUTHOR_DEVICE_SERIAL (adb devices) and one model option
```

Models, chosen in one place (`worker.model_provider`):

| Env | Model |
| --- | --- |
| `ACTION_AUTHOR_CHATGPT=1` | local browser OAuth against your ChatGPT subscription; Responses API, no API-key billing |
| `ACTION_AUTHOR_FAKE_MODEL=1` (or `FAKE=1` on make) | scripted, no key (see `fake_model.py`) |
| `ACTION_AUTHOR_MODEL_BASE_URL` (+ `_API_KEY`, `ACTION_AUTHOR_MODEL`) | OpenAI-compatible Chat Completions: Anthropic, Ollama, freellmapi, Gemini |
| `OPENAI_API_KEY` | OpenAI (`gpt-5.1`) |

For Claude evaluation, use a workspace-scoped Claude API key through
[Anthropic's OpenAI-compatible endpoint](https://platform.claude.com/docs/en/cli-sdks-libraries/libraries/openai-sdk):

```dotenv
ACTION_AUTHOR_CHATGPT=0
ACTION_AUTHOR_FAKE_MODEL=0
ACTION_AUTHOR_MODEL_BASE_URL=https://api.anthropic.com/v1/
ACTION_AUTHOR_MODEL_API_KEY=your-workspace-scoped-key
ACTION_AUTHOR_MODEL=claude-sonnet-4-6
```

This uses Claude API billing, not the ChatGPT subscription. The current configuration does
not expose the extra workspace header required by non-workspace-scoped personal/service
keys. This is compatibility-mode evaluation, not a native Anthropic adapter: prompt caching
is unavailable through that endpoint. The action-author tests cover the compatible request
path; a live Claude authoring run has not yet been verified.

For the ChatGPT subscription, run `make action-author.login` once in a local terminal and
approve ChatGPT plan usage for `credimi-action-author`. This uses OpenAI's documented OAuth
flow for local/open-source apps, **not** Codex CLI or its cached credentials. The credential
file is owner-only at `~/.config/credimi-action-author/chatgpt.json`; override with
`ACTION_AUTHOR_CHATGPT_CREDENTIALS`. The worker refreshes tokens without putting them in
Temporal history. Set `ACTION_AUTHOR_CHATGPT_MODEL` to override the default `gpt-6.1-sol`.
Login URLs never contain saved ID tokens. Credential files are created atomically with `0600`
permissions; new credential directories use `0700`, while existing parent permissions stay unchanged.
Subscription requests use streamed Responses (`store: false`) with local function definitions in
an `additional_tools` developer input item. This preserves the bare tool names registered by
the harness; adding a wire-only namespace breaks SDK dispatch. The harness still owns
model/tool execution. This option is for local tests, not a hosted multi-user deployment.
The provider removes the SDK's inherited `reasoning.effort: none` override when selecting a
subscription model, allowing that model's supported default; explicit reasoning efforts remain.

Budgets per `ask` message: `ACTION_AUTHOR_TOKEN_BUDGET` (default 600 000 input+output tokens)
and `ACTION_AUTHOR_MAX_MODEL_CALLS` (default 80). Every reply ends with the model calls and
tokens in/out of the message and of the session; the worker logs the same line per message.

Agent-device MCP activities have one attempt and a three-minute total deadline, including queue
wait. Device interactions are not automatically retried: a lost response must not duplicate a
tap or text entry. After updating the worker, stop old authoring sessions and create a fresh
session; already scheduled activities keep their original inputs and retry policy.

## Run

Use a dedicated test phone: installation replaces the selected package, and authoring/replay
clears its data. Stop older authoring workers and sessions before testing this version.
In Credimi, create or select a wallet version with an Android APK in an organization you own;
copy its canonical `organization/wallet/version` identifier. Set `TEMPORAL_NAMESPACE` to that
organization's canonical name in `agents/action-author/.env`, alongside the device serial
and one model option.

Start `make dev` in a separate terminal if the API, UI and Temporal are not already running.
Do not use `make dev.noworkers`: phase 1 needs the organization Go activities.

```sh
# Both processes must use the selected wallet owner's organization namespace.
TEMPORAL_NAMESPACE=your-org make action-author.worker
TEMPORAL_NAMESPACE=your-org make action-author.ui
make action-author.schema            # regenerate the schema consumed by agents.toml
make action-author.test              # ruff + pytest, no physical device required
```

Start a session through Credimi, not through the harness's unauthenticated session creator.
Create a Credimi user API key in the dashboard, or use an existing one for your owner account.
This is separate from the Claude API key configured on the worker. In a Bash terminal:

```bash
read -rsp 'Credimi user API key: ' CREDIMI_USER_API_KEY; echo
curl --fail-with-body http://localhost:8090/api/action-author/sessions \
  -H "Credimi-Api-Key: $CREDIMI_USER_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"wallet_version_identifier":"your-org/your-wallet/your-version","slot":"onboarding"}'
unset CREDIMI_USER_API_KEY
```

The authenticated caller must be an owner of the wallet's organization (or a superuser).
The response is `202` with `workflow_id`, `run_id`, `workflow_namespace` and the canonical
`wallet_version_identifier`. Optional `app_id` rejects a selected APK with a different
package ID; it does not select an installer. Set `ACTION_AUTHOR_CREDIMI_URL` to the trusted
Credimi origin when it is not `http://localhost:8090`.

Open `http://localhost:8001/?s=<workflow_id>` and send the authoring request. The local UI
rejects direct session creation, so actor and organization cannot be supplied through its
New session form. Keep the UI on loopback: it is not a hosted multi-user authentication layer.

The selected APK is streamed through `GET /api/action-author/installer` using a short-lived,
single-build `Credimi-Artifact-Grant` header. Downloads do not follow redirects or use an
environment proxy; the worker checks SHA256, package/version metadata and the installed
APK's SHA256. Later replays reject an installed APK that has changed.

After two successful replays, review the exact YAML and Candidate SHA256 in the reply.
Select the `save_wallet_action` handler, provide a name and that hash, and set `confirm`.
Saving is not a model tool. Missing confirmation, a stale candidate, or an unreplayed
candidate cannot save. New records are unpublished. Replacing an action additionally requires
its `replace_action_id` and the SHA256 of its current code in `expected_existing_sha256`;
a changed record conflicts instead of being overwritten. The existing publication state
is retained on explicit replacement.

Verify the saved record in the selected wallet's actions: code must match the accepted YAML,
category must be `onboarding`, and a newly created record must remain unpublished. Ordinary
PocketBase reads hide provenance; the confirmation reply identifies the validated build
and candidate hash. Stop the authoring session and then the local UI/worker when finished.

Saved provenance records the wallet version, APK/package, YAML hash, workflow/run and two
successful replays. It is hidden from ordinary PocketBase responses and non-superuser writes;
PocketBase silently ignores attempts to write hidden fields. Editing the action's code
through the native API clears provenance that no longer matches.

For deterministic tests, `FAKE=1` selects the scripted model. Without a configured serial,
the device is also fake; it does not demonstrate real-phone installation or replay.

## Tools

| Tool | What it does |
| --- | --- |
| `set_target(app_id, slot, parameters)` | sets the app, the slot and the parameter values (session state, hidden from the model) |
| agent-device: `open snapshot press click fill type scroll swipe back wait find get is screenshot close` | exploration over MCP (`agent-device mcp`) |
| `reset_app()` | closes the agent-device session and clears the app's data (`pm clear`): the slot precondition |
| `maestro_cheat_sheet()` | Maestro's syntax reference (Maestro MCP `cheat_sheet`) |
| `maestro_inspect_screen()` | closes the agent-device session, then Maestro MCP `inspect_screen`, compacted to one line per targetable node (text, resource-id, content-desc, hint or clickable) with bounds, at most 8 000 characters |
| `check_action(yaml)` | validation only (below): OK or the list of problems |
| `submit_action(yaml)` | validation, then two replays; both must pass. At most 3 per target |

The agent-device MCP server is wrapped on the worker (`device.AgentDeviceServer`):

- only the 15 tools above are listed; any other tool name is refused;
- device selection is pinned by the worker: `serial`, `platform`, `session` (and the other
  device/tenant fields) are removed from the schemas the model sees and overwritten on every
  call, and the process runs with `AGENT_DEVICE_ANDROID_DEVICE_ALLOWLIST=<serial>`;
- `open` only opens the app under test (no URLs, no Metro hints);
- text outputs are masked (parameter values become `${NAME}`) and capped at 20 000 characters.

The shared MCP context is refreshed by each successful `set_target`, including changes within
one chat turn. Device operations are refused until a target has been selected.
Structured UI `target` values (ref, selector or point) pass through unchanged; only string
device-form targets (`mobile`, `tv`, `desktop`) are removed to preserve device pinning.

Device turn-taking: Maestro's Android driver does not start while an agent-device session holds
the device, so every Maestro step first runs `agent-device close --session action-author`
(this ends the session; the app stays in the foreground). Only the structured `SESSION_NOT_FOUND`
error is tolerated; other close failures stop the Maestro handoff. CLI processes run in their
own process group, which is killed and reaped on timeout or cancellation.

### Replay (`submit_action`)

For each of 2 runs: `agent-device close`, `adb -s <serial> shell pm clear <appId>`, write the
YAML to a temporary file, `maestro --device <serial> test <file>` (timeout 5 minutes).
The requested PIN is embedded as a quoted literal in the action; there is no external `-e`
substitution. The per-command
`... COMPLETED|FAILED|SKIPPED` lines are parsed; a failure returns the failing command, the last
30 output lines and the compacted Maestro view of the screen at the failure. Two passes →
ACCEPTED: written to `out/<appId>/<slot>.yaml` and shown in the reply. The replay is one
activity (15 minutes, heartbeats, no retry).

## Validation

`validation.py`, in code, before any replay:

1. exactly two YAML documents: header mapping, `---`, command list;
2. header keys must be strings in {appId, name, tags, env}; `appId` is the session's app;
3. commands only from the allowlist (launchApp, tapOn, doubleTapOn, longPressOn, inputText,
   eraseText, hideKeyboard, pressKey, back, swipe, scroll, scrollUntilVisible, assertVisible,
   assertNotVisible, extendedWaitUntil, waitForAnimationToEnd, takeScreenshot, runFlow, retry,
   repeat, setPermissions, openLink, copyTextFrom, pasteText), nested runFlow/retry/repeat
   included; those need inline `commands` (no `file`); openLink never to an app store
   (`market:`, Play Store, App Store); launchApp/setPermissions never on another app;
4. at least one `takeScreenshot`;
5. each required input value is embedded as a quoted literal `inputText` value, preserving
   leading zeros; comments, names, tags and external `${PIN}` references do not satisfy this;
6. the last command other than `takeScreenshot` is `assertVisible` or `extendedWaitUntil` with
   `visible`.

Input values remain masked as `${NAME}` in device-tool outputs, replay reports and ordinary
reply prose. The saved YAML and the accepted-action code block are intentionally shown verbatim
with the requested literal test PIN, so the action runs without external parameters.

## Slots

Slot contracts live in `models.SLOTS`. One for now:

| Slot | Category | Precondition | Outcome | Parameters |
| --- | --- | --- | --- | --- |
| `onboarding` | onboarding | app data cleared | wallet home visible, PIN set | `PIN` (required) |

## Safety notes

- Device tools are marked inherently safe, so they do not wait for human approval. This includes
  `reset_app` and `submit_action`, which **wipe the app under test** (`pm clear`): use a dedicated
  test phone only. The selected build binds the app and slot; the model cannot retarget another
  app. Persisting a candidate requires the separate human-only `save_wallet_action` handler.
- The model can tap anything inside the app under test, and `back`/`press` can leave it; the
  system UI of the phone is reachable that way. Only `open` is scoped to the app.
- The PIN is in the user's message, workflow history, model context, saved YAML and accepted-action
  code block. Only device-tool outputs, replay reports and ordinary reply prose are masked.
  Use test PINs only; do not publish actions containing a personal PIN.
- Maestro `takeScreenshot` files go to a temporary directory that is deleted after the replay;
  Maestro's own debug artifacts stay in `~/.local/state/maestro/tests/`.

## Limits

- One device per worker; one session at a time per device (no device lease or lock: two chats
  on the same worker would fight over the phone).
- Exploration uses a stateless MCP provider: each agent-device call starts `agent-device mcp`
  (state lives in the agent-device daemon, so sessions survive between calls), which costs about
  a second per call.
- The agent-device tool schemas add about 35 kB to every model request; small local models with
  a short context (Ollama defaults) may lose the instructions. qwen3:14b on Ollama reached the
  phone in the smoke test but is not expected to produce an accepted action.
- The token budget is checked after each model call, so a turn may end slightly above it.
- The session (conversation, attempts, accepted action) lives in the workflow; there is no
  continue-as-new, hence the 20-message cap.
- Runner selection, device reservation and pipeline-compatible final cleanup remain phase 2.
  The last successful replay leaves the local wallet onboarded; this is not a clean-device handoff.
  Embedded Credimi chat remains phase 3.

## Phase 0c

- A native Anthropic model and prompt caching are not implemented. Add the provider in
  `worker.model_provider`; the compatibility-mode configuration above is for evaluation only.
- Check that prompt caching works through the harness model activity (the instructions plus the
  agent-device tool schemas are a large, stable prefix) and measure the cost of one accepted
  onboarding action.
