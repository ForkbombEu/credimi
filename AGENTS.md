<!--
SPDX-FileCopyrightText: 2025 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2025 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# AGENTS.md

## Architecture (LLM-first, current branch)

### What runs (dev)

- Temporal dev server: `temporal server start-dev --db-filename pb_data/temporal.db` (Procfile: `BE`, gRPC at `localhost:7233`, UI port per Temporal defaults).
- PocketBase API: `go run main.go serve` (Procfile: `API`, default `localhost:8090`, data in `pb_data/`).
- Webapp (SvelteKit/Vite): `cd webapp && bun dev` (Procfile: `UI`, default `localhost:5100`).
- Reverse proxy: PocketBase proxies `/{path...}` to `ADDRESS_UI` (default `http://localhost:5100`) in `pkg/routes/routes.go`.

### Persistence

- PocketBase SQLite: `pb_data/`.
- Temporal dev DB: `pb_data/temporal.db`.

### Key env vars

- `TEMPORAL_ADDRESS` (Temporal host:port).
- `ADDRESS_UI` (UI reverse proxy target).
- `MOBILE_RUNNER_SEMAPHORE_DISABLED`.
- `MOBILE_RUNNER_SEMAPHORE_WAIT_TIMEOUT`.

### Tenancy + Temporal namespaces

- `organizations.canonified_name` is the Temporal namespace for that tenant.
- Org create/update ensures the namespace exists and starts workers (`pkg/internal/pb/namespaces.go`).
- On server start, workers start for `default` and all org namespaces (`pkg/workflowengine/hooks/hook.go`).
- Mobile-runner semaphore workflows run in the Temporal `default` namespace (`pkg/workflowengine/mobile_runner_semaphore_constants.go`).

## Pipelines (Dynamic Pipeline Workflow)

### Pipeline input contract

- Schema: `schemas/pipeline/pipeline_schema.json`.
- Core types: `pkg/workflowengine/pipeline/types.go`.
- `step.with` shape (YAML + JSON parity in `pkg/workflowengine/pipeline/parser.go`):
  - `config` is reserved for per-step config.
  - `payload` is reserved for step payload.
  - Any other keys under `with` are merged into `payload`.
- Mobile runner selection invariants: each `mobile-automation` step must specify `with.payload.runner_id`, or the pipeline must set `runtime.global_runner_id` (`pkg/workflowengine/pipeline/mobile_automation_hooks.go`).

### Run pipeline (no mobile automation)

- UI calls `POST /api/pipeline/start` with `{ pipeline_identifier, yaml }`.
- Handler: `pkg/internal/apis/handlers/pipeline_handler.go` resolves canonified pipeline path and starts the Dynamic Pipeline Workflow on `PipelineTaskQueue` in the org namespace.
- The handler creates a `pipeline_results` record with `(owner, pipeline, workflow_id, run_id)` for tracking.

### Run pipeline (mobile automation → queue + semaphore)

- UI entrypoint: `webapp/src/lib/pipeline/utils.ts` chooses `/api/pipeline/queue` when any `mobile-automation` step exists.
- Queue endpoints (auth required; `pkg/internal/apis/handlers/pipeline_queue_handler.go`):
  - `POST /api/pipeline/queue` body `{ pipeline_identifier, yaml }`
  - `GET /api/pipeline/queue/{ticket}?runner_ids=...`
  - `DELETE /api/pipeline/queue/{ticket}?runner_ids=...`
  - `runner_ids` accepts `runner_ids=a,b,c` or repeated params.
- Queue response (webapp uses a subset):
  - `ticket_id`, `runner_ids`, `required_runner_ids`, `leader_runner_id`
  - `status` in `{queued|starting|running|failed|canceled|not_found}`
  - `position` is 0-based; UI displays `position + 1`
  - optional `workflow_id`, `run_id`, `workflow_namespace`, `error_message`
- Temporal semaphore (namespace `default`):
  - Workflow per runner ID: `mobile-runner-semaphore/<runner_id>` (`pkg/workflowengine/mobilerunnersemaphore/types.go`)
  - Updates: `EnqueueRun`, `CancelRun`, `RunDone`; queries: `GetRunStatus`, `GetState`
  - Implementation: `pkg/workflowengine/workflows/mobile_runner_semaphore.go`
- Start after grant:
  - Semaphore runs `StartQueuedPipelineActivity` (`pkg/workflowengine/activities/queued_pipeline.go`) which starts the pipeline workflow in the owner org namespace.
  - Injected config keys: `mobile_runner_semaphore_ticket_id`, `mobile_runner_semaphore_runner_ids`, `mobile_runner_semaphore_leader_runner_id`, `mobile_runner_semaphore_owner_namespace`.
  - The pipeline workflow reports completion to the leader semaphore via `ReportMobileRunnerSemaphoreDoneActivity` (`pkg/workflowengine/pipeline/semaphore_done.go`).
  - `pipeline_results` creation is best-effort after Temporal start and retried; the internal handler is idempotent on `(workflow_id, run_id)`.
  - PB uniqueness constraints: `(owner, workflow_id, run_id)` in `pb_migrations/1765364510_created_pipeline_results.js`.

## Mobile runners (PocketBase + internal lookup)

- PB collection: `mobile_runners` (migration `pb_migrations/1769505309_created_mobile_runners.js`).
- Internal API (no auth):
  - `GET /api/mobile-runner?runner_identifier=<canonified>` → `{ runner_url, serial }` (`pkg/internal/apis/handlers/mobile_runners_handlers.go`).
  - `GET /api/mobile-runner/semaphore?runner_identifier=...` → summarized semaphore state (`pkg/internal/apis/handlers/mobile_runners_handlers.go`).

