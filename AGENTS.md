<!--
SPDX-FileCopyrightText: 2025 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2025 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# AGENTS.md

This file is the source of truth for agents working in the Credimi repository.

Credimi is not a generic Go/Svelte app. It is a Temporal-backed conformance platform for decentralized identity systems, with PocketBase as the application database/API shell, a SvelteKit webapp, mobile-runner orchestration, and several related Forkbomb ecosystem libraries.

If an instruction in this file conflicts with a user prompt, local code, generated output, or tool suggestion:

-> STOP
-> surface the conflict
-> ask the user how to proceed

Do not invent conventions. Do not silently normalize ambiguity. If a decision is not defined here and is not directly evident from existing code, ask.

## Enforcement Model

All rules in this file are mandatory unless explicitly marked as optional.

A task is incomplete until the agent has:

- understood the relevant architecture boundary
- checked the affected code path
- made the smallest safe change
- preserved user and dependency work
- run the relevant validation, or explained why it could not run
- reported any residual risk

Agents MUST NOT:

- assume a pattern just because it appears once
- replace project-specific contracts with generic framework defaults
- remove private or related Forkbomb dependencies because a tool suggests it
- commit, push, or publish unless explicitly asked
- modify generated files without understanding their generator

## Puria Doctrine

This repository follows the operating doctrine from `puria/md` where it applies to an existing Credimi codebase.

Prime directive:

- Do not behave like a generic coding agent.
- Prefer clarity over cleverness.
- Prefer explicitness over magic.
- Prefer small steps over huge rewrites.
- Prefer working code over speculative architecture.
- Prefer tests over promises.
- Prefer boring, maintainable solutions over fashionable abstractions.
- Prefer deterministic and reproducible engineering.

Execution doctrine:

- Do not infer conventions.
- Do not adopt undocumented patterns.
- If a convention is observed but not documented here, record it in `.agents/HITL.md` and ask before relying on it.
- Make the smallest safe change.
- Validate the change before finishing.
- If validation fails, fix the issue and rerun validation before claiming completion.
- If a rule is violated, the task is incomplete until the agent corrects it.

Planning doctrine:

- Large tasks must be broken into phases.
- Stop and propose phases when a request is too broad to execute safely in one pass.
- Do not change direction mid-task without user agreement.
- Do not turn speculative architecture into implementation without evidence from the codebase.

Repository doctrine:

- Credimi is an existing project with `Makefile`, `Procfile.dev`, Go modules, and a SvelteKit app. Do not add `mise.toml` or `Taskfile.yml` merely because new-project PURIA defaults mention them.
- If creating a new standalone project inside or beside this repository, ask whether PURIA new-project skeleton rules should apply before adding `mise.toml`, `Taskfile.yml`, initial commits, or bootstrap files.
- Required tooling must be declared in the project-owned tool entrypoint where this repository already declares tools: `Makefile`, `webapp/package.json`, `docs/package.json`, Go tool directives, or explicit docs. Do not hide required tools in prose only.
- Do not leave binaries, logs, temp folders, coverage artifacts, local databases, or generated caches in the repository.

HITL doctrine:

- `.agents/HITL.md` is the human-in-the-loop decision backlog.
- Append to `.agents/HITL.md` when a convention, architectural rule, dependency contract, validation rule, or design rule is missing or ambiguous.
- Do not use a newly observed convention as precedent until the user confirms it or it is documented here.
- HITL entries must include date, question, context, options considered, default risk, and owner/status.
- Do not hide blockers in final summaries only; record durable decisions or open questions in `.agents/HITL.md`.

## Boot Sequence

Before modifying files:

1. Read this `AGENTS.md`.
2. Inspect the repository structure relevant to the request.
3. Check `git status --short` and preserve unrelated user changes.
4. For unfamiliar behavior, use GitNexus first when it applies.
5. Read the local code before editing it.
6. If a convention is unclear, append it to `.agents/HITL.md` and ask the user. Do not assume.
7. Choose the smallest change that satisfies the task.
8. Validate with the narrowest meaningful test first, then broader checks when risk warrants it.

For UI/Svelte tasks, also read `webapp/AGENTS.md` before editing webapp files.

## GitNexus Requirement

This project is indexed by GitNexus as `credimi`.

Use GitNexus for code intelligence whenever a task changes or depends on symbols, flows, or architecture.

