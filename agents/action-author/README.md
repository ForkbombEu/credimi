<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Action author agent

An agent that writes Maestro **wallet actions** for Credimi. From a chat message such as

> Write the onboarding action for eu.europa.ec.euidi with PIN 123456

it resolves a selected Credimi wallet version, verifies and installs its Android APK on the
worker's test phone, or its zipped simulator `.app` on a clone of the worker's iOS simulator,
explores the wallet, writes a Maestro flow, and replays it twice from a fresh install. The
accepted YAML is shown in the reply and written to `out/<appId>/<slot>.yaml` (gitignored). A
separate human confirmation saves the exact reviewed candidate into Credimi as an unpublished
`wallet_actions` record.

An OpenAI Agents SDK agent, hosted by the
[Temporal Agent Harness](https://github.com/temporal-community/temporal-agent-harness)
(`temporal-agent-harness==0.8.0`, pre-alpha). Version history and past verification runs are
in [CHANGELOG.md](CHANGELOG.md).

Credimi starts the Python `ActionAuthor` workflow on task queue `action-author` in the
selected wallet owner's organization namespace. Go activities on that namespace's
`PipelineTaskQueue` hold `core.App`: they authorize the actor, resolve the selected build
and persist approved actions. The Python worker does not receive a Credimi API key.
The harness owns the model/tool loop; device calls remain separate Temporal activities.
Devices are local to the worker host (an adb phone, or iOS simulators on a Mac); Credimi
runners are not used yet.

The UI uses its own `action-author-session-manager` workflow, so an existing news-agent
session manager cannot replace its registry. Both managers use the harness's session-manager
task queue.

The 0.8 registry reads `action-author.schema.json`, generated from the workflow by
`make action-author.schema`, instead of importing the Python class in the web server.
The UI and test targets regenerate this file automatically. The packaged 0.8 UI supplies
the session picker, structured forms, timeline and reload-resilient session rendering;
we retain the custom launcher to keep this agent's session manager separate.

The workflow preloads Pydantic's `annotated_types` metadata dependency through its
sandbox pass-through imports before updates execute. To detect late-import regressions:

```sh
cd agents/action-author
uv run pytest -q tests/test_workflow.py -W 'error:Module .* was imported after initial workflow load.:UserWarning'
```

## Setup

Requirements on the worker host: `uv`, `agent-device` (0.21.x, `npm i -g agent-device`),
`maestro` CLI (2.11 or later: older releases lack Maestro MCP `inspect_screen`), Credimi with
its organization Go workers enabled, and Temporal (`make dev`, or infrastructure through
`scripts/worktree-compose.sh up`). One device per worker:

- **Android**: `adb`, `apkanalyzer` (Android SDK command-line tools) and a dedicated phone with
  USB debugging; set `ACTION_AUTHOR_DEVICE_SERIAL` (`adb devices`).
- **iOS simulator** (macOS): Xcode with an iOS simulator runtime (`xcrun`, `plutil`, `ditto`);
  set `ACTION_AUTHOR_IOS_SIMULATOR_UDID` to a simulator that stays shut down
  (`xcrun simctl list devices`). The worker only clones it (see [iOS simulator](#ios-simulator)).
  The first agent-device call builds its XCTest runner once.

```sh
cp agents/action-author/.env.example agents/action-author/.env
# set the device (serial or simulator UDID) and one model option
```

Models, chosen in one place (`worker.provider_kind`). If several are set, the first row wins:

| Env | Model |
| --- | --- |
| `ACTION_AUTHOR_FAKE_MODEL=1` (or `FAKE=1` on make) | scripted, no key (see `fake_model.py`) |
| `ACTION_AUTHOR_CHATGPT=1` | local browser OAuth against your ChatGPT subscription; Responses API, no API-key billing |
| `ACTION_AUTHOR_ANTHROPIC=1` + `ACTION_AUTHOR_MODEL_API_KEY`, `ACTION_AUTHOR_MODEL` | native Claude Messages, automatic caching and selective context editing |
| `ACTION_AUTHOR_MODEL_BASE_URL` (+ `_API_KEY`, `ACTION_AUTHOR_MODEL`) | OpenAI-compatible Chat Completions: Ollama, freellmapi, Gemini; Anthropic's official host selects the native provider |
| `OPENAI_API_KEY` | OpenAI (`gpt-5.1`) |

For Claude, use a workspace-scoped API key with the native Anthropic Messages provider:

```dotenv
ACTION_AUTHOR_CHATGPT=0
ACTION_AUTHOR_FAKE_MODEL=0
ACTION_AUTHOR_ANTHROPIC=1
ACTION_AUTHOR_MODEL_API_KEY=your-workspace-scoped-key
ACTION_AUTHOR_MODEL=claude-sonnet-5-5
```

This uses Claude API billing, not the ChatGPT subscription. Non-workspace-scoped keys requiring
an extra workspace header are not exposed by the current configuration.
`ACTION_AUTHOR_MODEL_BASE_URL=https://api.anthropic.com/v1/` selects native Messages, not
Chat Completions.

Restart the worker after changing providers and start a fresh session when changing model IDs.
Claude's signed thinking blocks are preserved across Temporal model activities. New sessions
keep system instructions stable and append changing authoring state to the new user message;
do not switch providers back and forth within a conversation containing signed thinking.

For the ChatGPT subscription, run `make action-author.login` once in a local terminal and
approve ChatGPT plan usage for `credimi-action-author`. This uses OpenAI's documented OAuth
flow for local/open-source apps, **not** Codex CLI or its cached credentials. The credential
file is owner-only at `~/.config/credimi-action-author/chatgpt.json`; override with
`ACTION_AUTHOR_CHATGPT_CREDENTIALS` in `.env` (read by both the login and worker targets).
The worker refreshes tokens without putting them in
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

If a message reaches its token budget or model-call limit, completed exploration and tool
results are retained. Send `Continue` in the same session to resume from the current device
state with fresh per-message budgets. The model response that crossed the limit is not
executed, and no action is saved automatically. Keep the same device and installed build; do
not clear the app between messages. Continuation still incurs model charges and is not
context compaction or caching.

Agent-device MCP activities have one attempt and a three-minute total deadline, including queue
wait. Device interactions are not automatically retried: a lost response must not duplicate a
tap or text entry. After updating the worker, stop old authoring sessions and create a fresh
session; already scheduled activities keep their original inputs and retry policy.

## Run

Use a dedicated test phone: installation replaces the selected package, and authoring/replay
clears its data. Stop older authoring workers and sessions before testing this version.
In Credimi, create or select a wallet version with an installer for your device (an Android
APK, or a zipped iOS simulator `.app`) in an organization you own; copy its canonical
`organization/wallet/version` identifier. Set `TEMPORAL_NAMESPACE` to that organization's
canonical name in `agents/action-author/.env`, alongside the device and one model option.

Start `make dev` in a separate terminal if the API, UI and Temporal are not already running.
Do not use `make dev.noworkers`: authoring needs the organization Go activities.

```sh
# Both processes must use the selected wallet owner's organization namespace.
TEMPORAL_NAMESPACE=your-org make action-author.worker
TEMPORAL_NAMESPACE=your-org make action-author.ui
make action-author.schema            # regenerate the schema consumed by agents.toml
make action-author.test              # ruff + pytest, no physical device required
```

Open `http://localhost:8001/` (`ACTION_AUTHOR_UI_PORT` changes the port; the UI binds to loopback
only). The harness opens its authenticated start page. Enter the
Credimi URL (normally `http://localhost:8090`) and a **Credimi user API key** from your
dashboard, then select the platform of the worker's device and a wallet version. This is
separate from the Claude API key configured on the worker. The page exchanges the user key for
a bearer token and lists the versions with an installer for that platform in organizations
where you are an owner (superusers see all versions). A session started for the other platform
than the worker's device stops before installing anything and says which platform to choose.
Select **Start authoring session**. Setup stays open until that exact workflow is listed as
running and answers both the status and interface queries. It waits up to 60 seconds; if the
worker is unavailable or the namespace differs, **Retry opening session** retries the already
created workflow instead of creating another. The bearer token is cleared after creation.

The API key is cleared after version lookup and is never stored in browser storage, workflow
inputs, or the URL. The authenticated caller must be an owner of the wallet's organization
(or a superuser). The start endpoint still derives the actor and organization from Credimi
authentication; the browser cannot supply either. The request carries `platform` (`android`
or `ios`), which selects the installer. The response is `202` with `workflow_id`,
`run_id`, `workflow_namespace` and the canonical `wallet_version_identifier`. Optional API
field `app_id` rejects a selected build with a different package or bundle ID; it does not
select an installer. Set `ACTION_AUTHOR_CREDIMI_URL` to the trusted Credimi origin used by the
worker when it is not `http://localhost:8090`.

Select **Existing sessions** to return to the packaged session picker. Both **New session**
and **Start a new session** in the harness return to authenticated setup; they do not open
the stock caller-supplied-actor form. Direct creation through the harness API remains rejected.
Keep the UI on loopback: it is not a hosted multi-user authentication layer.

The selected installer is streamed through `GET /api/action-author/installer` using a
short-lived, single-build `Credimi-Artifact-Grant` header. Downloads do not follow redirects or
use an environment proxy; the worker checks the installer's SHA256 and the package/version
metadata. On Android it also checks the installed APK's SHA256, and later replays reject an
installed APK that has changed.

### iOS simulator

`simctl` cannot clear one app's data, and uninstalling keeps keychain items, so the worker
isolates runs the way pipelines do, with simulator clones (`ios.IOSSimulatorDevice`):

1. the selected zip is extracted with `ditto` and must hold exactly one `.app` (top level or
   under `Payload/`) whose `CFBundleSupportedPlatforms` includes `iPhoneSimulator`;
2. the base simulator is cloned into a session template, the `.app` is installed there and the
   template is shut down. The base simulator is never booted or modified;
3. every `reset_app` and every replay deletes the current clone and boots a new clone of the
   template: data, keychain and permissions start as freshly installed.

Clones are named `credimi-action-author-<base UDID>-…`. The worker deletes them when it stops,
and deletes leftovers of a killed worker when it starts. A reset takes about 10–15 s; the first
Maestro command on each clone adds about 20 s while Maestro sets up its driver.

The installed bundle cannot be hashed back to the uploaded zip: the worker binds the template to
the bundle ID and digest it installed and checks that the clone still has that bundle.
agent-device has no iOS device allowlist, so pinning relies on the wrapper overwriting `udid`.
Maestro's `back` does nothing on iOS; the instructions tell the model to tap the app's own back
button instead.

After two successful replays, the chat shows the reviewed candidate and asks whether to save
it. Choose **Accept** or **Refuse**; do not select a handler or copy a hash into a form.
The name defaults to **Onboarding**. To choose another name, include it in your request,
for example `Write action "First setup" for onboarding` or
`Write the onboarding action and store it as "Demo onboarding"`.

**Accept** saves that exact candidate as an unpublished action, then closes the session after
delivering the receipt. **Refuse** saves nothing and leaves the chat open for revisions, with
the submission budget restored.
The workflow supplies the selected wallet/build, name, category, candidate hash and replay
evidence; only your boolean decision comes from the browser. Automatic tool-approval settings
cannot approve this confirmation. A stale, already answered or malformed decision cannot save.
Reloading the page restores the pending confirmation. If saving fails, the chat offers a fresh
**Accept**/**Refuse** decision for the same candidate without rerunning exploration or replays.

This flow creates a new action; it never selects an existing action to overwrite by name.
The backend's explicit replacement contract still requires the existing record ID and current
code SHA256. The legacy save handler is retained for recorded workflow replay and hidden from
the chat's handler selector; it cannot bypass the callback for new authoring turns.

Verify the saved record in the selected wallet's actions: code must match the accepted YAML,
category must be `onboarding`, and a newly created record must remain unpublished. Ordinary
PocketBase reads hide provenance; the confirmation reply identifies the validated build
and candidate hash. A successful save closes the workflow after draining the final reply;
failed saves leave the session open for correction. To discard a session without saving,
open **Sessions**, hover its row, select **End session**, and confirm. This is a graceful
close: in-flight work drains rather than being terminated. Close the local UI/worker when done.

Saved provenance records the wallet version, platform, installer SHA256, package or bundle ID,
YAML hash, workflow/run and two successful replays. It is hidden from ordinary PocketBase
responses and non-superuser writes;
PocketBase silently ignores attempts to write hidden fields. Editing the action's code
through the native API clears provenance that no longer matches. Credimi exposes only whether
provenance is present, as the computed field `__agent_authored__`; the wallet dashboard shows a
robot icon on those actions.

Closing does not delete Temporal history or an already saved action. Recent closed authoring
runs remain in the session picker (up to 200 closed runs, alongside upstream running-session
discovery), and the UI serves recorded events through read-only workflow queries because the
upstream live stream uses updates, which cannot run on completed workflows. Browser transcript
caching remains upstream behavior; Temporal history follows the namespace's retention policy.
The adapter reuses the pinned harness's SSE encoding helpers and customizes its callback card;
re-check stream delivery, closed-session history, callback selectors and new-session buttons
on harness upgrades.

For deterministic tests, `FAKE=1` selects the scripted model. Without a configured device,
the device is also fake (Android); it does not demonstrate real installation or replay.

## Tools

| Tool | What it does |
| --- | --- |
| `set_target(app_id, slot, parameters)` | sets the app, the slot and the parameter values (session state, hidden from the model) |
| agent-device: `open snapshot press click fill type scroll swipe back wait find get is screenshot close` | exploration over MCP (`agent-device mcp`) |
| `reset_app()` | closes the agent-device session and resets the app to a fresh install: `pm clear` on Android, a new simulator clone on iOS |
| `maestro_cheat_sheet()` | Maestro's syntax reference (Maestro MCP `cheat_sheet`) |
| `maestro_inspect_screen()` | closes the agent-device session, then Maestro MCP `inspect_screen`, compacted to one line per targetable node (text, resource-id or accessibility identifier, content-desc or accessibility label, hint or clickable) with bounds, at most 8 000 characters |
| `check_action(yaml)` | validation only (below): OK or the list of problems |
| `submit_action(yaml)` | validation, then two replays; both must pass. At most 3 per target |

The agent-device MCP server is wrapped on the worker (`device.AgentDeviceServer`):

- only the 15 tools above are listed; any other tool name is refused;
- device selection is pinned by the worker: `serial` (Android) or `udid` (iOS, the current
  clone), `platform`, `session` (and the other device/tenant fields) are removed from the
  schemas the model sees and overwritten on every call; on Android the process also runs with
  `AGENT_DEVICE_ANDROID_DEVICE_ALLOWLIST=<serial>`;
- `open` only opens the app under test (no URLs, no Metro hints);
- text outputs are capped at 20 000 characters.

The shared MCP context is refreshed by each successful `set_target`, including changes within
one chat turn. Device operations are refused until a target has been selected.
Structured UI `target` values (ref, selector or point) pass through unchanged; only string
device-form targets (`mobile`, `tv`, `desktop`) are removed to preserve device pinning.

Device turn-taking: Maestro's Android driver does not start while an agent-device session holds
the device, so every Maestro step first runs `agent-device close --session action-author`
(this ends the session; the app stays in the foreground). Only the structured `SESSION_NOT_FOUND`
error is tolerated; other close failures stop the Maestro handoff. CLI processes run in their
own process group, which is killed and reaped on timeout or cancellation. On iOS both tools can
drive the simulator at once; the same order is kept.

### Replay (`submit_action`)

For each of 2 runs: `agent-device close`, reset the app (`adb -s <serial> shell pm clear
<appId>`, or a new simulator clone on iOS), write the YAML to a temporary file,
`maestro --device <serial or clone UDID> test <file>` (timeout 5 minutes).
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

The saved YAML and the accepted-action code block show the requested literal test PIN, so the
action runs without external parameters.

## Slots

Slot contracts live in `models.SLOTS`. One for now:

| Slot | Category | Precondition | Outcome | Parameters |
| --- | --- | --- | --- | --- |
| `onboarding` | onboarding | fresh install (data cleared) | wallet home visible, PIN set | `PIN` (required) |

## Safety notes

- Device tools are marked inherently safe, so they do not wait for human approval. This includes
  `reset_app` and `submit_action`, which **wipe the app under test** (`pm clear`, or a new
  simulator clone): use a dedicated test phone only. The selected build binds the app and slot;
  the model cannot retarget another app. Persisting a candidate requires a separate human
  Accept/Refuse callback, never a model tool.
- The model can tap anything inside the app under test, and `back`/`press` can leave it; the
  system UI of the phone is reachable that way. Only `open` is scoped to the app.
- The PIN is stored and shown in clear: in the user's message, workflow history, model context,
  tool outputs, replay reports, replies, saved YAML and accepted-action code block. This is
  intended for test PINs, which are not secrets.
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
- Runner devices, device reservation and automatic cleanup are not implemented. The last
  successful replay leaves the local wallet onboarded; this is not a clean-device handoff.
- The chat runs in the local harness UI, not inside Credimi.

## Native Claude caching and context

Native Messages enables automatic prompt caching (`cache_control`) with a five-minute TTL.
`ACTION_AUTHOR_ANTHROPIC_CACHE_TTL=1h` selects an hour; `off` disables caching. Cache writes cost
more than ordinary input, while cache reads cost less; see
[Anthropic's pricing](https://platform.claude.com/docs/en/about-claude/pricing).
Token budgets count all input, including cache reads and writes, plus output. Caching reduces
cost/latency, not the token budget itself. Replies and worker logs report cache read/write tokens
separately.

Selective server-side context editing uses Anthropic's `context-management-2025-06-27` beta.
Above `ACTION_AUTHOR_ANTHROPIC_CONTEXT_TOKENS` (default 30,000 input tokens per request), it can
clear old `snapshot`, `screenshot`, `maestro_inspect_screen`, `find`, `get` and `is` results.
It retains the most recent three tool uses, never clears tool inputs, and excludes interaction
results, parameters, generated YAML, validation/replay reports and the Maestro cheat sheet.
The full conversation remains in Temporal, and the UI still shows the original evidence.
Set the threshold to `0` to disable editing.

Editing is selective, not a hard context limit: protected evidence may still grow. Clearing
invalidates some cached prefixes, so each edit must remove at least 5,000 input tokens;
replies/logs report cleared tokens. No local rewriting of signed history or model-generated
summarization is used. Native requests are text/function-tool based and non-streaming at the
model boundary; the harness still streams tool progress and final replies to its UI.

See [prompt caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching) and
[context editing](https://platform.claude.com/docs/en/build-with-claude/context-editing).
