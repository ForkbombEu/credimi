# Scoreboard Feature Architecture

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend (Svelte/TS)                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────┐      ┌──────────────────────┐            │
│  │ /scoreboard          │      │ Homepage section     │            │
│  │ (Public)             │      │ (random sample)      │            │
│  │                      │      │                      │            │
│  │ - Paginated table    │      │ - Pipeline cards     │            │
│  │ - Sortable columns   │      │ - Success summary    │            │
│  │ - Entity display     │      │ - Entity links       │            │
│  └──────────────────────┘      └──────────────────────┘            │
│           │                             │                           │
└───────────┼─────────────────────────────┼───────────────────────────┘
            │                             │
            └──────────────┬──────────────┘
                           │
                           ▼
                ┌──────────────────────┐
                │ PocketBase API       │
                │ pipeline_scoreboard_ │
                │ cache                │
                └──────────────────────┘
                           ▲
                           │
┌──────────────────────────┴──────────────────────────────────────────┐
│                       Backend (Go + Temporal)                         │
├───────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │ AggregateScoreboardWorkflow (Temporal)                     │     │
│  │                                                            │     │
│  │  1. List org namespaces                                    │     │
│  │  2. GET /api/pipeline/scoreboard/{namespace} per tenant    │     │
│  │  3. Merge pipeline stats                                   │     │
│  │  4. POST /api/pipeline/scoreboard/save-results             │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────┐     │
│  │ Scoreboard handlers (pkg/internal/apis/handlers)           │     │
│  │                                                            │     │
│  │  - GET  /api/pipeline/scoreboard/{namespace}               │     │
│  │  - POST /api/pipeline/scoreboard/save-results              │     │
│  │  - POST /api/pipeline/scoreboard/aggregate/start           │     │
│  │  - DELETE /api/pipeline/scoreboard/aggregate/schedule/{id} │     │
│  └────────────────────────────────────────────────────────────┘     │
│                            │                                         │
└────────────────────────────┼─────────────────────────────────────────┘
                             │
                             ▼
                ┌────────────────────────┐
                │ PocketBase Collections │
                ├────────────────────────┤
                │ - pipeline_scoreboard_ │
                │   cache (read model)   │
                │ - pipeline_results     │
                │ - pipelines            │
                │ - wallets, issuers,    │
                │   verifiers, etc.      │
                └────────────────────────┘
```

## Data Flow

### Public Scoreboard Flow

1. User navigates to `/scoreboard`
2. SvelteKit `load` calls `Scoreboard.loadData()` from `$lib/scoreboard`
3. Frontend queries PocketBase `pipeline_scoreboard_cache` with expanded relations
4. `ScoreboardTable` renders a paginated, sortable TanStack table
5. Rows link to marketplace pipeline pages via entity display helpers

### Homepage Summary Flow

1. Public homepage loads `loadScoreboardSummary()` from `scoreboard-section.svelte`
2. Frontend fetches a small random sample from `pipeline_scoreboard_cache`
3. Cards show pipeline name, success rate, and related entities

### Aggregation Flow

1. An operator or scheduler starts `POST /api/pipeline/scoreboard/aggregate/start`
2. Temporal runs `AggregateScoreboardWorkflow` in the `default` namespace
3. For each org namespace, the workflow calls `GET /api/pipeline/scoreboard/{namespace}`
4. Per-namespace stats are merged and posted to `POST /api/pipeline/scoreboard/save-results`
5. `pipeline_scoreboard_cache` is refreshed (truncate + upsert by pipeline)

## Integration Points

### Current State

- Public scoreboard UI at `/scoreboard`
- Homepage scoreboard section on the public landing page
- PocketBase-backed read model (`pipeline_scoreboard_cache`)
- Temporal aggregation workflow and save endpoint
- Shared frontend module at `webapp/src/lib/scoreboard`

### Removed

- `/my/scoreboard` and `/my/scoreboard/[type]/[id]` routes
- Legacy tabbed scoreboard UI and OpenTelemetry viewer
- `/api/my/results` and `/api/all-results` handlers (commented out in `scoreboard_handler.go`)

## File Structure

```
credimi/
├── pkg/internal/apis/handlers/
│   ├── scoreboard.go                    (active aggregation + save handlers)
│   ├── scoreboard_handler.go            (legacy OTel handler, commented out)
│   └── scoreboard_test.go
├── pkg/workflowengine/workflows/
│   └── scoreboard.go                    (AggregateScoreboardWorkflow)
├── webapp/src/lib/scoreboard/
│   ├── index.ts
│   ├── functions.ts                     (PocketBase query)
│   ├── table.svelte / table.svelte.ts
│   ├── columns/                         (TanStack column definitions)
│   ├── entity-display/                  (avatars, lists, links)
│   └── extras/pipeline-content-summary.svelte
├── webapp/src/routes/
│   ├── (public)/scoreboard/
│   │   ├── +page.ts
│   │   └── +page.svelte
│   └── (public)/_partials/scoreboard-section.svelte
└── docs/
    ├── SCOREBOARD.md
    ├── ARCHITECTURE.md                  (this file)
    └── SUMMARY.md