Before editing any function, class, method, exported type, workflow, activity, handler, route registration, Svelte component logic, or shared module:

- run `gitnexus_impact({target: "symbolName", direction: "upstream"})`
- report the blast radius to the user: direct callers, affected process groups, and risk level
- if the risk is HIGH or CRITICAL, warn the user before editing

Before committing, if the user explicitly requested a commit:

- run `gitnexus_detect_changes()`
- verify the affected symbols and execution flows are expected

If GitNexus warns that the index is stale, run:

```sh
npx gitnexus analyze
```

Useful resources:

- `gitnexus://repo/credimi/context`
- `gitnexus://repo/credimi/clusters`
- `gitnexus://repo/credimi/processes`
- `gitnexus://repo/credimi/process/{name}`

Skill references:

- `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md`
- `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md`
- `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md`
- `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md`
- `.claude/skills/gitnexus/gitnexus-guide/SKILL.md`
- `.claude/skills/gitnexus/gitnexus-cli/SKILL.md`

## Forkbomb Ecosystem Boundary

Treat Credimi and its related Forkbomb organization dependencies as one product boundary unless the user says otherwise.

Important related dependencies:

- `github.com/forkbombeu/credimi-extra`: private Go module used for mobile automation and runner-facing workflows.
- `github.com/forkbombeu/credimi-conformance-assessment`: conformance report/domain assessment code.
- `github.com/forkbombeu/eudi-conformance-evidence`: evidence generation and parsing for EUDI conformance artifacts.
- `github.com/ForkbombEu/et-tu-cesr`: CESR tooling and binary integration.
- `github.com/forkbombeu/avdctl`: Android virtual device support used through the mobile automation stack.
- `github.com/ForkbombEu/stepci-captured-runner`: external binary downloaded into `.bin/` by `make tools`.
- `@forkbombeu/temporal-ui`: Svelte Temporal workflow UI package.
- `github.com/forkbombeu/credimi_plugins`: ecosystem plugins/resources referenced by the product.

Rules:

- Do not remove these dependencies from `go.mod`, `go.sum`, `webapp/package.json`, or lockfiles unless a human maintainer explicitly asks.
- Do not replace Forkbomb modules with public substitutes just because they are private or hard to resolve in CI.
- When changing contracts consumed by these libraries or services, inspect the local call sites and, when necessary, ask whether the sibling repository must be updated too.
- If a dependency cannot be fetched because of private access, report that as an environment/access issue, not as evidence that the dependency is unused.

## Repository Shape

Main areas:

- `main.go`, `cmd/`: PocketBase app entrypoints and CLI commands.
- `pkg/internal/apis`: HTTP API handlers and route groups.
- `pkg/internal/routing`: route abstraction, validation binding, and typed input retrieval.
- `pkg/internal/middlewares`: auth, validation, and error middleware.
- `pkg/internal/pb`: PocketBase hooks and collection behavior.
- `pkg/workflowengine`: Temporal clients, workflows, activities, pipeline engine, registry, and hooks.
- `pkg/workflowengine/pipeline`: dynamic pipeline parser/execution helpers and pipeline-specific hooks.
- `pkg/workflowengine/workflows`: Temporal workflow implementations.
- `pkg/workflowengine/activities`: Temporal activity implementations.
- `pb_migrations`: PocketBase schema migrations.
- `schemas`: pipeline and domain schemas.
- `config_templates`: generated and maintained conformance templates.
- `webapp`: SvelteKit/Vite frontend.
- `docs`: documentation site.
- `fixtures`: test fixtures; keep read-only unless a task explicitly changes fixtures.

## Dev Runtime

Source of truth:

- `Makefile`
- `Procfile.dev`
- `docker-compose.yaml`

Local dev process:

- `make dev` starts infrastructure and runs the API/UI through `hivemind`.
- Temporal is provided by Docker Compose in dev, with gRPC at `localhost:7233`.
- PocketBase API runs at `localhost:8090`.
- Webapp runs at `localhost:5100`.
- PocketBase proxies `/{path...}` to `ADDRESS_UI` in `pkg/routes/routes.go`.

Procfile dev processes:

- `API`: waits for Temporal at `localhost:7233`, then runs `go tool gow run -tags=credimi_extra main.go serve`.
- `UI`: waits for PocketBase at `localhost:8090`, then runs `cd webapp && bun i && bun dev`.

Persistence:

