<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# News agent (experimental, local only)

A proof of concept of agent-driven News on Credimi's own Temporal. It runs alongside the current
News pipeline (changedetection.io + n8n, see `credimi-extra/news/`), which it does not touch.

An OpenAI Agents SDK agent, hosted by the
[Temporal Agent Harness](https://github.com/temporal-community/temporal-agent-harness)
(`temporal-agent-harness==0.6.0`, pre-alpha), researches the sources in `sources.toml`: it decides
what to read and what is newsworthy, and drafts articles. Drafts land in the `news` collection
with `published=false`; a human publishes them in the PocketBase admin.

It accepts two messages:

- `scan`: unattended review of a time window (CLI or UI form). Drafts are submitted with
  `submit_draft` and stored after the turn.
- `ask`: chat in the harness UI ("what changed in OpenID4VP between 2026-10-01 and
  2026-10-14?"). The conversation and its evidence persist across messages. A draft is checked
  with `check_draft`, then saved with `save_draft`, which waits until you approve it.

```
CLI ──update-with-start──▶ NewsAgent workflow (Python, ns default, tq news-agent)
                             ├─ model calls ............ Temporal activities (OpenAI or scripted)
                             ├─ list_changes, get_pull_request, compare_refs, fetch_document
                             │                           Python activities (GitHub REST, allowlisted fetch)
                             ├─ search_news ............ Go activity "Search news"
                             └─ after the turn ......... Go activity "Store a news draft"
                                                         (tq NewsToolsTaskQueue, in `credimi serve`)
```

## Safeguards

| Risk | Guard | Where |
|---|---|---|
| Unintended publication | No tool can publish. The workflow stores accepted drafts after the turn; the Go activity always writes `published=false`. | `workflow.py`, `pkg/workflowengine/activities/news.go` |
| Drafts from chat without consent | `save_draft` is the only tool that is not inherently safe, so the approval policy pauses it until you approve it in the UI. Approval shows the whole draft. **Always allow** turns the gate off for the rest of that session. | `tools.py` |
| Unsupported claims | Every key point cites evidence ids; ids must be in the ledger of what activities actually retrieved; quotes must be verbatim; `diff` must be a retrieved URL. Rejections go back to the model. | `validation.py`, `tools.py` |
| Duplicates | `diff` (the exact change URL) is the key: drafts with a diff already in `news` are refused, the Go store returns the existing record instead of writing. The model also sees `search_news`. | `tools.py`, `news.go` |
| Injected HTML | The model writes Markdown; raw HTML is escaped. Sources are rendered from the ledger. The webapp sanitizes again (DOMPurify). | `render.py` |
| Fetching anything | `fetch_document` only reaches `allowed_domains`, over https, without following redirects. `list_changes` only covers `sources.toml`. | `activities.py`, `tools.py` |
| Wide or future windows | `list_changes` refuses windows over 92 days or ending after today; a scan cannot leave its own window. | `validation.py` |
| Runaway cost | Scan: `max_turns` (default 30) and `max_drafts` (default 3). Chat: 15 model calls per answer, 3 drafts per session; usage in every reply. | `ScanRequest`, `models.py` |
| Unbounded chat history | A chat session takes at most 20 messages, then asks for a new session (the harness does not continue-as-new). | `models.py` |

## Run it

With `make dev` running (the Go activities are registered by `credimi serve`):

```sh
# No API key, no network: scripted model + recorded GitHub fixtures (openid4vp, 2026-10-01..14)
make news-agent.worker FAKE=1
make news-agent.scan FAKE=1

# Real model and live GitHub (GITHUB_TOKEN optional, raises the GitHub rate limit)
OPENAI_API_KEY=... make news-agent.worker
make news-agent.scan SINCE=2026-10-01 UNTIL=2026-10-14 [SOURCE=openid4vp] [MODEL=gpt-5.1]
```

The scan prints each tool call, then the report (decisions, stored drafts, rejected
submissions, usage) and a link to the run in the Temporal UI (`default` namespace). Drafts are in
the `news` collection of the PocketBase admin (`http://localhost:8090/_/`); set `published` to
show one on `/news`.

### Harness web UI

`make news-agent.ui` starts the harness session manager and its web UI on
`http://localhost:8000` (the worker still runs separately). Open the session manager and choose
**New session → Credimi news agent**. The chat box sends `ask`; pick `scan` in the handler menu
next to it to get the scan form. The state flow shows every tool call with its input and result,
and an approval card for each `save_draft`. The UI lists only sessions it created or that are
still running: scans started by `make news-agent.scan` are closed when they end, so look those up
in the Temporal UI. UI sessions stay open until you stop them.

In fake mode the scripted model only follows a fixed chat script: name a source and two dates
(`openid4vp 2026-10-01 2026-10-14`) and add `draft` to reach the approval step.

`make news-agent.test` runs lint and the tests, including an end-to-end scan on a test Temporal
server. To record new fixtures, run the worker with `NEWS_AGENT_RECORD_DIR=fixtures/github`.

## Known limits

- The harness is pre-alpha; its version is pinned. Each scan is one short session, closed by the
  CLI, because the harness does not continue-as-new.
- Duplicate detection is exact on `diff`; two different change URLs for the same story can both
  be stored. The model's `search_news` judgement is the only semantic check.
- The window is the cursor: there is no stored per-source position.
- Turns run without streaming, so the harness emits tool events but no token-level events.
- With a real key, the OpenAI Agents SDK exports traces to OpenAI by default.
