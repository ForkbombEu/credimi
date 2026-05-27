# Codebase Structure

**Analysis Date:** 2026-05-27

## Directory Layout

```
DIDimo/                          # module github.com/forkbombeu/credimi
├── main.go                      # PocketBase entry (→ cmd.Start)
├── cmd/                         # CLI: credimi root, pipeline/schema subcommands, seeds, template
├── pkg/
│   ├── routes/                  # PocketBase Setup: proxy, RegisterMyRoutes, PB hooks
│   ├── internal/                # Application core (not importable externally)
│   │   ├── apis/                # Route registry + HTTP handlers
│   │   ├── routing/             # RouteGroup, validation registration
│   │   ├── middlewares/         # Auth, validation, error middleware
│   │   ├── pb/                  # PocketBase record hooks (pipelines, namespaces, schedules)
│   │   ├── temporalclient/      # Per-namespace Temporal client cache
│   │   ├── canonify/            # Canonified naming hooks/resolver
│   │   ├── pipeline/            # Server-side pipeline path resolution (non-workflow)
│   │   ├── apierror/            # Handler JSON errors
│   │   └── errorcodes/          # CRE domain codes
│   ├── workflowengine/          # Temporal workflows, activities, pipeline engine, workers
│   ├── templateengine/          # Conformance YAML/template helpers
│   ├── generate_client/         # Code generation utilities
│   └── utils/                   # Env, URL, shared helpers
├── webapp/                      # SvelteKit 2 + Vite (bun); UI proxied by PocketBase
│   └── src/
│       ├── routes/              # SvelteKit file-based routes
│       ├── lib/                 # Feature libraries (pipeline, workflows, scoreboard, …)
│       └── modules/             # Shared UI, PocketBase types, i18n, forms (@ alias)
├── pb_migrations/               # PocketBase collection migrations (JS)
├── pb_hooks/                    # PocketBase JS hooks (orgs, auth, features)
├── schemas/                     # JSON/YAML schemas (pipeline, OIDC, VLEI, checks)
├── config_templates/            # Conformance/wallet/issuer template trees
├── migrations/                  # Go SQL migrations (blank-imported from cmd)
├── fixtures/                    # Test PocketBase data
├── docs/                        # Astro/Starlight documentation site
├── deployment/                  # Deploy configs
├── .github/workflows/           # CI (lint, test, release)
├── Makefile                     # dev, test, lint, fmt, generate
├── Procfile.dev                 # hivemind: API + UI (+ Temporal via compose)
└── docker-compose.yaml          # Temporal stack for local/prod-like dev
```

## Directory Purposes

**`cmd/`:**
- Purpose: Process and CLI entrypoints separate from library `pkg/`.
- Contains: `credimi.go` (`Start`), `cli/` (pipeline, schema commands), `seeds/`, `template/`.
- Key files: `cmd/credimi.go`, `cmd/cli/pipeline.go`.

**`pkg/internal/apis/handlers/`:**
- Purpose: One file (or small group) per API domain; each exports `*Routes routing.RouteGroup`.
- Contains: `pipeline_handler.go`, `workflows_handlers.go`, `wallet_handler.go`, `mobile_runners_handlers.go`, `*_test.go`.
- Key files: Handler vars listed in `pkg/internal/apis/RoutesRegistry.go`.

**`pkg/workflowengine/`:**
- Purpose: All Temporal workflow and activity code plus dynamic pipeline runtime.
- Contains: `workflows/`, `activities/`, `pipeline/`, `registry/`, `hooks/`, `mobilerunnersemaphore/`.
- Key files: `pipeline/pipeline.go`, `registry/registry.go`, `hooks/hook.go`.

**`webapp/src/routes/`:**
- Purpose: SvelteKit pages and layouts; URL structure for the product.
- Contains: Route groups `(public)`, `(nru)` (not registered user), `my/` (authenticated app).
- Key files: `+layout.ts` (global `ssr = false`), `my/pipelines/`, `my/tests/runs/`.

**`webapp/src/lib/`:**
- Purpose: Domain-specific client logic decoupled from route files.
- Contains: `pipeline/`, `pipeline-form/`, `workflows/`, `scoreboard/`, `conformance/`, `wallet/`.
- Key files: `lib/pipeline/queue.ts`, `lib/pipeline/actions.ts`, `lib/workflows/queries.ts`.

**`webapp/src/modules/`:**
- Purpose: Reusable UI kit, PocketBase integration, i18n, forms — imported as `@/…`.
- Contains: `components/ui/`, `pocketbase/types/`, `collections-components/`, `i18n/`.
- Key files: `modules/pocketbase/index.ts`, `modules/pocketbase/types/`.

**`pb_migrations/` / `pb_hooks/`:**
- Purpose: PocketBase schema and server-side JS event hooks.
- Contains: Timestamped `*_created_*.js` migrations; `organizations.pb.js`, `users.pb.js`, etc.
- Key files: `pb_migrations/1765364510_created_pipeline_results.js`, `pb_hooks/organizations.pb.js`.

**`schemas/` / `config_templates/`:**
- Purpose: Machine-readable contracts and canned conformance configs.
- Contains: `schemas/pipeline/pipeline_schema.json`; per-standard template YAML under `config_templates/`.

## Key File Locations

**Entry Points:**
- `main.go`: Delegates to `cmd.Start()`.
- `cmd/credimi.go`: PocketBase app creation, `routes.Setup`, CLI subcommands, `app.Start()`.
- `webapp/src/routes/+layout.ts`: Global load (feature flags, maintenance), disables SSR.

