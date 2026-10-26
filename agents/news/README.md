<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# News agent

Agent-driven News on Credimi's own Temporal. It runs alongside the current News pipeline
(changedetection.io + n8n, see `credimi-extra/news/`), which it does not touch.

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
CLI / news.run_scan ─update-with-start─▶ NewsAgent workflow (Python, ns default, tq news-agent)
                             ├─ model calls ............ Temporal activities (OpenAI or scripted)
                             ├─ list_changes, get_pull_request, compare_refs, fetch_document
                             │                           Python activities (GitHub REST, allowlisted fetch)
                             ├─ search_news ............ Go activity "Search news"
                             └─ after the turn ......... Go activity "Store a news draft"
                                                         (tq NewsToolsTaskQueue, in `credimi serve`)

Schedule news-daily-scan ──▶ NewsDailyScan workflow (ns default, tq news-agent)
                             └─ news.run_scan activity: one NewsAgent scan of the days since the
                                last successful run (cursor = last completion result)
```

## Safeguards

| Risk | Guard | Where |
|---|---|---|
| Unintended publication | No tool can publish. The workflow stores accepted drafts after the turn; the Go activity always writes `published=false`. | `workflow.py`, `pkg/workflowengine/activities/news.go` |
| Drafts from chat without consent | `save_draft` is the only tool that is not inherently safe, so the approval policy pauses it until you approve it in the UI. Approval shows the whole draft. **Always allow** turns the gate off for the rest of that session. | `tools.py` |
| Unsupported claims | Every key point cites evidence ids; ids must be in the ledger of what activities actually retrieved, and at least one must have been opened (`get_pull_request`, `compare_refs`, `fetch_document`), not only listed; quotes must be verbatim; `diff` must be the URL of opened evidence. Rejections go back to the model. | `validation.py`, `tools.py` |
| Wrong scan reports | Decisions must name retrieved items, `draft` decisions must match an accepted draft, and `list_changes` must have succeeded. A report with problems gets one repair turn: the model sees the problems and can submit the missing draft or change its decisions. Problems left after it are listed under **Report problems** and fail the scan (`invalid report: …`), so the daily cursor does not advance; accepted drafts are stored all the same. | `check_scan_report` in `validation.py`, `scan` in `workflow.py` |
| Duplicates | `diff` (the exact change URL) is the key: drafts with a diff already in `news` are refused, the Go store returns the existing record instead of writing, and a partial unique index on `news.diff` (non-empty values) turns a lost race into the same duplicate result. The model also sees `search_news`. | `tools.py`, `news.go`, `pb_migrations/1792000000_unique_news_diff.js` |
| Injected HTML | The model writes Markdown; raw HTML is escaped. Sources are rendered from the ledger. The webapp sanitizes again (DOMPurify). | `render.py` |
| Fetching anything | `fetch_document` only reaches `allowed_domains`, over https, without following redirects. `list_changes` only covers `sources.toml`. | `activities.py`, `tools.py` |
| Wide or future windows | A scan with a reversed window, a window over 92 days or one ending after today fails before any model call; `list_changes` applies the same rules, and a scan cannot leave its own window. | `validation.py`, `workflow.py` |
| Runaway cost | Scan: `max_turns` (default 30), `max_drafts` (default 3) and a token budget (`max_tokens`, default 400 000; the scan stops with `TokenBudgetExceeded` and keeps accepted drafts). Chat: 15 model calls per answer, 3 drafts per session; usage in every reply. | `ScanRequest`, `models.py`, `TokenBudget` in `workflow.py` |
| Unbounded chat history | A chat session takes at most 20 messages, then asks for a new session (the harness does not continue-as-new). | `models.py` |

## Run it

`make dev` starts the worker (`NEWS`) and the harness UI (`NEWSUI`, `http://localhost:8000`)
next to the API and the webapp, through `scripts/news-agent-dev.sh`. The worker reads its model
settings from `agents/news/.env`; copy `agents/news/.env.example` and uncomment one option.
Without a configured model, or with `CREDIMI_TEMPORAL_WORKERS_DISABLED` set, the worker idles and
logs why; the rest of the stack is unaffected.

The make targets below run the worker or a scan on their own (with `make dev` running, since the
Go activities are registered by `credimi serve`); they also read `agents/news/.env`. Stop the
`NEWS` process first if you start a second worker.

```sh
# No API key, no network: scripted model + recorded GitHub fixtures (openid4vp, 2026-10-01..14)
make news-agent.worker FAKE=1
make news-agent.scan FAKE=1

# Real model and live GitHub (GITHUB_TOKEN optional, raises the GitHub rate limit)
OPENAI_API_KEY=... make news-agent.worker
make news-agent.scan SINCE=2026-10-01 UNTIL=2026-10-14 [SOURCE=openid4vp] [MODEL=gpt-5.1]
```

The scan prints each tool call, then the report (decisions, stored drafts, rejected
submissions, report problems, usage) and a link to the run in the Temporal UI (`default`
namespace). Drafts are in the `news` collection of the PocketBase admin
(`http://localhost:8090/_/`); set `published` to show one on `/news`.

### Daily scan

`POST /api/news-agent/schedule` (internal admin key only, `Credimi-Api-Key`) creates or updates
the Temporal Schedule `news-daily-scan` in the `default` namespace; `DELETE` removes it.

