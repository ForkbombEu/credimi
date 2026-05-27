# Scoreboard interop matrix (`/scoreboard/interop`)

**Date:** 2026-05-27  
**Status:** Approved  
**Methodology:** Tracer bullet — API + thin page first (Wallet × Issuer), then expand pair modes and UX.

## Goal

Public **matrix comparison table** showing interoperability test results between ecosystem entities. Each cell summarizes how pipelines that exercise a given pair have performed: pipeline count, run-weighted success rate, and successes/runs totals that always agree.

**v1 pair mode:** Wallets × Issuers only.  
**Data source:** `pipeline_scoreboard_cache` (same read model as `/scoreboard`).

## Decisions (brainstorming)

| Topic | Decision |
|-------|----------|
| Route | `/scoreboard/interop` (public) |
| Axes | Rows = wallets ↓, columns = issuers → (`WALLET ↓ / ISSUER →`) |
| Entity set | Data-driven — only wallets/issuers appearing in cache relation fields |
| Pipeline attribution | Cartesian — pipeline with wallets `{W1,W2}` and issuers `{I1}` contributes to `(W1,I1)` and `(W2,I1)` |
| Aggregation | **Run-weighted:** Σ `total_successes` / Σ `total_runs` (not average of per-pipeline rates) |
| Data path | **Dedicated API** `GET /api/scoreboard/interop` (Go aggregates; UI does not pivot) |
| Status colors | Bands on `success_rate`: Stable ≥90%, Flaky 70–89%, Failing 50–69%, Broken &lt;50% |
| v1 scope | Wallet × Issuer only; other modes and Wallet×Specs later |

## Coherent cell metric

One ratio drives every numeric display in a filled cell:

```
total_runs      = Σ total_runs      over contributing pipelines
total_successes = Σ total_successes over contributing pipelines
success_rate    = total_successes / total_runs   (when total_runs > 0)
pipeline_count  = |distinct pipeline IDs|
status          = band(success_rate)
```

**UI (filled cell):**

- Headline: `success_rate` as rounded integer percent (e.g. `85%`)
- Subline: `{total_successes}/{total_runs}` (same ratio as headline)
- Footer: `{pipeline_count} pipe.` (singular/plural i18n)
- Background: status band color

**Empty cell:** “Not tested”, no metrics, no status color.

## Product & UX

### Page

- **URL:** `/scoreboard/interop`
- **Audience:** Public (unauthenticated), consistent with `/scoreboard`
- **Load:** SvelteKit `+page.ts` calls `GET /api/scoreboard/interop?mode=wallets_issuers` with SSR `fetch`
- **Navigation:** Link from `/scoreboard` header; optional link from public homepage later

### Layout (v1)

- Top: title + description + status **legend** (Broken / Failing / Flaky / Stable)
- Matrix: sticky row headers (wallets with name + version when available) and column headers (issuers)
- Corner cell: `WALLET ↓ / ISSUER →`
- Footnote (short): relations reflect **last successful execution** per pipeline; Cartesian attribution; private pipelines may hide pipeline links elsewhere

### Deferred

- Cross-mode pills (Wallet×Verifier, Issuer×Verifier, Wallet×Specs)
- Axis swap toggle
- Hub catalog padding (show published entities with no data)
- Cell click → filtered scoreboard/pipeline list
- Response caching / materialized matrix collection

## API

### Endpoint

```http
GET /api/scoreboard/interop?mode=wallets_issuers
```

| Property | Value |
|----------|--------|
| Auth | None |
| Registration | New public `routing.RouteGroup` (e.g. `ScoreboardInteropPublicRoutes`), `BaseURL: /api`, path `/scoreboard/interop` |
| Errors | Existing `apierror` JSON shape for this route group |

### Query: `mode`

| Value | Row axis | Column axis | Relation fields on cache row |
|-------|----------|-------------|------------------------------|
| `wallets_issuers` (v1) | wallet | issuer | `wallets`, `issuers` |
| `wallets_verifiers` (future) | wallet | verifier | `wallets`, `verifiers` |
| `issuers_verifiers` (future) | issuer | verifier | `issuers`, `verifiers` |

Invalid or missing `mode` → `400` with clear message. Unknown future mode same.

### Response JSON

```json
{
  "mode": "wallets_issuers",
  "row_axis": "wallet",
  "column_axis": "issuer",
  "rows": [
    {
      "id": "record_id",
      "name": "EUDI Reference Wallet",
      "path": "owner/wallets/eudi-reference",
      "version_label": "v0.7.0"
    }
  ],
  "columns": [
    {
      "id": "record_id",
      "name": "German PID Issuer",
      "path": "owner/issuers/german-pid"
    }
  ],
  "cells": [
    {
      "row_id": "wallet_record_id",
      "column_id": "issuer_record_id",
      "pipeline_count": 1,
      "total_runs": 184,
      "total_successes": 156,
      "success_rate": 84.78,
      "status": "flaky"
    }
  ]
}
```

