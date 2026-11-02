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
platform. Starting a session queues it for the device; the page shows the queue position and
the device preparation, offers **Cancel** while waiting, and opens the chat once the session
runs. The key is never stored; the bearer token is kept in the tab's `sessionStorage`
(`credimi-action-author`, with the Credimi URL) so the chat's save review can call Credimi, and
it never reaches workflow inputs or the URL.

Credimi derives the actor and organization from authentication; the browser cannot supply
either. Direct creation through the harness API is rejected, and **New session** in the harness
returns to the start page. Keep the UI on loopback: it is not a hosted multi-user
authentication layer.

## Action spec

Before starting, the page describes the action to author: the **slot** (onboarding, get a
credential, get a credential with issuer-specific steps, present a credential, other), the
expected **outcome** (success or a wallet rejection; not for onboarding), whether the flow
runs in **Chrome** (Android devices only), the **prerequisites** (saved actions of the
selected wallet from `GET /api/action-author/wallet-actions`, in order, each with a source per
parameter it reads), and the **parameters** of the new action. A source is a Credimi
credential offer, a use-case verification, an HTTP request (JSON: `request`, `value_path`,
optional `captures` and, on the new action's parameters, `status`), or a fixed value. Credimi
validates the spec at start (`400` with its message); a used-up daily quota answers `429`.
Values are stored in Temporal history: use test data only.

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
