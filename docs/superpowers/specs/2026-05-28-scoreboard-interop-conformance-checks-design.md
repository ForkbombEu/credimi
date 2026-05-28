# Scoreboard interop matrix extension (`wallets_conformance_checks`)

**Date:** 2026-05-28  
**Status:** Approved (design)  
**Based on:** `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`, `docs/superpowers/specs/2026-05-28-scoreboard-interop-wallet-credential-design.md`

## Goal

Extend `/scoreboard/interop` and `GET /api/scoreboard/interop` with a new interoperability mode:

- `wallets_conformance_checks` (Wallet × Conformance Check)

while keeping existing `wallets_credentials` and `wallets_issuers` support.

The new mode uses the `conformance_checks` JSON field on `pipeline_scoreboard_cache` as the column axis.

## Scope decisions

1. Add a new mode on the same endpoint/page (do not replace existing modes).
2. Existing default mode (`wallets_credentials`) remains unchanged.
3. Keep existing run-weighted, Cartesian aggregation semantics.
4. `conformance_checks` is a JSON field (not a relation field) storing canonical path strings. Column metadata resolution is split: backend sends minimal placeholder metadata, frontend enriches at render time from the already-loaded conformance store.
5. Columns are per-check-path (not per-suite). One column per unique conformance check path appearing in cache data.
6. Check paths carry the full canonical path in the `path` field for hub page linking.

## Metadata contract (unchanged)

Rows and columns use the same entity interface established for all interop modes:

```json
{
  "id": "record_id_or_path",
  "name": "Display Name",
  "subtitle": "Optional subtitle",
  "avatar_url": "Optional absolute or API-resolved image URL",
  "path": "canonified/path"
}
```

Mode-specific defaults for `wallets_conformance_checks`:

- **Rows (wallets):** resolved from `wallets` PB collection as in other modes.
- **Columns (conformance checks):**
  - Backend sends `id` = full check path, `name` = humanized file stem, `path` = full check path
  - Frontend overrides `name`, `subtitle`, `avatar_url` from conformance store if available:
    - `name` = check file stem (e.g., `"WEBUILD-VP001-x509-direct_post-post-dcql-sd_jwt"`)
    - `subtitle` = suite name (e.g., `"WEBUILD Interoperability Test Bed"`)
    - `avatar_url` = suite logo
  - Backend's `name` serves as a fallback when the conformance store hasn't loaded yet (SSR, direct API consumers)

## API design

### Endpoint

```http
GET /api/scoreboard/interop?mode=wallets_conformance_checks
```

- Public (no auth), same route group as other interop modes.
- Invalid mode → 400.

### Valid modes

| Value | Row axis | Column axis | Column field type |
|-------|----------|-------------|-------------------|
| `wallets_credentials` | wallet | credential | relation (M2M) |
| `wallets_issuers` | wallet | issuer | relation (M2M) |
| `wallets_conformance_checks` (new) | wallet | conformance_check | JSON (string array) |

### Response shape

```json
{
  "mode": "wallets_conformance_checks",
  "row_axis": "wallet",
  "column_axis": "conformance_check",
  "rows": [
    {
      "id": "wallet_id",
      "name": "EUDI Reference Wallet",
      "subtitle": "v0.7.0",
      "avatar_url": "https://...",
      "path": "org/wallets/eudi-reference"
    }
  ],
  "columns": [
    {
      "id": "openid4vp_wallet/1.0/webuild/WEBUILD-VP001-x509-direct_post-post-dcql-sd_jwt",
      "name": "Webuild Vp001 X509 Direct Post Post Dcql Sd Jwt",
      "subtitle": null,
      "avatar_url": null,
      "path": "openid4vp_wallet/1.0/webuild/WEBUILD-VP001-x509-direct_post-post-dcql-sd_jwt"
    }
  ],
  "cells": [
    {
      "row_id": "wallet_id",
      "column_id": "openid4vp_wallet/1.0/webuild/WEBUILD-VP001-x509-direct_post-post-dcql-sd_jwt",
      "pipeline_count": 2,
      "total_runs": 184,
      "total_successes": 156,
      "success_rate": 84.7826,
      "status": "flaky"
    }
  ]
}
```

## Backend architecture

### Mode config extension

Add `ColumnIsPathBased` boolean to `interopModeConfig` to signal that the column relation field is a JSON string array rather than a PocketBase relation field:

```go
type interopModeConfig struct {
    RowRelationField    string
    ColumnRelationField string
    RowAxis             string
    ColumnAxis          string
    RowCollection       string
    ColumnCollection    string
    ColumnIsPathBased   bool
}
```

New mode config:

| Field | Value |
|-------|-------|
| `RowRelationField` | `"wallets"` |
| `ColumnRelationField` | `"conformance_checks"` |
| `RowAxis` | `"wallet"` |
| `ColumnAxis` | `"conformance_check"` |
| `RowCollection` | `"wallets"` |
| `ColumnCollection` | `""` (empty — no PB collection) |
| `ColumnIsPathBased` | `true` |