- `success_rate`: float 0–100, full precision in JSON; UI rounds for display
- `status`: `stable` | `flaky` | `failing` | `broken`
- Emit a cell only when `total_runs > 0`
- `rows` / `columns`: sorted by display `name` (stable, case-insensitive); include only IDs that appear in at least one cell

### Aggregation algorithm (Go)

1. List all records in `pipeline_scoreboard_cache` (paginate internally if needed).
2. For each cache row `P`:
   - Read relation ID slices for row-axis and column-axis collections.
   - Skip `P` if either set is empty.
   - For each `(row_id, column_id)` in Cartesian product, add `P` to that cell’s accumulator:
     - Track distinct `pipeline` record IDs (or cache row’s pipeline FK) for `pipeline_count`
     - Add `total_runs` and `total_successes` from `P`
3. For each non-empty accumulator, compute `success_rate` and `status`.
4. Resolve row/column metadata by loading `wallets` / `issuers` records (name, canonified path; wallet version from `wallet_versions` relation on cache row when pairing that wallet — v1: best-effort version label on row entity).

### Implementation files

```
pkg/internal/apis/handlers/scoreboard_interop.go
pkg/internal/apis/handlers/scoreboard_interop_test.go
pkg/internal/apis/RoutesRegistry.go   # register public route group
```

Reuse patterns from `scoreboard.go` (collection access, canonify paths, tests with PB test app).

## Frontend

### Route

```
webapp/src/routes/(public)/scoreboard/interop/
  +page.ts
  +page.svelte
```

### Library

```
webapp/src/lib/scoreboard/interop/
  types.ts              # mirrors API DTOs
  status.ts             # status → Tailwind classes (bands duplicated from server for styling only)
  matrix-cell.svelte
  matrix-grid.svelte
```

- **No** client-side pivot of PocketBase rows in v1
- Reuse `$lib/scoreboard/entity-display` or hub chip patterns for row/column headers where practical

### i18n

New keys: page title/description, “Not tested”, footnote, legend labels, `pipe.` / `pipes.`, corner axis label.

## Tracer bullet plan

1. **Backend:** `buildInteropMatrix` pure logic + handler + route registration + table-driven Go tests (Cartesian, multi-pipeline sum, empty cache, single cell).
2. **Frontend:** `+page.ts` fetch + minimal grid with real data.
3. **Polish:** Sticky headers, status colors, legend, link from `/scoreboard`, footnote.
4. **Verify:** `go test -tags=unit` for handler; `cd webapp && bun run check` for types.

## Data semantics & limitations

Document in UI footnote and API docs:

1. **Last-success snapshot:** `wallets` / `issuers` on a cache row come from the last successful execution, not historical per-pair coverage.
2. **Cartesian attribution:** One pipeline with multiple wallets and/or issuers fills multiple cells; each cell sums that pipeline’s full run totals (appropriate for “any pipeline that tests this pair together”).
3. **Not global totals:** Summing `total_runs` across cells double-counts pipelines that appear in multiple cells — intentional for per-pair view, not a platform-wide run count.
4. **Private pipelines:** Cache row may exist without visible `expand.pipeline`; matrix still uses wallet/issuer relations when present.

## Wallet × Specs (out of scope)

`conformance_checks` is a string array (paths), not M2M to hub entities. Requires normalization (path → suite/spec entity) before a fourth matrix mode. Defer until a separate design approves matching rules.

## Testing

| Layer | Focus |
|-------|--------|
| Go unit | Matrix builder: Cartesian, sums, rate, status bands, empty input, mode validation |
| Go handler | HTTP 200 shape, 400 bad mode, integration with test PB fixtures if available |
| Webapp | Optional smoke; prefer Go as source of truth for math |

## Success criteria

- [ ] `GET /api/scoreboard/interop?mode=wallets_issuers` returns valid matrix JSON from real cache data
- [ ] `/scoreboard/interop` renders matrix; filled cells show % and fraction that match (`successes/runs`)
- [ ] Empty pairs show “Not tested”
- [ ] Status colors match band thresholds on `success_rate`
- [ ] Link from `/scoreboard` to interop page works

## References

- Scoreboard cache: `docs/src/content/docs/software-architecture/scoreboard.md`
- Aggregation workflow: `pkg/workflowengine/workflows/scoreboard.go`
- Mock / UX reference: interop matrix screenshot (2026-05-27)
- Architecture placeholder: Hub/Comparison Tool in `docs/src/content/docs/software-architecture/building-blocks.md`