```sh
curl -X POST localhost:8090/api/news-agent/schedule -H "Credimi-Api-Key: $CREDIMI_INTERNAL_ADMIN_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"hour_utc":6,"source_ids":["openid4vp"],"max_drafts":3,"trigger_now":true}'
curl -X DELETE localhost:8090/api/news-agent/schedule -H "Credimi-Api-Key: $CREDIMI_INTERNAL_ADMIN_KEY"
```

All fields are optional: `hour_utc` defaults to 6, `source_ids` to every source, `max_drafts` to 3
(1 to 10). An update keeps a paused schedule paused. Overlapping runs are skipped; missed runs are
caught up within 12 hours.

Each run starts `NewsDailyScan`, which scans from the day after the previous successful run to
yesterday (UTC), at most 92 days, as the NewsAgent session `news-scan-<since>-<until>`. A failed
scan fails the run and does not advance the cursor, so the next run scans those days again.

### Docker

`docker compose --profile news-agent up news_agent` runs the worker image built from
`agents/news/Dockerfile`, connected to the `temporal` service. It takes `OPENAI_API_KEY` (or
`NEWS_AGENT_MODEL_BASE_URL`, `NEWS_AGENT_MODEL_API_KEY`, `NEWS_AGENT_MODEL`) and `GITHUB_TOKEN`
from the Compose environment. Without the profile the service does not start. The harness UI is
not deployed.

### Harness web UI

`make dev` serves the harness session manager and its web UI on `http://localhost:8000`
(`NEWS_AGENT_UI_PORT` in worktrees); `make news-agent.ui` starts them alone. Open the session
manager and choose
**New session → Credimi news agent**. The chat box sends `ask`; pick `scan` in the handler menu
next to it to get the scan form. The state flow shows every tool call with its input and result,
and an approval card for each `save_draft`. The UI lists only sessions it created or that are
still running: scans started by `make news-agent.scan` are closed when they end, so look those up
in the Temporal UI. UI sessions stay open until you stop them.

In fake mode the scripted model only follows a fixed chat script: name a source and two dates
(`openid4vp 2026-10-01 2026-10-14`) and add `draft` to reach the approval step.

### Other OpenAI-compatible models (Gemini, Ollama)

The worker can use any endpoint that implements OpenAI's Chat Completions API. Set these in
`agents/news/.env` (or the worker's environment); without `NEWS_AGENT_MODEL_BASE_URL` it uses
OpenAI as before.

```sh
# Gemini free tier (key from https://aistudio.google.com/apikey, no billing)
NEWS_AGENT_MODEL_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai/
NEWS_AGENT_MODEL_API_KEY=...
NEWS_AGENT_MODEL=gemini-3.5-flash

# or a local model through Ollama (no key)
NEWS_AGENT_MODEL_BASE_URL=http://localhost:11434/v1
NEWS_AGENT_MODEL=qwen3:14b
```

`NEWS_AGENT_MODEL` replaces the model name the workflow asks for. In this mode the worker allows
10 minutes per model call and exports no traces. The adapter in `worker.py` works around
differences observed on 15/10/2026; recheck them when upgrading the harness or the SDK:

- The SDK adds GPT-5 settings (`verbosity`, `reasoning`) for the default `gpt-5.1` model name;
  Gemini rejects them, so they are dropped.
- Gemini 3 requires each tool call to come back with its thought signature. The harness loses it
  when decoding the model activity's input, so the adapter sends Gemini's placeholder signature
  instead, as the SDK's LiteLLM model does. Gemini then lacks its own reasoning context between
  steps.
- Ollama constrains the whole reply to a requested JSON schema, so a typed scan answer blocks every
  tool call. For non-Gemini endpoints the schema moves into the instructions; the SDK still
  validates the reply, and code fences around it are removed.
- Ollama does not enforce tool schemas, so small models omit fields. Invalid tool arguments come
  back to the model as the tool's reply instead of failing the turn (`workflow.py`).

Observed on 15/10/2026: `gemini-2.5-flash` is no longer offered to new keys; the Gemini free tier
allowed about 20 requests per day for `gemini-3.5-flash` (a scan takes about 6, a chat answer 2 to
4), and after that every model call is retried until the quota resets. `qwen3:8b` and `qwen3:14b`
on a 12 GB GPU answer chat questions with tools but rarely follow the drafting steps of a scan;
use them to exercise the infrastructure, not to judge drafts.

`make news-agent.test` runs lint and the tests, including an end-to-end scan on a test Temporal
server. To record new fixtures, run the worker with `NEWS_AGENT_RECORD_DIR=fixtures/github`.

## Known limits

- The harness is pre-alpha; its version is pinned. Each scan is one short session, closed by the
  CLI or the `news.run_scan` activity, because the harness does not continue-as-new.
- Duplicate detection is exact on `diff`; two different change URLs for the same story can both
  be stored. The model's `search_news` judgement is the only semantic check.
- The daily cursor is the last successful run's `until`; there is no stored per-source position.
  A retried `news.run_scan` reattaches to a running session; once that session has closed, a new
  attempt rescans the window, and the `diff` checks keep it from storing duplicates.
- `list_changes` reads at most 5 pages per GitHub list and returns at most 60 items; it reports
  `truncated` beyond that, and the model is told to split the window.
- Turns run without streaming, so the harness emits tool events but no token-level events.
- With a real key, the OpenAI Agents SDK exports traces to OpenAI by default.
