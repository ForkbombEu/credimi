# Scoreboard interop — axes-only refactor

**Date:** 2026-05-29  
**Status:** Approved  
**Based on:** `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`, subsequent interop mode extension specs

## Goal

Replace the **mode enum** (`wallets_credentials`, `wallets_issuers`, …) with a **hub-collection axis registry**. The API accepts any valid row/column hub pair; the UI keeps six **featured pairs** as tabs. Remove duplicated mode/axis definitions across Go and TypeScript.

## Scope decisions

| Topic | Decision |
|-------|----------|
| Approach | In-place refactor in `handlers` package (no new top-level Go package) |
| API pair policy | **Open** — any distinct hub pair is valid; empty/sparse matrix when cache has no data |
| UI | **Curated** — same six featured tabs; row/column pickers deferred |
| Query params | `row` + `column` (hub collection slugs); both required |
| Legacy `?mode=` | **400** — no alias, no redirect |
| Response | Drop `mode`; `InteropAxis` is `{ hub_collection, path_based }` only |
| Bare page URL | Redirect to `?row=wallets&column=credentials` |
| `path_based` | Per-axis flag; path entity loading works on **row or column** side |
| Axis registry fields | `hub_collection`, `cache_field`, `path_based` (explicit cache fields — no derivation) |
| Frontend labels | `Record<InteropHubCollection, EntityData>` from `$lib/global/entities` |
| Out of scope | `/axes` metadata endpoint, Go package extraction, axis swap, row/column pickers, conformance SSR store fix, i18n cleanup beyond removing unused `interop_mode_*` keys |

## Axis registry

Hub collection is the **public identifier** for an axis. Cache field names stay internal to scan/load.

| Hub collection | Cache field | path_based |
|----------------|-------------|------------|
| `wallets` | `wallets` | false |
| `credential_issuers` | `issuers` | false |
| `credentials` | `credentials` | false |
| `verifiers` | `verifiers` | false |
| `use_cases_verifications` | `use_case_verifications` | false |
| `conformance-checks` | `conformance_checks` | true |

Entity loading uses `hub_collection` as the PocketBase/resolver collection key (unchanged from today). Wallet version labels apply when `row.hub_collection == "wallets"`.

## API

### Endpoint

```http
GET /api/scoreboard/interop?row=wallets&column=credential_issuers
```

| Property | Value |
|----------|--------|
| Auth | None (public route group unchanged) |
| Required query | `row`, `column` — hub collection slugs from registry |

### Validation (all → 400)

- `row` or `column` missing
- Unknown hub collection slug
- `row === column`
- `mode` query param present (explicit rejection)

Error hint lists valid hub collection slugs.

### Response JSON

```json
{
  "row": { "hub_collection": "wallets", "path_based": false },
  "column": { "hub_collection": "credential_issuers", "path_based": false },
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

- No `mode` field.
- Both `row` and `column` include `path_based` from the axis registry.
- Cell metrics, status bands, and entity metadata contract unchanged.

### Aggregation (unchanged)

1. List `pipeline_scoreboard_cache` records.
2. For each record, read row axis `cache_field` and column axis `cache_field`.
3. Skip record if either ID set is empty.
4. Cartesian product per record; run-weighted sums; status from `success_rate`.
5. Resolve row/column entities (PB load or inline path entities when `path_based`).

## Backend implementation

### New file: `scoreboard_interop_axes.go`

```go
type interopAxis struct {
    HubCollection string
    CacheField    string
    PathBased     bool
}

var interopAxisRegistry = map[string]interopAxis{ ... } // keyed by hub_collection
```

Helpers:

- `getInteropAxis(hubCollection string) (interopAxis, bool)`
- `supportedInteropHubCollections() []string`

### Changes to `scoreboard_interop.go`

- Remove `interopMode`, `interopModeConfig`, `interopModeConfigs`, mode string helpers.
- `HandleInteropMatrix`: parse `row`/`column`; reject `mode`; validate pair.
- `loadInteropMatrixFromCache(app, rowAxis, colAxis interopAxis)` replaces mode-based load.
- `scanInteropCacheRecords(records, rowAxis, colAxis)` — read each side's `CacheField`; build path entities on whichever side has `PathBased: true`.
- `buildInteropMatrix`: accept row/column `InteropAxis` in response assembly; remove hardcoded wallet/issuer defaults.
- Route `QuerySearchAttributes`: replace `mode` with required `row` and `column`.

### Resolvers

No new resolvers. Existing `interopEntityResolvers` keyed by collection name (`wallets`, `credential_issuers`, …) continue to apply via `hub_collection`.

## Frontend

### New files

**`interop-hub-collections.ts`**

```ts
export const INTEROP_HUB_COLLECTIONS = [
  'wallets',
  'credential_issuers',
  'credentials',
  'verifiers',
  'use_cases_verifications',
  'conformance-checks'
] as const;