**Configuration:**
- `.env.example`: Documented env vars (never commit `.env` secrets).
- `Makefile`: Build, test, lint, dev orchestration.
- `Procfile.dev` / `Procfile`: Process definitions for hivemind/Docker.
- `webapp/svelte.config.js`: Kit adapter (Bun), path aliases (`@`, `$pipeline-form`, etc.).
- `.golangci.yaml`, `.editorconfig`: Go/TS formatting and lint rules.

**Core Logic:**
- `pkg/routes/routes.go`: Central server wiring.
- `pkg/internal/apis/RoutesRegistry.go`: Lists all HTTP route groups.
- `pkg/workflowengine/pipeline/pipeline.go`: Dynamic Pipeline Workflow implementation.
- `pkg/workflowengine/registry/registry.go`: Pipeline step type registry.

**Testing:**
- Go unit: co-located `*_test.go` under `pkg/…` with `//go:build unit` where applicable; `make test`.
- Webapp unit: `webapp/**/*.test.ts`, `*.svelte.test.ts`; `cd webapp && bun run test:unit`.
- Webapp E2E: `webapp/e2e/`; requires running stack.
- Fixtures: `fixtures/test_pb_data/` (read-only in tests).

## Naming Conventions

**Files:**
- Go handlers: `{domain}_handler.go` or `{domain}_handlers.go` (e.g. `pipeline_handler.go`, `workflows_handlers.go`).
- Go tests: `{file}_test.go`, table tests named `Test{Type}_{Scenario}`.
- Route groups: `{Domain}Routes`, `{Domain}TemporalInternalRoutes` exported vars in handlers package.
- Svelte routes: SvelteKit conventions — `+page.svelte`, `+layout.ts`, `+page.ts`; private partials under `_partials/`.
- Svelte libs: kebab-case files (`runner-select-list.svelte`); state modules often `*.svelte.ts`.

**Directories:**
- Go: lowercase single-word or short phrases (`workflowengine`, `mobilerunnersemaphore`).
- Webapp routes: parenthesized groups for layout/auth — `(public)`, `(nru)`, `(group)` for layout resets.
- PocketBase migrations: Unix timestamp prefix — `1769505309_created_mobile_runners.js`.

## Where to Add New Code

**New authenticated REST endpoint:**
- Primary code: `pkg/internal/apis/handlers/{feature}_handler.go` — define `var FeatureRoutes routing.RouteGroup`.
- Registration: append to `RouteGroups` or `RouteGroupsNotExported` in `pkg/internal/apis/RoutesRegistry.go`.
- Tests: `pkg/internal/apis/handlers/{feature}_handler_test.go` with `//go:build unit` if no Temporal/PB integration.

**New Temporal workflow or activity:**
- Workflow: `pkg/workflowengine/workflows/{name}.go` implementing `workflowengine.Workflow`.
- Activity: `pkg/workflowengine/activities/{name}.go` embedding `BaseActivity`.
- Registration: add to `OrgWorkers` or `DefaultWorkers` in `pkg/workflowengine/hooks/hook.go`; for pipeline steps, add `registry.Registry` entry.

**New pipeline step type (`use` in YAML):**
- Registry: `pkg/workflowengine/registry/registry.go`.
- Execution/hooks: `pkg/workflowengine/pipeline/` (e.g. `mobile_automation_hooks.go` as reference).
- Schema: update `schemas/pipeline/pipeline_schema.json` if shape changes.
- UI builder: `webapp/src/lib/pipeline-form/steps/` and `steps-builder/`.

**New PocketBase collection:**
- Migration: `pb_migrations/{timestamp}_created_{collection}.js`.
- Optional JS hook: `pb_hooks/{collection}.pb.js`.
- Go hook (if invariant in Go): `pkg/internal/pb/`.
- Webapp types: regenerate or extend `webapp/src/modules/pocketbase/types/`.

**New UI page:**
- Route: `webapp/src/routes/my/{feature}/+page.svelte` (or `(public)/` for unauthenticated).
- Logic: prefer `webapp/src/lib/{feature}/` over bloated `+page.svelte`.
- Shared UI: `webapp/src/modules/components/` or `ui-custom/`.

**Utilities:**
- Go shared: `pkg/utils/`.
- TS shared: `webapp/src/modules/utils/` or `webapp/src/lib/utils/`.

## Special Directories

**`pb_data/`:**
- Purpose: Runtime PocketBase SQLite and (dev) Temporal file DB.
- Generated: Yes (local/dev).
- Committed: No (gitignored).

**`webapp/build/`, `webapp/.svelte-kit/`:**
- Purpose: Production build output and Kit generated types.
- Generated: Yes.
- Committed: Build output typically no; check CI/deploy expectations.

**`.planning/codebase/`:**
- Purpose: GSD codebase map documents for planners/executors.
- Generated: By `/gsd-map-codebase`.
- Committed: Per project policy.

**`LICENSES/`, `REUSE.toml`:**
- Purpose: REUSE compliance and license metadata.
- Generated: Partially maintained.
- Committed: Yes.

**`.gitnexus/`:**
- Purpose: Code intelligence index for GitNexus MCP.
- Generated: By `npx gitnexus analyze`.
- Committed: Optional/local.

---

*Structure analysis: 2026-05-27*