- PocketBase SQLite data lives in `pb_data/`.
- Dev Temporal state also uses local project data/infrastructure and must be treated as disposable dev state.

Key environment variables:

- `TEMPORAL_ADDRESS`: Temporal host and port.
- `ADDRESS_UI`: UI reverse proxy target.
- `MOBILE_RUNNER_SEMAPHORE_DISABLED`: disables the mobile-runner semaphore path when configured.
- `MOBILE_RUNNER_SEMAPHORE_WAIT_TIMEOUT`: mobile-runner queue wait timeout.
- `CREDIMI_INTERNAL_ADMIN_KEY`: plaintext runtime key for trusted internal HTTP activities and internal result posting.
- `CREDIMI_EXTRA_PAT`: optional Docker build token for private `credimi-extra` access.

Do not commit local `pb_data/`, `.env`, generated local databases, secrets, coverage files, binaries, or downloaded `.bin/` tools.

## Tenancy And Temporal

- `organizations.canonified_name` is the Temporal namespace for that tenant.
- Organization create/update ensures the namespace exists and starts workers in `pkg/internal/pb/namespaces.go`.
- Server startup starts workers for `default` and all organization namespaces in `pkg/workflowengine/hooks/hook.go`.
- Mobile-runner semaphore workflows run in the Temporal `default` namespace.
- Pipeline workflows run in the owner organization namespace.
- The `mobile-automation` child workflow runs in the same namespace as the pipeline and uses a runner-specific task queue.

Namespace changes are high risk. Use GitNexus and inspect startup hooks, route handlers, workflow starts, and worker registration before editing.

## Dynamic Pipeline Workflow

Pipeline contract:

- Schema: `schemas/pipeline/pipeline_schema.json`.
- Core types: `pkg/workflowengine/pipeline/types.go`.
- Parser: `pkg/workflowengine/pipeline/parser.go`.
- `step.with.config` is reserved for per-step config.
- `step.with.payload` is reserved for step payload.
- Other keys under `step.with` are merged into `payload`.
- `mobile-automation` steps must provide `with.payload.runner_id`, or the pipeline must set `runtime.global_runner_id`.

Direct run path:

- UI calls `POST /api/pipeline/start` with `{ pipeline_identifier, yaml }`.
- Handler: `pkg/internal/apis/handlers/pipeline_handler.go`.
- The handler resolves the canonified pipeline path and starts the Dynamic Pipeline Workflow on `PipelineTaskQueue` in the organization namespace.
- The handler creates a `pipeline_results` record with `(owner, pipeline, workflow_id, run_id)`.

Queued mobile run path:

- UI logic in `webapp/src/lib/pipeline/utils.ts` chooses `/api/pipeline/queue` when the YAML contains a `mobile-automation` step.
- Queue handler: `pkg/internal/apis/handlers/pipeline_queue_handler.go`.
- Queue endpoints require user auth:
  - `POST /api/pipeline/queue` with `{ pipeline_identifier, yaml }`
  - `GET /api/pipeline/queue/{ticket}?runner_ids=...`
  - `DELETE /api/pipeline/queue/{ticket}?runner_ids=...`
- `runner_ids` accepts both `runner_ids=a,b,c` and repeated params.
- Queue statuses are `queued`, `starting`, `running`, `failed`, `canceled`, and `not_found`.
- `position` is 0-based; the UI displays `position + 1`.

Semaphore:

- Namespace: `default`.
- Workflow ID: `mobile-runner-semaphore/<runner_id>`.
- Types: `pkg/workflowengine/mobilerunnersemaphore/types.go`.
- Implementation: `pkg/workflowengine/workflows/mobile_runner_semaphore.go`.
- Updates: `EnqueueRun`, `CancelRun`, `RunDone`.
- Queries: `GetRunStatus`, `GetState`.

Grant/start path:

- Semaphore runs `StartQueuedPipelineActivity` in `pkg/workflowengine/activities/queued_pipeline.go`.
- The activity starts the pipeline workflow in the owner organization namespace.
- Injected config keys:
  - `mobile_runner_semaphore_ticket_id`
  - `mobile_runner_semaphore_runner_ids`
  - `mobile_runner_semaphore_leader_runner_id`
  - `mobile_runner_semaphore_owner_namespace`