### External runner HTTP contracts (runner_url)

- `POST {runner_url}/fetch-apk-and-action`
  - Body: `{ instance_url, version_identifier, action_identifier }`
- `POST {runner_url}/store-pipeline-result`
  - Body: `{ video_path, last_frame_path, logcat_path, run_identifier, runner_identifier, instance_url }`
  - Response: `{ result_urls: string[], screenshot_urls: string[] }`
- Implemented in the external runner service (from `github.com/forkbombeu/credimi-extra`).

### Temporal runner worker contract

- Task queue: `${runner_id}-TaskQueue` (set in `pkg/workflowengine/pipeline/mobile_automation_hooks.go`).
- Note: `workflows.MobileAutomationTaskQueue` exists (`pkg/workflowengine/workflows/mobile.go`) but pipeline execution uses the dynamic `${runner_id}-TaskQueue`.
- Namespace: the `mobile-automation` child workflow runs in the same Temporal namespace as the pipeline (org namespace), so the runner worker must poll `${runner_id}-TaskQueue` in that namespace.
- Must register:
  - Workflow `mobile-automation` (denylisted from the pipeline worker; see `pkg/workflowengine/registry/registry.go`).
  - Activities in `pkg/workflowengine/activities/mobileflow.go` (these call `github.com/forkbombeu/credimi-extra/mobile`; `avdctl` is used inside `credimi-extra`).

## Routes + DTOs

- Route groups: `pkg/internal/apis/RoutesRegistry.go` wires handler groups.
- Route abstraction: `pkg/internal/routing/routing.go` (`RouteGroup`, `RouteDefinition`, validation binding).
- Validation middleware: `pkg/internal/middlewares/validation.go` stores typed input in context; handlers use `routing.GetValidatedInput[T](e)`.
- Error middleware: `pkg/internal/middlewares/errors.go`.

## Errors (CRE codes)

- CRE code source of truth: `pkg/internal/errorcodes/errorcodes.go`.
- Activities return Temporal `ApplicationError` via `workflowengine.BaseActivity` helpers (`pkg/workflowengine/activity.go`), typically embedding a CRE code.
- Workflows wrap app errors with `workflowengine.NewWorkflowError` (adds metadata like Temporal UI link).
- API endpoints sometimes parse workflow errors via `workflowengine.ParseWorkflowError`.
- HTTP error response shapes (both exist):
  - Direct `apierror.APIError` JSON (common in handlers): `{status, error, reason, message}`.
  - Middleware-wrapped errors (when errors bubble through `ErrorHandlingMiddleware`): `{ apiVersion:"2.0", error:{ code, message, errors:[{domain, reason, message}] } }`.
  - Follow the existing endpoint contract you are modifying; error shapes are mixed today.

## Build / Test

- Source of truth: `Makefile` (targets) + `Procfile.dev` (processes/ports).
- Dev: `make dev` (hivemind runs `Procfile.dev`).
- Go unit (default): `make test`; focused: `go test -tags=unit ./pkg/... -run TestName`; watch: `make test.p TestName`.
- Lint/format/generate: `make lint`, `make fmt`, `make generate`.
- API only: `go run ./main.go serve`.

## Test Suites

- Go unit (default): `make test` or `go test -tags=unit ./...` (deterministic; no external services).
- Go integration (opt-in): `go test ./...` without `-tags=unit` to include `//go:build !unit` tests; requires external services (e.g., Temporal) if/when enabled. Currently deferred; no CI job runs these yet.
- Webapp unit: `cd webapp && bun run test:unit -- --run` (fast; pure module tests preferred).
- Webapp E2E (opt-in): `cd webapp && bun run test:e2e` (requires a running backend + deterministic fixtures).

## Webapp

- Install: `cd webapp && bun install` (bun is default JS runtime).
- Dev: `cd webapp && bun run dev` (after `bun run predev`).
- Lint/typecheck: `cd webapp && bun run lint`; `cd webapp && bun run check` (`--watch` to iterate).
- Tests: `cd webapp && bun run test:unit -- -t "spec name"`; `cd webapp && bun run test:e2e -- tests/path.spec.ts`.

## Docs

- `cd docs && bun i && bun run docs:dev --host` serves documentation.

## Code Style

- Go formatting via gofumpt/gofmt/gci/golines; tabs per .editorconfig; import order stdlib/third-party/internal.
- Wrap errors with `fmt.Errorf("context: %w", err)` and prefer `CredimiError` for domain surfaces.
- Tests use `stretchr/testify` with table-driven cases; rely on `require`/`assert` helpers.
- Favor dependency injection over globals; keep constructors returning interfaces.
- Svelte/TS code: Prettier tabs + single quotes + width 100; ESLint perfectionist sorts imports.
- Use TypeScript-first patterns, prefer type aliases for unions, rely on Tailwind (sorted by plugin).
- Reuse `effect`/`zod` utilities for async flows/validation; no Cursor/Copilot rule overrides present.

## Private Dependencies

- `github.com/forkbombeu/credimi-extra` is a **private module**.
- It must **never** be removed from `go.mod` or `go.sum`, even if:
    - `go mod tidy` marks it as unused.
    - CI cannot resolve it.
    - static analysis or lint tooling flags it.
- Any changes touching Go module files MUST preserve this entry unless explicitly instructed by a human maintainer.

## Test Conventions

- Go: table-driven `TestX_Y` names, `require` for hard failures, avoid IO in unit tests.
- Webapp: prefer pure-module Vitest tests; for E2E use stable selectors (`data-testid`) and avoid time-based waits.
- Fixtures: use `fixtures/test_pb_data` (or `test_pb_data` where already established) and keep fixtures read-only.
