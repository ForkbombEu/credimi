<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Action author chat UI

A local [Temporal Agent Harness](https://pypi.org/project/temporal-agent-harness/) chat UI for
action author sessions. Credimi starts each session on a device of an admin-managed runner;
the agent itself runs on that runner (`credimi-runner/agent`). This UI only shows the chat, the
review of the accepted YAML, and the **Accept** / **Refuse** card.

## Run

```sh
make action-author.ui        # http://localhost:8001 (ACTION_AUTHOR_UI_PORT changes the port)
make action-author.ui-test   # lint and tests; the Temporal-backed test needs the temporal CLI
```

The UI needs only `TEMPORAL_ADDRESS` (the make target uses the dev stack's Temporal) and, at its
start page, a Credimi URL and a **Credimi user API key** from your dashboard. No namespace or
task queue is configured.

Open `http://localhost:8001/` or `http://127.0.0.1:8001/` (the UI binds to loopback only). It
answers only requests whose `Host` is one of those two addresses, refuses state-changing
requests and API calls from any other `Origin`, and sends no CORS headers, so other web pages
and DNS-rebinding hosts cannot drive it. The start page exchanges the key for a bearer token
and lists the devices of admin-managed runners of organizations where you are an owner
(superusers see all), and the wallet versions with an installer for the selected device's
platform, plus the wallets the session installs from the Play Store or App Store (listed as
"installed from the store"; the device needs the store with a signed-in account). Starting a
session queues it for the device; the page shows the queue position and
the device preparation, offers **Cancel** while waiting, and opens the chat once the session
runs. The key is never stored; the bearer token is kept in the tab's `sessionStorage`
(`credimi-action-author`, with the Credimi URL) so the chat's plan and save reviews can call
Credimi, and it never reaches workflow inputs or the URL.

Credimi derives the actor and organization from authentication; the browser cannot supply
either. Direct creation through the harness API is rejected, and **New session** in the harness
returns to the start page. Keep the UI on loopback: it is not a hosted multi-user
authentication layer.

No page can be framed. Every HTML page but the start page (which has its own policy) sends a
`Content-Security-Policy` that runs only scripts of the UI's own origin: the harness chat page
has no inline script, and the scripts the UI adds to it (`/action-new-session.js`,
`/action-save-review.js`) are files. Styles may be inline; requests may go to the UI and, for
the plan and save reviews, to Credimi over HTTP(S).

## Action plan

The start page asks only for the device and the wallet version. Describe the action in the
chat: its category, success or rejection, input values (PIN) and references (a Credimi URL or
identifier, or a deeplink). The agent proposes a **plan**: the slot, the outcome, the
prerequisites (saved actions of the wallet) and the parameter sources.
The plan card (`confirm_action_plan`) shows each summary line in its own text-only element
(`white-space: pre-wrap`, so indented detail lines keep their indentation) and the input values; **Accept** /
**Refuse** sends `POST /api/action-author/plans` with the bound namespace, the chat's workflow
ID, the card's `tool_id`, its `plan` and the stored bearer token. Credimi validates the plan and
signs it; a `400` shows Credimi's message and keeps the card, so you can still refuse it. A
session can save several actions in a row, each with its own plan. An organization has no daily
token allowance until a Credimi superuser sets one, and a missing or used-up allowance answers
`429` with Credimi's message. Values are stored in Temporal history: use test data only.

## Save review

The **Accept** / **Refuse** card shows the accepted YAML (`tool_input.code`) and the plan the
session's plan token signs (`Plan fixed when the session started.` when the token is empty),
and enables **Accept** only when the YAML's SHA-256 matches the review's `candidate_sha256`;
otherwise it says `The reviewed YAML does not match its hash. Refuse this candidate.` and
leaves only **Refuse**. A non-empty plan token that does not decode shows
`Unreadable plan token: refuse this candidate.` and also leaves only **Refuse**.
It sends the decision to Credimi, not to the workflow:
`POST /api/action-author/reviews` with the bound namespace, the chat's workflow ID, the
review's `tool_id`, `candidate_sha256` and `plan_token`, and the stored bearer token. Credimi
checks that you started the session, signs the approval the save activity verifies, and
settles the review. Without a stored token, or after a `401`, both cards ask for a Credimi user
API key (and the Credimi URL when unknown) and retry; a `409` means the card is no longer
pending and it reloads. The UI answers every `/api/callback-result` request with `403
review_through_credimi`, so the harness route cannot settle a plan or a review without Credimi.

## Namespace binding

Sessions run in the Temporal namespace of the device's organization. The start page binds the
UI to it through `POST /api/action-author/namespace`: with the session's `workflow_namespace`
when a session starts, and with the selected device's `namespace` (from
`GET /api/action-author/devices`) on **Existing sessions**. A binding connects to that
namespace, runs the harness session-manager worker in this process, and ensures the
`action-author-session-manager` workflow there.

The UI holds one binding at a time (one user, one organization): starting a session on a
device of another organization switches it, and the previous organization's sessions leave the
list until it is bound again. Until the first binding, API calls answer `409
namespace_not_selected` and chat links redirect to the start page. The UI never starts agent
workflows: Credimi starts each chat on `<runner_id>-ActionAuthor`, so the registry's task queue
is a placeholder.

## Runner agent

A device is selectable only when Credimi reports it `is_online` and `agent_available` (its
runner's action author polled `<runner_id>-ActionAuthor` in the last two minutes); others are
labelled `· offline` or `· agent not enabled`, and a start Credimi refuses with `503
agent_unavailable` shows Credimi's message. Once the session runs, the page asks
`GET /api/action-author/chat-ready/<workflow_id>` (`{ready, running, execution_status}`;
`ready` once the chat's history holds more than its start and first scheduled task) and sends
the chat's status and interface queries only when it is ready, since Temporal buffers queries
to a workflow no worker has run. If the started chat is not ready within 60 s, the page says
the runner's action author is not answering and keeps **Cancel** available while it polls on.
Temporal service errors (for example `RESOURCE_EXHAUSTED`) answer `503 temporal_unavailable`
with `Retry-After: 2` and one warning log line.