- The pipeline reports completion to the leader semaphore through `ReportMobileRunnerSemaphoreDoneActivity` in `pkg/workflowengine/pipeline/semaphore_done.go`.
- `pipeline_results` creation after Temporal start is best-effort and retried.
- The internal result handler is idempotent on `(workflow_id, run_id)`.
- PocketBase uniqueness constraint: `(owner, workflow_id, run_id)` in `pb_migrations/1765364510_created_pipeline_results.js`.

## CI Wallet APK Runs

Endpoint:

- `POST /api/pipeline/run-wallet-apk`

Contract:

- Auth accepts user auth or API key.
- Request must include `pipeline_identifier`.
- `metadata.sha` is required and must be a string.
- Exactly one APK source is allowed: multipart `apk_file` or HTTP(S) `apk_url`.
- The pipeline YAML must reference exactly one wallet through `mobile-automation` `version_id` values.

Behavior:

- The handler creates a caller-owned temporary `wallet_versions` record named by the canonified commit SHA.
- It rewrites only the run YAML to use the temporary wallet version.
- It enqueues through the existing mobile-runner semaphore path.
- It does not mutate the stored pipeline record.
- Runtime cleanup is driven by server workflow input config `temp_wallet_version = { record_id, identifier, owner_id, cleanup: true }`.
- `PipelineWorkflow.Start` ignores YAML-provided `config.temp_wallet_version`; that key is reserved for internal server input.
- Cleanup calls internal `DELETE /api/wallet/temp-version/{record_id}` once per run with expected owner/identifier metadata.
- The internal route rejects deletes that do not match the wallet version record.
- Canceling a not-yet-running wallet APK ticket deletes the temp `wallet_versions` record after semaphore cancellation succeeds.

Known risk:

- There is no durable `temporary` or `expires_at` field on `wallet_versions`; failures after enqueue but before workflow cleanup can leave temporary records.

Do not redesign this cleanup model without asking. It is a known V1 tradeoff.

## Pipeline Visibility

- Pipeline workflows set the `PipelineIdentifier` search attribute on direct, queued, and scheduled starts.
- `PipelineIdentifier` is the canonified owner/pipeline path.
- `GET /api/list-workflows` lists only non-pipeline workflow trees.
- `GET /api/pipeline/list-executions` keeps the grouped latest-executions-by-pipeline view.
- `GET /api/pipeline/list-executions/{id}` lists executions for one pipeline.
- `GET /api/pipeline/list-results` no longer exists.
- Pipeline execution listing prefers Temporal `ListWorkflows` with `PipelineIdentifier` filtering.
- Missing search attributes fall back to `pipeline_results` mapping.

Search attribute changes affect Temporal visibility, UI execution pages, and fallback behavior. Treat them as cross-cutting.

## Mobile Runners

PocketBase collection:

- `mobile_runners`
- Migration: `pb_migrations/1769505309_created_mobile_runners.js`.

Public list shape:

- `GET /api/mobile-runners`
- Items include `path`, `is_owned`, `is_published`, `is_online`.
- Selector views omit `url`, `type`, `devices`, and `queue_length`.
- Pipeline YAML still uses `runner_id` and `global_runner_id`.

Internal lookup:

- `GET /api/mobile-runner?runner_identifier=<canonified>` returns `{ runner_url, serial }`.
- `GET /api/mobile-runner/semaphore?runner_identifier=...` returns summarized semaphore state.
- Handler: `pkg/internal/apis/handlers/mobile_runners_handlers.go`.

External runner HTTP contract:

- `POST {runner_url}/fetch-apk-and-action`
  - Body: `{ instance_url, version_identifier, action_identifier }`
- `POST {runner_url}/store-pipeline-result`
  - Body: `{ video_path, last_frame_path, logcat_path, run_identifier, runner_identifier, instance_url }`
  - Response: `{ result_urls: string[], screenshot_urls: string[] }`

The external runner service is implemented in `github.com/forkbombeu/credimi-extra`. If the contract changes, ask whether the sibling repository must change.

Temporal runner worker contract:

- Task queue: `${runner_id}-TaskQueue`.
- The dynamic task queue is set in `pkg/workflowengine/pipeline/mobile_automation_hooks.go`.
- `workflows.MobileAutomationTaskQueue` exists in `pkg/workflowengine/workflows/mobile.go`, but pipeline execution uses the dynamic task queue.
- Runner workers must poll `${runner_id}-TaskQueue` in the organization namespace.
- Runner workers must register workflow `mobile-automation`.
- `mobile-automation` is denylisted from the pipeline worker in `pkg/workflowengine/registry/registry.go`.
- Runner workers must register activities in `pkg/workflowengine/activities/mobileflow.go`.

