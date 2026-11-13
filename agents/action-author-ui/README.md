<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Action author chat UI

The [Temporal Agent Harness](https://pypi.org/project/temporal-agent-harness/) chat UI that
Credimi embeds in its action author session pages (`/my/action-author/<session id>`). Credimi
starts each session from a wallet card on `/my/wallets`, on a device of a runner of your
organization or of another organization's published runner that offers the agent; the agent
itself runs on that runner (`credimi-runner/agent`). This service shows only the harness chat,
the session flow, the timeline and the other harness panes. The session page itself shows the
save review, the idle warning and the Android live view.

## Run

Docker Compose runs the service as `action_author_ui` (built from this directory, listening on
port 8080, no published port in production). Credimi reaches it at `ADDRESS_ACTION_AUTHOR_UI`
(default `http://localhost:8282`; the worktree dev stack publishes it on
`127.0.0.1:${ACTION_AUTHOR_UI_PORT}`, default 8282).

```sh
make action-author.ui-test   # lint and tests; the Temporal-backed test needs the temporal CLI
```

Environment:

- `TEMPORAL_ADDRESS`, and the other `TEMPORAL_*` settings of the Temporal client config. No
  namespace is configured: each request names its own.
- `ACTION_AUTHOR_UI_PORT`: the listening port (default 8080).

## Credimi's proxy

Browsers never reach the service directly. The session page frames
`/action-author-ui/<session id>/?s=<chat id>&temporal-ui=…` on Credimi, and Credimi's proxy
(`pkg/internal/actionauthorui`) is the security boundary: it authenticates the user from the
browser session, checks that Credimi started the session for the caller's organization, allows
only the harness routes of that session's chat (`<session id>-author`), refuses cross-site state
changes, strips the Credimi credentials, and forwards the request without the
`/action-author-ui/<session id>` prefix with two headers:

- `X-Credimi-Namespace`: the caller's organization namespace;
- `X-Credimi-Action-Author-Chat`: the chat workflow ID.

The service answers `400 missing_scope` to any request without both headers (or with a value
that is not an organization name or a chat ID). It keeps one harness app, with its own Temporal
client, per namespace, created on first use; a failed connection answers `503
temporal_unavailable` and is retried on the next request.

## Routes

The harness routes run unchanged, on the request's namespace, except:

- `GET /` serves the packaged chat page with `./action-embed.css` (relative, since the page sits
  under Credimi's sub-path), which hides the harness **New session** and **Start a new session**
  buttons, its sessions drawer and its `confirm_wallet_action_save` callback card: Credimi starts
  and lists sessions and shows the save review on its session page.
- `GET /api/agents` lists the agent on the chat's own task queue (`<runner_id>-ActionAuthor`)
  with whether a worker polls it.
- `GET /api/sessions` lists only the scoped chat: running, closed, or none before it starts.
- `GET /api/attach` and `POST /api/chat` finish a stream that the chat's end cut short with its
  last events, read through the runner agent's `authoring_events` query, since the upstream
  stream reader needs a running workflow. Credimi shows a closed chat's transcript itself (from
  its Temporal history) and never embeds the UI for it.
- `POST /api/sessions` answers `403 authenticated_start_required` and every
  `/api/callback-result` request `403 review_through_credimi`: Credimi signs save approvals
  through `POST /api/action-author/reviews`, so the harness route cannot settle a review.

The service never starts agent workflows: Credimi starts each chat on
`<runner_id>-ActionAuthor`. Temporal service errors (for example `RESOURCE_EXHAUSTED`) answer
`503 temporal_unavailable` with `Retry-After: 2` and one warning log line.

Every response sends `X-Frame-Options: SAMEORIGIN`, and HTML pages a `Content-Security-Policy`
with `frame-ancestors 'self'` that runs only scripts of the page's own origin (the packaged chat
page has no inline script): only Credimi's pages can frame the UI.

## Large payloads

The service reads Temporal payloads with the default harness converter and no offload storage:
it does not run on the runner host, so it cannot read payloads the runner agent offloads to its
private directory. Stream events stay below the offload threshold.
