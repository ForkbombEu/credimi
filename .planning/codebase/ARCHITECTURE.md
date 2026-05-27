<!-- refreshed: 2026-05-27 -->
# Architecture

**Analysis Date:** 2026-05-27

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Browser (SvelteKit SPA, SSR off)                       │
│                         `webapp/src/routes/`, `webapp/src/lib/`              │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │ HTTPS (PocketBase auth + custom /api/*)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│              PocketBase server (`go run main.go serve`, :8090)               │
│  ┌─────────────────────┐  ┌──────────────────────┐  ┌─────────────────────┐ │
│  │ Reverse proxy       │  │ Custom API routes    │  │ PB collections +    │ │
│  │ `pkg/routes/`       │  │ `pkg/internal/apis/` │  │ `pb_migrations/`    │ │
│  │ → ADDRESS_UI        │  │ + `pkg/internal/     │  │ + `pb_hooks/`       │ │
│  │   (SvelteKit :5100) │  │   routing/`          │  │                     │ │
│  └─────────────────────┘  └──────────┬───────────┘  └─────────────────────┘ │
└──────────────────────────────────────┼──────────────────────────────────────┘
                                       │ Temporal client (per-namespace)
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Temporal (`TEMPORAL_ADDRESS`, default :7233)              │
│  Namespace `default`: semaphore workflows, worker-manager, shared infra      │
│  Namespace `<org.canonified_name>`: conformance, wallet, pipeline, OIDC…   │
│  `pkg/workflowengine/` — workflows, activities, pipeline engine, workers     │
└───────────────────────────────────┬─────────────────────────────────────────┘
                                    │ HTTP to external mobile runners
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  External services: mobile runner (`credimi-extra`), Docker, StepCI, mail    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Process entry | Delegates to PocketBase bootstrap | `main.go`, `cmd/credimi.go` |
| App wiring | Reverse proxy, route registration, PB/JS hooks, migrations CLI | `pkg/routes/routes.go` |
| HTTP API surface | Route groups, handlers, auth hooks (login, Turnstile) | `pkg/internal/apis/`, `pkg/internal/apis/RoutesRegistry.go` |
| Route framework | Groups, validation binding, handler factories | `pkg/internal/routing/routing.go` |
| Cross-cutting HTTP | Auth (Bearer / API key), validation, errors | `pkg/internal/middlewares/` |
| Persistence hooks | Pipeline publish lock, schedules, org namespace lifecycle | `pkg/internal/pb/` |
| Temporal client | Lazy per-namespace client cache | `pkg/internal/temporalclient/client.go` |
| Workflow runtime | Workflow/activity interfaces, start helpers, CRE errors | `pkg/workflowengine/workflow.go`, `pkg/workflowengine/activity.go` |
| Dynamic pipelines | YAML-driven step execution, mobile automation hooks | `pkg/workflowengine/pipeline/` |
| Task registry | Maps pipeline step `use` keys → activities/workflows | `pkg/workflowengine/registry/registry.go` |
| Worker lifecycle | Start/stop workers per namespace, worker configs | `pkg/workflowengine/hooks/hook.go` |
| Frontend app | SvelteKit routes, domain libs, PocketBase typed client | `webapp/src/routes/`, `webapp/src/lib/`, `webapp/src/modules/` |
| Schema contracts | Pipeline JSON schema, OIDC/EUDIW/VLEI templates | `schemas/`, `config_templates/` |

## Pattern Overview

**Overall:** Multi-process **modular monolith** — PocketBase as the API host and auth/data layer, Temporal for durable orchestration, SvelteKit as a separately built UI proxied under the same origin in dev/production.

**Key Characteristics:**
- **Tenant isolation via Temporal namespaces** — `organizations.canonified_name` is the namespace for org-scoped workflows; `default` holds cross-tenant infrastructure (mobile-runner semaphore, worker-manager).
- **Declarative pipelines** — YAML/JSON workflow definitions validated against `schemas/pipeline/pipeline_schema.json`, executed by `Dynamic Pipeline Workflow` on `PipelineTaskQueue`.
- **Route groups as modules** — Each handler file exports a `routing.RouteGroup`; `RegisterMyRoutes` mounts all groups at serve time.
- **Dual auth planes** — User routes accept Bearer token or `Credimi-Api-Key` (user scope); Temporal-internal routes use internal-admin API key middleware on specific paths.

## Layers

**Presentation (webapp):**
- Purpose: Operator UI for pipelines, wallets, verifiers, conformance, organizations, workflow runs.
- Location: `webapp/src/routes/`, `webapp/src/lib/`, `webapp/src/modules/`
- Contains: Svelte 5 pages, feature modules (`pipeline-form`, `scoreboard`), shadcn-style UI in `webapp/src/modules/components/`
- Depends on: PocketBase JS SDK (`webapp/src/modules/pocketbase/`), custom `/api/*` via `fetch` with auth headers
- Used by: End users; proxied through PocketBase in `pkg/routes/routes.go`

**HTTP / application API:**
- Purpose: Custom REST beyond PocketBase CRUD — start workflows, queue pipelines, list Temporal executions, internal activity callbacks.
- Location: `pkg/internal/apis/handlers/`, wired by `pkg/internal/apis/RoutesRegistry.go`
- Contains: Handler factories, request/response DTOs, Temporal/PB orchestration
- Depends on: `pkg/internal/routing`, `pkg/internal/middlewares`, `pkg/internal/temporalclient`, `pkg/workflowengine`
- Used by: Webapp, Temporal activities (`InternalHTTPActivity`), CI integrations

**Domain / PocketBase integration:**
- Purpose: Collection rules, canonified naming, pipeline immutability when published, org namespace provisioning.
- Location: `pkg/internal/pb/`, `pkg/internal/canonify/`, `pb_hooks/`, `pb_migrations/`
- Contains: Go record hooks; JS hooks for org auth, features, audit
- Depends on: PocketBase `core.App`, Temporal namespace APIs
- Used by: All features storing entities in SQLite (`pb_data/`)

**Workflow orchestration:**
- Purpose: Long-running conformance checks, wallet tests, pipeline steps, email, Docker, HTTP chains.
- Location: `pkg/workflowengine/workflows/`, `pkg/workflowengine/activities/`, `pkg/workflowengine/pipeline/`
- Contains: Temporal workflow structs, activity implementations, pipeline executor
- Depends on: Temporal SDK, `credimi-extra` (mobile), Docker API, external HTTP
- Used by: API handlers (start workflow), pipeline child workflows, scheduled enqueue

**Infrastructure utilities:**
- Purpose: Env helpers, URL utilities, code generation, templates for conformance YAML.
- Location: `pkg/utils/`, `pkg/templateengine/`, `pkg/generate_client/`, `cmd/cli/`
- Depends on: stdlib and third-party libs only
- Used by: All layers above

## Data Flow

### Primary request path (authenticated UI → API → Temporal)

1. Browser loads SvelteKit app (proxied `/{path...}` → `ADDRESS_UI`, default `http://localhost:5100`) (`pkg/routes/routes.go`).
2. User authenticates via PocketBase; `webapp/src/modules/pocketbase/index.ts` holds `pb` client and `currentUser` store.
3. Feature code calls custom API, e.g. `POST /api/pipeline/queue` with Bearer or `Credimi-Api-Key` (`webapp/src/lib/pipeline/queue.ts` → `pb.send` / `fetch`).
4. PocketBase `OnServe` mounts route group; `routing.RegisterRoutesWithValidation` applies `RequireAuthOrAPIKey` and optional body validation (`pkg/internal/routing/routing.go`).
5. Handler resolves org namespace, canonified pipeline path, starts Temporal workflow or enqueues semaphore ticket (`pkg/internal/apis/handlers/pipeline_handler.go`, `pipeline_queue_handler.go`).
6. Temporal worker in org namespace executes workflow/activities; results persisted to `pipeline_results` and/or returned via queue status polling.

### Pipeline run with mobile automation (queue + semaphore)

1. UI detects `mobile-automation` steps in YAML (`webapp/src/lib/pipeline/utils.ts`, `webapp/src/lib/pipeline/queue.ts`) and uses `POST /api/pipeline/queue` instead of direct start.
2. Handler enqueues ticket on per-runner semaphore workflow in namespace `default` (`pkg/workflowengine/workflows/mobile_runner_semaphore.go`).
3. On grant, `StartQueuedPipelineActivity` starts Dynamic Pipeline Workflow in owner org namespace (`pkg/workflowengine/activities/queued_pipeline.go`).
4. Pipeline injects semaphore metadata in config; `mobile-automation` steps dispatch to `${runner_id}-TaskQueue` (`pkg/workflowengine/pipeline/mobile_automation_hooks.go`).
5. Completion reported via `ReportMobileRunnerSemaphoreDoneActivity` (`pkg/workflowengine/pipeline/semaphore_done.go`); UI polls `GET /api/pipeline/queue/{ticket}`.

### PocketBase CRUD path (collections)

1. UI uses typed PocketBase SDK for records (pipelines, wallets, organizations, etc.).
2. `pb_hooks/*.pb.js` enforce org membership, features, audit logging on record events.
3. Go hooks in `pkg/internal/pb/` enforce invariants (e.g. published pipeline immutability via `RegisterPipelineHooks`).

**State Management:**
- **Server:** PocketBase SQLite (`pb_data/`); Temporal workflow history (`pb_data/temporal.db` in dev).
- **Webapp:** Svelte 5 runes/stores (`webapp/src/lib/app-state/`, module-level `.svelte.ts` state); no global Redux-style store. SSR disabled globally (`webapp/src/routes/+layout.ts`).

## Key Abstractions

**`routing.RouteGroup`:**
- Purpose: Declarative bundle of HTTP routes with shared base URL, middlewares, and auth flag.
- Examples: `handlers.PipelineRoutes` (`pkg/internal/apis/handlers/pipeline_handler.go`), `handlers.WorkflowsRoutes` (`pkg/internal/apis/handlers/workflows_handlers.go`)
- Pattern: `var XRoutes routing.RouteGroup = routing.RouteGroup{...}`; registered in `RoutesRegistry.go`

**`workflowengine.Workflow` / `ExecutableActivity`:**
- Purpose: Uniform interface for Temporal registration and execution options.
- Examples: `pkg/workflowengine/workflows/openid4vp_wallet.go`, `pkg/workflowengine/activities/stepci.go`
- Pattern: Struct with `Workflow(ctx, WorkflowInput) (WorkflowResult, error)`; registered in `pkg/workflowengine/hooks/hook.go` worker configs

**`pipeline.WorkflowDefinition` + registry `use` keys:**
- Purpose: Parse and execute YAML pipeline steps (`http-request`, `mobile-automation`, `child-pipeline`, etc.).
- Examples: `pkg/workflowengine/pipeline/parser.go`, `pkg/workflowengine/registry/registry.go`
- Pattern: Registry maps string `use` → activity/workflow factory; pipeline workflow sequences steps in `pkg/workflowengine/pipeline/pipeline.go`

**Canonified paths:**
- Purpose: Stable identifiers for entities and Temporal search attributes (`PipelineIdentifier`).
- Examples: `pkg/internal/canonify/canonify.go`, used in handlers and pipeline starts

**Temporal namespace = tenant:**
- Purpose: Isolate workflow executions per organization.
- Examples: `pkg/internal/pb/namespaces.go` (`ensureNamespaceAndWorkers`), handler namespace resolution in pipeline tests/handlers

## Entry Points

**Backend server:**
- Location: `main.go` → `cmd/credimi.go` `Start()`
- Triggers: `go run main.go serve`, Procfile `API`, Docker `credimi serve`
- Responsibilities: Create PocketBase app, `routes.Setup(app)`, load `.env`, run HTTP server

**Route setup:**
- Location: `pkg/routes/routes.go` `Setup()`
- Triggers: Called once at process start from `cmd.Start()`
- Responsibilities: UI reverse proxy, `apis.RegisterMyRoutes`, PB hooks, canonify/logo/wallet hooks, JSVM + automigrate

**Temporal workers:**
- Location: `pkg/workflowengine/hooks/hook.go` (`WorkersHook`, `StartAllWorkersByNamespace`)
- Triggers: Intended on `OnServe` via `WorkersHook(app)` — **currently commented out** in `pkg/routes/routes.go`; org create path uses `HookNamespaceOrgs` → `ensureNamespaceAndWorkers` (**also commented out** in `routes.go`). Worker-manager workflow in `default` namespace can start workers when org hooks run.
- Responsibilities: Poll task queues per namespace; register workflows/activities from `OrgWorkers` / `DefaultWorkers` configs

**Webapp dev server:**
- Location: `webapp/` — `bun dev` (Vite + SvelteKit)
- Triggers: Procfile `UI`, `make dev` via hivemind
- Responsibilities: SPA; talks to PocketBase at `PUBLIC_POCKETBASE_URL`

**CLI:**
- Location: `cmd/cli/pipeline.go`, `cmd/cli/schema.go`
- Triggers: `credimi` subcommands on root Cobra command
- Responsibilities: Pipeline/schema utilities outside the HTTP server

## Architectural Constraints

- **Threading:** Go HTTP handlers are concurrent; Temporal workers run in goroutines per namespace (`go startAllWorkersByNamespace` in `WorkersHook`). PocketBase record hooks must not block on long Temporal calls without `go` (org hooks use goroutines for worker stop/start).
- **Global state:** Temporal client cache in `pkg/internal/temporalclient/client.go` (`sync.Map`); worker cancel map in `pkg/workflowengine/hooks/hook.go` (`workerCancels`). PocketBase `app` is the central dependency injected into handlers via `core.RequestEvent`.
- **Build tags:** Mobile/extra integrations may require `-tags=credimi_extra` (see `Procfile.dev` `go tool gow run -tags=credimi_extra`).
- **Private module:** `github.com/forkbombeu/credimi-extra` must remain in `go.mod` for mobile runner activities.
- **UI coupling:** Production serves prebuilt `webapp/build` via Bun adapter (`Procfile`); dev proxies live Vite server.

## Anti-Patterns

### Mixed HTTP error shapes

**What happens:** Some handlers return `apierror.APIError` JSON directly; others rely on `ErrorHandlingMiddleware` wrapper format.
**Why it's wrong:** Clients and tests must branch on response shape per endpoint.
**Do this instead:** Match the error contract of the route group you modify (`pkg/internal/middlewares/errors.go` vs `pkg/internal/apierror/apierror.go`); see `AGENTS.md` Errors section.

### Commented worker bootstrap in `routes.Setup`

**What happens:** `hooks.WorkersHook(app)` and `pb.HookNamespaceOrgs(app)` are commented in `pkg/routes/routes.go`, so a bare `serve` may not start in-process Temporal workers unless another mechanism runs them.
**Why it's wrong:** Local/dev expectations in `AGENTS.md` assume workers on server start; workflows stall without pollers.
**Do this instead:** Confirm worker startup path for your environment (re-enable hooks, separate worker process, or worker-manager workflow) before debugging "workflow started but never progresses."

### Bypassing registry for pipeline steps

**What happens:** New pipeline step types added only to YAML without a `registry.Registry` entry fail at runtime.
**Why it's wrong:** Dynamic pipeline resolves `use` keys only through the registry map.
**Do this instead:** Add `TaskFactory` in `pkg/workflowengine/registry/registry.go` and wire execution in `pkg/workflowengine/pipeline/` hooks if needed.

## Error Handling

**Strategy:** Domain **CRE codes** in `pkg/internal/errorcodes/errorcodes.go`; activities surface `ApplicationError` via `workflowengine.BaseActivity`; workflows wrap with `workflowengine.NewWorkflowError` (Temporal UI metadata). API handlers use `apierror` or middleware-wrapped errors.

**Patterns:**
- Activities: return Temporal application errors with CRE code for retry semantics (`pkg/workflowengine/activity.go`).
- Handlers: `apierror.New(...).JSON(e)` for consistent handler-local responses.
- Workflow listing: parse stored workflow errors via `workflowengine.ParseWorkflowError` where needed.

## Cross-Cutting Concerns

**Logging:** Standard library `log` in hooks and route registration; Temporal workflow loggers inside workflows.

**Validation:** Route-level `RequestSchema` → `middlewares.DynamicValidateInputByType`; handlers read typed input via `routing.GetValidatedInput[T](e)`. PocketBase collection validation via ozzo-validation in hooks.

**Authentication:** `middlewares.RequireAuthOrAPIKey()` for user routes; `middlewares.RequireInternalAdminAPIKey()` on `*TemporalInternalRoutes` groups; PocketBase built-in auth for collection APIs. Turnstile on login via `apis.HookTurnstileVerification` in `routes.Setup`.

---

*Architecture analysis: 2026-05-27*