## Routes, DTOs, Auth, Errors

Route wiring:

- `pkg/internal/apis/RoutesRegistry.go` wires handler groups.
- `pkg/internal/routing/routing.go` defines `RouteGroup`, `RouteDefinition`, validation binding, and typed input helpers.
- Validation middleware stores typed input in context.
- Handlers use `routing.GetValidatedInput[T](e)`.

Auth:

- Route groups with `AuthenticationRequired=true` accept either `Authorization` token or `X-Api-Key`.
- Temporal-internal routes enforce `X-Api-Key` via `RequireInternalAdminAPIKey`.
- Internal admin key surfaces are sensitive. Do not log keys or include them in test snapshots.

Errors:

- CRE code source of truth: `pkg/internal/errorcodes/errorcodes.go`.
- Activities return Temporal `ApplicationError` through `workflowengine.BaseActivity` helpers, usually embedding a CRE code.
- Workflows wrap app errors with `workflowengine.NewWorkflowError`, adding metadata such as Temporal UI links.
- API endpoints sometimes parse workflow errors through `workflowengine.ParseWorkflowError`.

HTTP error response shapes are mixed:

- Direct `apierror.APIError`: `{status, error, reason, message}`.
- Middleware-wrapped errors: `{ apiVersion:"2.0", error:{ code, message, errors:[{domain, reason, message}] } }`.

Follow the endpoint contract you are modifying. Do not normalize all error shapes opportunistically.

## Build, Test, Validate

Validation is mandatory. Run the checks that correspond to the files touched, mirroring `.github/workflows`.

Go changes are any changes to `**/*.go`, `go.mod`, `go.sum`, `.mise.toml`, Go generators, Go schemas/templates consumed by Go, or workflow/activity/handler behavior.

Required for Go changes:

- Format: `make fmt`.
- Module hygiene: `go mod tidy -diff`.
- Module verification: `go mod verify`.
- Vet: `go vet ./...`.
- Vulnerability check: `govulncheck -C . -format text ./...`.
- Lint: `golangci-lint run ./...`.
- Revive: run the repository's revive configuration or the same revive action used in CI when available.
- Tests: `make test`.

`make lint` is the preferred local aggregate for module hygiene, vet, govulncheck, and golangci-lint because it is the repository-owned entrypoint. Still run or account for `revive` when preparing a PR, because CI runs `morphy/revive-action:v2`.

Focused Go iteration is allowed before the final required checks:

- Focused unit: `go test -tags=unit ./pkg/... -run TestName`.
- Watch focused unit: `make test.p TestName`.
- Coverage gate when coverage is relevant: `make coverage-check`.
- Full integration-style Go run when explicitly needed: `go test ./...`.

Webapp changes are any changes under `webapp/**`.

Required for webapp changes:

- Install/update deps when needed: `cd webapp && bun install`.
- Format or format-check according to scope: `cd webapp && bun run format` or `cd webapp && bun run lint`.
- Lint: `cd webapp && bun run lint`.
- Typecheck: `cd webapp && bun run check`.
- Unit tests: `cd webapp && bun run test:unit -- --run`.
- Build for route/load/layout changes or dependency changes: `cd webapp && bun run build`.

The `.github/workflows/webapp.yml` workflow is currently commented out, but agents must still run local webapp validation for webapp changes.

Docs changes are any changes under `docs/**`.

Required for docs changes:

- Generate OpenAPI if docs depend on generated API state: `go generate ./pkg/gen.go`.
- Build docs: `cd docs && bun install && bun run docs:build`.
- Check internal `credimi.io` links using the same HTTP/1.1 logic as `.github/workflows/docs.credimi.io.yaml`.
- Check external docs links with Lychee using `docs/.lychee.toml` when `lychee` is available.
- Local preview when useful: `cd docs && bun run docs:dev --host`.

PR metadata:

- PR titles must pass `.github/workflows/pr-conventional-title.yml`.
- Use a Conventional Commit-style PR title, for example `feat(pipeline): add queued wallet apk cleanup guard`.

Release/version files:

- `VERSION` changes trigger release automation.
- Do not edit `VERSION`, tags, or release metadata unless the user explicitly asks for release/version work.
- Release automation uses semantic-release and GoReleaser; preserve Conventional Commit semantics so release calculation remains deterministic.