### Path-based column ID extraction

When `ColumnIsPathBased` is true, read column IDs from PocketBase's JSON field value (stored as `[]interface{}`) rather than using `GetStringSlice()` (which only handles relation fields). Each element is cast to string and skipped if empty.

### Column metadata resolution

For path-based columns, the backend emits minimal entities:

- `id` = full path string
- `name` = humanized file stem: split path by `/`, take last segment, strip extension, replace `-` and `_` with spaces, apply `strings.Title` (or equivalent words.Title)
- `path` = full path string (for hub page linking)
- `subtitle`, `avatar_url` = `null` (frontend fills these in)

No filesystem I/O, no PB collection lookups, no new API endpoint.

### Aggregation semantics (unchanged)

For each cache row:

- read row relation ids (`wallets` field, M2M relation)
- read column ids (`conformance_checks` field, JSON string array)
- skip if either side is empty
- apply Cartesian attribution to row × column pairs
- add full row `total_runs` and `total_successes` to each pair
- track distinct pipeline ids for `pipeline_count`

Cell metrics:

```
total_runs      = sum(total_runs)
total_successes = sum(total_successes)
success_rate    = total_successes / total_runs * 100
status          = stable|flaky|failing|broken by existing thresholds
```

## Frontend architecture

### Path resolution

New utility `webapp/src/lib/scoreboard/interop/resolve-conformance.ts`:

```ts
export function resolveConformanceCheck(
    path: string,
    standards: readonly Standard[]
): ConformanceMetadata | undefined
```

Logic:

1. Split path by `/` → `[standardUid, versionUid, suiteUid, ...checkParts]`
2. Walk the standards tree: match `standard.uid` → find version by `version.uid` → find suite by `suite.uid`
3. Return `{ name: checkFileName, subtitle: suite.name, avatar_url: suite.logo }`
4. Return `undefined` if any tree node is missing

### Grid enrichment

In `matrix-grid.svelte`, when the mode is `wallets_conformance_checks`, for each column header:

1. Call `resolveConformanceCheck(column.id, get().standards)`
2. If resolved, override `name`, `subtitle`, `avatar_url` for display
3. If unresolved (store not loaded yet, or path doesn't match), use backend-provided placeholder `name`
4. Column `path` (full check path) is used for hub page linking as-is

The conformance store is loaded at app startup (via `load()` from `$lib/conformance/store.svelte.ts`), called from the root layout or equivalent. The store uses `$state` so the grid updates reactively when the store populates.

### Mode selector

Add a third option to the existing mode pills/tabs on `/scoreboard/interop`:

- Wallet x Credential
- Wallet x Issuer  
- Wallet x Conformance Checks (new)

URL: `?mode=wallets_conformance_checks`.

Default mode remains `wallets_credentials` (unchanged from prior PR).

### i18n

New key: `interop_mode_wallets_conformance_checks` = `"Wallet x Conformance Checks"`

## Error handling and edge cases

1. **Cache row has empty `conformance_checks`:** skip that row (same as other modes with empty column relations).
2. **Conformance store not loaded:** render backend placeholder names; grid updates when store populates reactively.
3. **Path doesn't match any known suite:** `resolveConformanceCheck` returns `undefined`; fall back to backend `name`.
4. **Malformed path:** fewer than 4 segments → `resolveConformanceCheck` returns `undefined`.
5. **Unsupported mode:** 400 with listing of valid modes.

## Testing strategy

### Go unit tests

- Mode validation accepts `wallets_conformance_checks`, rejects others.
- Config relation tests for path-based mode field mapping.
- `getPathIDs` — JSON field extraction (string array, empty, non-string elements).
- `conformanceCheckName` — humanize with underscores, dashes, dots.
- Aggregation parity: Cartesian behavior for multi-wallet rows with conformance check columns.
- Builder test: inputs with path-based column IDs produce correct cells.

### Go handler tests

- 200 for `wallets_conformance_checks` with expected JSON shape.
- Existing modes (`wallets_credentials`, `wallets_issuers`) still return 200.
- 400 for missing/invalid mode.

### Webapp checks

- Typecheck/lint for resolve utility and grid changes.
- Mode selector includes all three options.
- Column headers render enriched metadata when store is loaded.
- Hub links point to correct conformance check paths.

## Compatibility and rollout

- Non-breaking API extension: existing modes unchanged.
- Same metadata contract; optional fields already handled by existing renderer.
- No new collections, migrations, or filesystem dependencies.

## Out of scope

- Additional modes beyond the three current ones.
- Suites-as-columns granularity.
- Click-through from matrix cells to filtered pipeline/result views.
- New caching/materialization strategy for matrix endpoint.
