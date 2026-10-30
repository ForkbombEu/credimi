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
runs. The key and the bearer token are never stored in browser storage, workflow inputs, or the
URL.

Credimi derives the actor and organization from authentication; the browser cannot supply
either. Direct creation through the harness API is rejected, and **New session** in the harness
returns to the start page. Keep the UI on loopback: it is not a hosted multi-user
authentication layer.

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