General validation policy:

- Run the narrowest relevant test while iterating.
- Before finishing, run the full required checks for every touched area unless blocked.
- If validation cannot run because of private modules, missing `CREDIMI_EXTRA_PAT`, network, missing services, missing tools, or sandbox limits, report the exact blocker and what remains unverified.
- Do not claim a task is complete if required validation is skipped without explanation.

## Code Style

Go:

- Use `gofmt`/repo formatters; `make fmt` is the repo entrypoint.
- Keep import order consistent with existing files: standard library, third-party, internal.
- Wrap errors with `fmt.Errorf("context: %w", err)`.
- Prefer `CredimiError` for domain surfaces where existing code does.
- Use dependency injection over new globals.
- Keep constructors returning interfaces when that is the local pattern.
- Tests use `stretchr/testify` with table-driven cases.
- Use `require` for hard failures and `assert` for independent checks.
- Avoid IO in unit tests unless the package already tests filesystem behavior.

Svelte/TypeScript:

- Follow `webapp/AGENTS.md`.
- Use Svelte 5/SvelteKit docs when changing Svelte or SvelteKit behavior.
- Use TypeScript-first patterns.
- Prefer type aliases for unions.
- Reuse existing `effect` and `zod` utilities for async flow and validation.
- Follow Prettier tabs, single quotes, width 100.
- Let ESLint perfectionist sort imports.
- Use Tailwind and keep classes sorted by the configured plugin.
- Prefer pure-module Vitest tests.
- For E2E, use stable `data-testid` selectors and avoid time-based waits.

Generated code:

- Identify the generator before editing generated output.
- Prefer changing source schemas/templates and running `make generate`.
- Do not hand-edit generated client/types files unless the user explicitly asks for a temporary patch.

## Brand And Design

Do not create `DESIGN.md` in this repository unless a human maintainer explicitly asks.

When the `credimi-design` skill is available, use it for any Credimi visual, UI, brand, asset, slide, mock, or prototype task. Read the skill's `README.md` first for the file index, then read its `DESIGN.md` completely. Treat that skill `DESIGN.md` as the canonical brand spec. Use its `colors_and_type.css`, `assets/`, `Components Library.html`, `ui_kits/webapp/`, `preview/`, and `uploads/prototypes/` as the reference materials for production UI and throwaway artifacts.

If the `credimi-design` skill is unavailable in the current agent environment, use the rules in this section as the local fallback and report that the canonical skill files could not be read.

Credimi is the trustworthy compliance checker for decentralized identity solutions. It sits in a serious, technical, EU-regulated market, but must never feel bureaucratic, cold, or opaque.

Brand principles:

- Calm, not corporate: generous whitespace, restrained color, no exclamation marks.
- Honest about complexity: keep domain language exact and explain it inline when needed.
- Numbers earn their place: use exact stats, dates, scores, and counts.
- Evidence over claims: show status chips, scoreboards, audit trails, and verifiable facts.
- EU-formal, not EU-stiff: address users in second person; use `DD/MM/YYYY`.

Voice and content:

- Use sentence case for product UI.
- Section headers carry a trailing colon, for example `Test results:`.
- Buttons use imperatives: `Start a new test`, `Explore Marketplace`, `See pipelines`.
- `See all ↗` is the canonical link for more content elsewhere.
- Never use emoji in product UI.
- Keep these terms exact: credential issuer, verifier, wallet, conformance, interoperability, OpenID4VC, OID4VCI, OpenID4VP, EUDI, EUDIW, mDoc, SD-JWT VC, JWT, ES256, jwk, cose_key, Draft 13, Temporal, trust list, EUDI ARF.
- Percentages use `N% compliant`.
- Run counts use `N over M tests passed`.
- Dates use `DD/MM/YYYY`; date-time uses `DD MMM YYYY, HH:mm`.
- Identifiers and versions are monospace.

Color:

- Use CSS custom properties from `colors_and_type.css`; never inline a hex when a token exists.
- Deep indigo is the only dominant layout color: `--brand-primary`, `--brand-primary-700`, `--brand-primary-600`, `--brand-accent`, `--brand-accent-vivid`.
- Lavender is the page wash: `--brand-secondary`, `--brand-secondary-mid`, `--brand-secondary-strong`, `--brand-secondary-deep`.
- White is for cards, popovers, and inputs; never put content directly on a pure-white page background.
- Status chips use exact semantic tokens; do not invent statuses without asking.
- Score bands:
  - `>= 80%`: Stable, green.
  - `60-79%`: Flaky, amber.
  - `30-59%`: Failing, orange.
  - `< 30%`: Broken, red.
  - no data: grey dash.