export type InteropHubCollection = (typeof INTEROP_HUB_COLLECTIONS)[number];

export function isInteropHubCollection(value: string): value is InteropHubCollection { ... }
```

**`interop-entity-data.ts`**

```ts
export const interopEntityData: Record<InteropHubCollection, EntityData> = {
  wallets: entities.wallets,
  credential_issuers: entities.credential_issuers,
  credentials: entities.credentials,
  verifiers: entities.verifiers,
  use_cases_verifications: entities.use_cases_verifications,
  'conformance-checks': entities.conformance_checks
};
```

**`featured-pairs.ts`** (replaces `modes.ts`)

```ts
export type InteropPair = {
  row: InteropHubCollection;
  column: InteropHubCollection;
};

export const FEATURED_INTEROP_PAIRS = [
  { row: 'wallets', column: 'credentials' },
  { row: 'wallets', column: 'credential_issuers' },
  { row: 'wallets', column: 'verifiers' },
  { row: 'wallets', column: 'use_cases_verifications' },
  { row: 'wallets', column: 'conformance-checks' },
  { row: 'use_cases_verifications', column: 'conformance-checks' }
] as const satisfies readonly InteropPair[];

export const DEFAULT_INTEROP_PAIR = FEATURED_INTEROP_PAIRS[0];
```

No label functions in featured pairs — labels come from `interopEntityData` at render time.

### Removed files

- `modes.ts`, `modes.test.ts`
- `axes.ts`, `axes.test.ts`

### Updated files

| File | Change |
|------|--------|
| `types.ts` | Remove `InteropMode`; drop `mode` from `InteropMatrixResponse`; `InteropAxis` = `{ hub_collection, path_based }` |
| `+page.ts` | Redirect when params missing; validate with `isInteropHubCollection`; fetch `?row=&column=`; return `{ matrix, row, column }` |
| `+page.svelte` | Tab links use hub slugs; tab label = `interopEntityData[row].labels.plural × interopEntityData[column].labels.plural`; active when row+column match |
| `to-view-matrix.ts` | Corner label from `interopEntityData` singular labels; remove `axisLabel` option |
| `matrix-grid.svelte` | Pass entity data lookup instead of `interopAxisLabel` |
| Tests | New tests for hub collections + entity data map; update `to-view-matrix.test.ts` fixtures (no `key` on axis) |

### i18n

Remove unused `interop_mode_*` message keys. Corner label uses entity singular labels (e.g. `Wallet ↓ / Issuer →`) — update or replace `interop_matrix_corner_label` usage accordingly.

## Featured pairs (UI)

Same six views as today, new URLs:

| Tab label (from EntityData plurals) | URL |
|-----------------------------------|-----|
| Wallets × Credentials | `?row=wallets&column=credentials` |
| Wallets × Issuers | `?row=wallets&column=credential_issuers` |
| Wallets × Verifiers | `?row=wallets&column=verifiers` |
| Wallets × Use case verifications | `?row=wallets&column=use_cases_verifications` |
| Wallets × Conformance checks | `?row=wallets&column=conformance-checks` |
| Use case verifications × Conformance checks | `?row=use_cases_verifications&column=conformance-checks` |

Default redirect: first row (`wallets` × `credentials`).

## Testing

| Layer | Focus |
|-------|--------|
| Go unit | Axis registry lookup; cache field wiring; `path_based` on row side; aggregation/status unchanged |
| Go handler | 200 shape (no `mode`); 400 for missing params, unknown hub, equal axes, `mode` param present |
| Webapp unit | `INTEROP_HUB_COLLECTIONS` exhaustiveness; `interopEntityData` covers all hubs; redirect logic; hub-pair tab URLs |
| Webapp | `bun run check` |

## Success criteria

- [ ] `GET /api/scoreboard/interop?row=wallets&column=credentials` returns valid matrix JSON without `mode`
- [ ] `?mode=wallets_credentials` returns 400
- [ ] Missing `row` or `column` returns 400
- [ ] `/scoreboard/interop` redirects to default pair URL
- [ ] Six featured tabs render with EntityData labels and correct matrix data
- [ ] Any valid hub pair returns a matrix (may be empty)
- [ ] `path_based` axis works as row (e.g. `row=conformance-checks&column=wallets`)
- [ ] `go test -tags=unit ./pkg/internal/apis/handlers/...` and `cd webapp && bun run check` pass

## Migration notes

- **Breaking API change** for any consumer using `?mode=`. No compatibility shim.
- Update docs referencing interop modes (`docs/src/content/docs/software-architecture/scoreboard.md`, etc.) in implementation pass or follow-up.

## References

- Original matrix design: `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`
- Entity display data: `webapp/src/lib/global/entities.ts`
- Current handler: `pkg/internal/apis/handlers/scoreboard_interop.go`