- Gradients are forbidden except inside the provided Credimi wordmark SVG.

Typography:

- Inter is the only sans family.
- Source Code Pro is the only monospace family.
- Display and headings use Inter 700 with the existing type scale.
- Do not use Roboto, Arial, system-default sans, or Manrope.
- Eyebrows are 12px, weight 500, letter-spacing `0.14em`, uppercase, and always paired with a heading below.
- Use monospace for identifiers, run counts, versions, IDs, timestamps, and test names.
- Use sans for measures such as percentages, totals, and durations.

Layout:

- Max content width is 1280px.
- Page side padding is 24-32px on desktop and 16px on mobile.
- Topbar is sticky, white, with a 1px bottom border and no shadow.
- Footer is a full-bleed deep-indigo slab.
- Main content uses lavender wash plus white cards.
- Section headers use title + optional count + right-aligned `See all ↗`, with a 1px bottom border.
- Use the diagonal-fold motif only on page headers and marketing surfaces; never as body decoration.

Spacing, radii, elevation:

- Use the spacing scale from `colors_and_type.css`.
- Canonical radius is 6px for buttons, inputs, and cards.
- Chips/badges use 4px; alerts use 8px.
- Cards are white, hairline-bordered, 10px radius, no shadow by default.
- Credimi is almost flat. Use shadows only for popovers, dropdowns, menus, modals, toasts, or focused controls.

Icons and illustration:

- Lucide is the only icon family in product UI.
- Use 1.5px stroke, line style, currentColor, 16-20px default.
- Do not fill or tint icons except for existing status-chip conventions.
- Use existing illustrations for empty/error states; do not generate AI illustrations.
- Hero photography is rare; default to lavender page/header treatments and the diagonal-fold motif.
- Do not recolor, recreate, shadow, outline, or modify Credimi logos.

Components:

- Primary buttons: `--brand-primary` fill, `--fg-on-primary` text, 6px radius, Inter 500, one primary CTA per surface.
- Secondary buttons: `--brand-secondary-deep` fill, `--brand-primary` text.
- Outline buttons: white background, 1px border.
- Ghost buttons: transparent for dense toolbars.
- Destructive buttons: destructive fill, white text, confirm-delete only.
- Link buttons: transparent, indigo, underlined.
- Alerts are bordered, 8px radius, 60% white background, lucide icon, heading and body.
- Popovers are white, 6px radius, `--border-strong`, `--shadow-md`, 16-24px padding.
- Status chips are pill-shaped, with a colored dot and 11px label.
- Inputs are 36px tall, 10px radius, white background, 1px border, indigo focus border and ring.
- Compliance pills always pair score percentage with band label.

Motion:

- Motion is functional, not decorative.
- Hover: 150ms ease-out.
- Focus ring and popovers/dropdowns: 200ms ease-out.
- Toast: 220ms ease-out.
- No page transitions.
- No bounces, spring physics, parallax, autoplay video, scroll-jacking, mouse-tracking, or animations over 250ms in product UI.
- Loading uses indigo spinner rings or static skeletons; no shimmer.

Product patterns:

- Service/app cards use avatar, name, status chip, one-line description, optional compliance pill.
- Test result rows show name, last check, pass ratio, and score pill; dates are monospace.
- Pipeline details show flow chain, grouped step results, recent runs/specs, and root-cause failure callouts when failing or broken.
- Empty/error states use an illustration, sentence-case headline, and one CTA.

Strict don'ts:

- Do not use gradients outside the wordmark.
- Do not use emoji in product UI.
- Do not use SHOUTY CASE for titles or buttons.
- Do not use exclamation marks in product UI.
- Do not use shadows on cards by default.
- Do not use rounded corners with a left-border accent color.
- Do not use icons inside buttons unless semantically required.
- Do not use the diagonal-fold motif as body decoration.
- Do not recolor or modify the logo.

When editing Svelte components, validate with `bun run check` or a focused test when practical. For visual or interactive changes, use a browser/screenshot check when risk warrants it.

## Secrets, Data, And Safety

Never commit:

- `.env` or `.env.*`
- API keys, tokens, private keys, certificates, credentials
- local database files
- `pb_data/`
- build outputs and binaries
- logs, caches, coverage artifacts
- downloaded `.bin/` tools

## Git And Commit Rules

Do not commit substantial work directly to `main`. Use a branch and pull request unless a maintainer explicitly requests direct commits.

Branch naming:

- Use lowercase kebab-case.
- Use the format `<type>/<short-scope>-<short-description>`.
- Allowed types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `build`, `perf`, `security`, `release`.
- Include an issue number when one exists: `<type>/<issue-number>-<short-description>`.
- Keep branch names descriptive and under 80 characters when practical.
- Do not use spaces, uppercase letters, underscores, or personal names.
- Do not use vague branches such as `changes`, `updates`, `fix`, `wip`, or `agent-work`.

Examples:

- `docs/agents-puria-doctrine`
- `fix/482-wallet-apk-cancel-cleanup`
- `feat/pipeline-execution-filters`
- `ci/go-vulnerability-gate`

Agents MUST NOT push unless the user explicitly asks.

Forbidden unless explicitly requested:

```sh
git push
```

Agents MUST NOT commit unless the user explicitly asks for a commit.

When a commit is requested, the commit is invalid unless it follows this format:

```text
<type>(<scope>): <subject>

reason:
<why>

prompt:
<short intent>
```

Commit rules:

- Use Conventional Commits.
- The subject is imperative, concise, and lowercase.
- The subject explains intent, not just the diff.
- `reason` is required and is at most 3 lines.
- `prompt` is required and is at most 2 lines.
- Do not describe only changed files.
- Do not include secrets, credentials, tokens, private keys, local database paths, or private runtime data.

Valid examples:

```text
feat(pipeline): add queued wallet apk cleanup guard

reason:
Prevent temporary wallet versions from surviving canceled queued runs.

prompt:
Make queued wallet APK cancellation clean up temporary versions.
```

```text
docs(agents): define credimi design source

reason:
Ensure future agents use the canonical Credimi design skill before UI work.

prompt:
Document the credimi-design skill contract in AGENTS.md.
```

Before every requested commit:

- run the relevant formatter: usually `make fmt` for Go changes and `cd webapp && bun run format` for broad webapp formatting, or a narrower existing formatter when appropriate
- run the relevant lint/type/test validation: usually `make lint`, `make test`, `cd webapp && bun run lint`, `cd webapp && bun run check`, or focused tests based on scope
- inspect `git status --short`
- inspect staged files with `git diff --cached --stat` and, when needed, `git diff --cached`
- inspect staged files for secrets and sensitive data
- ensure private Forkbomb dependencies remain present
- run `gitnexus_detect_changes()`

If formatting, linting, tests, staged-file inspection, or GitNexus change detection fails:

-> STOP
-> fix the issue
-> rerun the failed validation
-> do not commit until corrected

If the repository has no applicable formatter or validation command for the affected area:

-> STOP
-> report the missing rule
-> ask before committing

Repository cleanliness before completion:

- Run or inspect `git status --short`.
- Leave the worktree clean if the task included committing.
- If the worktree is not clean because the task did not request a commit, explain the remaining modified files.
- Never revert unrelated user changes.

Do not run destructive commands such as `git reset --hard`, broad `rm -rf`, database purges, or migration rewrites unless the user explicitly asks or approves.

## Private Dependencies

`github.com/forkbombeu/credimi-extra` is a private module.

It must never be removed from `go.mod` or `go.sum`, even if:

- `go mod tidy` marks it as unused
- CI cannot resolve it
- static analysis flags it
- a local environment lacks private repository access

Any change touching Go module files must preserve this entry unless a human maintainer explicitly instructs otherwise.

The same caution applies to the other Forkbomb ecosystem dependencies listed above.

## When To Ask

Ask the user before proceeding when:

- a rule here conflicts with the requested change
- a dependency contract appears to require sibling repository changes
- a migration would delete, rename, or reinterpret persisted data
- a Temporal namespace/task queue/workflow ID/search attribute change is required
- a public API error shape or DTO contract would change
- a private dependency cannot be fetched and the task depends on it
- a test requires external services that are not available
- a convention is present in code but not documented here and multiple interpretations are plausible

If the behavior is present in `puria/md`, it is acceptable inspiration for this repository. If it is not present there, not documented here, and not clear from Credimi code, ask.
