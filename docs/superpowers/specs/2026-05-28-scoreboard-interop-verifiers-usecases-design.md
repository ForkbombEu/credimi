# Scoreboard interop matrix extension (`wallets_verifiers`, `wallets_use_case_verifications`)

**Date:** 2026-05-28
**Status:** Approved (design)
**Based on:** `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`, `docs/superpowers/specs/2026-05-28-scoreboard-interop-wallet-credential-design.md`

## Goal

Add two new interoperability modes to `/scoreboard/interop` and `GET /api/scoreboard/interop`:

- `wallets_verifiers` (Wallet × Verifier)
- `wallets_use_case_verifications` (Wallet × Use Case Verification)

while keeping existing `wallets_credentials` and `wallets_issuers`.

Both new modes use relation fields that already exist on `pipeline_scoreboard_cache` (`verifiers` and `use_case_verifications`), populated by the scoreboard aggregation workflow.

## Scope decisions

1. Add two new entries to the existing `interopModeConfigs` registry — no new code paths.
2. Add a new `use_cases_verifications` branch in `interopEntityFromRecord` for metadata enrichment.
3. Verifiers require no dedicated metadata branch — the generic fallback covers them with a one-line `AvatarURL` addition.
4. Rename `buildCredentialEntityMetadata` → `buildEnrichedEntityMetadata` since it now serves both credentials and use case verifications.
5. Keep `wallets_credentials` as the default UI mode.
6. Reuse all existing aggregation, rendering, mode selector, and sorting behavior unchanged.

## Metadata enrichment contract

Same unified entity interface across all modes — no new fields:

```json
{
  "id": "record_id",
  "name": "Display Name",
  "path": "canonified/path",
  "subtitle": "Optional subtitle",
  "avatar_url": "Optional image URL"
}
```

Mode-specific enrichment:

| Mode | Column entity | `subtitle` | `avatar_url` |
|---|---|---|---|
| `wallets_verifiers` | verifier | — (nil) | verifier logo |
| `wallets_use_case_verifications` | use_case_verification | verifier name | use_case logo → verifier logo → nil |
| `wallets_credentials` (existing) | credential | issuer name | credential logo → issuer logo → nil |
| `wallets_issuers` (existing) | issuer | — (nil) | issuer logo |

## Backend design

### Mode config entries

Two new entries in `interopModeConfigs`:

```go
interopModeWalletsVerifiers: {
    RowRelationField:    "wallets",
    ColumnRelationField: "verifiers",
    RowAxis:             "wallet",
    ColumnAxis:          "verifier",
    RowCollection:       "wallets",
    ColumnCollection:    "verifiers",
},
interopModeWalletsUseCaseVerifications: {
    RowRelationField:    "wallets",
    ColumnRelationField: "use_case_verifications",
    RowAxis:             "wallet",
    ColumnAxis:          "use_case_verification",
    RowCollection:       "wallets",
    ColumnCollection:    "use_cases_verifications",
},
```

New mode constants:

```go
const (
    interopModeWalletsVerifiers              interopMode = "wallets_verifiers"
    interopModeWalletsUseCaseVerifications   interopMode = "wallets_use_case_verifications"
)
```

### Entity metadata resolution

**`interopEntityFromRecord` — new `use_cases_verifications` branch** (before the generic fallback):

Load the use_case_verification record, resolve its verifier relation, build enriched metadata with verifier name as `subtitle` and logo fallback chain.

**`interopEntityFromRecord` — generic fallback** (handles verifiers, issuers, etc.):

Add `AvatarURL` resolution from the record's `avatar` and `logo` fields:

```go
return InteropMatrixEntity{
    ID:        record.Id,
    Name:      record.GetString("name"),
    Path:      path,
    AvatarURL: firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo")),
}, nil
```

**Rename:** `buildCredentialEntityMetadata` → `buildEnrichedEntityMetadata`. The function signature and logic are unchanged; only the name reflects its broader use.

### Reused unchanged

- `loadInteropMatrixFromCache` — mode config drives relation field selection via `modeConfig.RowRelationField` / `modeConfig.ColumnRelationField`. New modes work without changes.
- `buildInteropMatrix` — Cartesian aggregation, run-weighted sums, status bands, sorting — all generic.
- `HandleInteropMatrix` — `isSupportedInteropMode` covers the new modes automatically.
- `ScoreboardInteropPublicRoutes` — route definition unchanged.
- `mergeInteropEntities` — generic; uses `modeConfig.RowCollection` / `modeConfig.ColumnCollection`.

## Frontend design

### Types

```ts
export type InteropMode =
  | 'wallets_credentials'
  | 'wallets_issuers'
  | 'wallets_verifiers'
  | 'wallets_use_case_verifications';
```

`SUPPORTED_MODES` array in `+page.ts` grows to include all four. Default stays `wallets_credentials`.

### i18n

New keys in `webapp/messages/en.json`:

```json
"interop_mode_wallets_verifiers": "Wallet × Verifier",
"interop_mode_wallets_use_case_verifications": "Wallet × Use case verification"
```

### Components

No new components or component changes. Reused unchanged:

- `matrix-grid.svelte` — renders `subtitle`, `avatar_url`, and `name` generically.
- `matrix-cell.svelte` — cell metrics unchanged.
- `+page.svelte` — mode selector renders all 4 pills. Legend, footnote, corner label all unchanged.

## API design

### Endpoint (unchanged)

```http
GET /api/scoreboard/interop?mode=wallets_verifiers
GET /api/scoreboard/interop?mode=wallets_use_case_verifications
```

Public, no auth. Invalid mode → `400`.

### Response shape

Same as existing modes. `mode`, `row_axis`, and `column_axis` reflect the selected mode:

```json
{
  "mode": "wallets_verifiers",
  "row_axis": "wallet",
  "column_axis": "verifier",
  "rows": [{ "id": "...", "name": "Wallet A", "subtitle": "v0.7.0", "avatar_url": "...", "path": "org/wallets/wallet-a" }],
  "columns": [{ "id": "...", "name": "Verifier X", "avatar_url": "...", "path": "org/verifiers/verifier-x" }],
  "cells": [{ "row_id": "...", "column_id": "...", "pipeline_count": 1, "total_runs": 50, "total_successes": 45, "success_rate": 90.0, "status": "stable" }]
}
```

```json
{
  "mode": "wallets_use_case_verifications",
  "row_axis": "wallet",
  "column_axis": "use_case_verification",
  "rows": [{ "id": "...", "name": "Wallet A", ... }],
  "columns": [{ "id": "...", "name": "PID Verification", "subtitle": "Verifier X", "avatar_url": "...", "path": "org/verifiers/verifier-x/pid-verification" }],
  "cells": [...]
}
```

### Aggregation semantics (unchanged)

- Cartesian attribution from cache row relation IDs.
- Run-weighted sums (`Σ total_successes / Σ total_runs`).
- Status bands: Stable ≥90%, Flaky 70–89%, Failing 50–69%, Broken <50%.
- Only emit cells with `total_runs > 0`.
- Rows/columns only include entities appearing in at least one emitted cell.
- Sorting by name (case-insensitive, stable).

## Error handling and edge cases

1. Unsupported mode → `400` with mode guidance (all 4 modes enumerated in message).
2. Missing verifier relation on use_case_verification → `subtitle` nil, logo fallback skips verifier.
3. Missing/deleted related records → `app.FindRecordById` errors are silently skipped (existing pattern).
4. Cache row with empty `verifiers` or `use_case_verifications` field → row skipped in aggregation (existing pattern).
5. No data for a mode → empty `rows`, `columns`, `cells` arrays in response.

## Testing strategy

### Go unit tests

- Mode validation: `TestInteropModeValidation` extended to accept 4 modes, reject others.
- Mode config relations: `TestInteropModeConfigRelations` extended with new entries.
- Use case metadata resolver:
  - `subtitle` from verifier name (when relation exists).
  - `avatar_url` fallback: use_case logo → verifier logo → nil.
  - Missing verifier → nil subtitle, nil avatar (when no use_case logo either).
  - Missing verifier → use_case logo still used for avatar.
- Verifier generic fallback: `AvatarURL` populated from `logo` field.
- Aggregation parity: existing Cartesian/run-weighted tests still pass.
- Handler: `200` for `wallets_verifiers`, `200` for `wallets_use_case_verifications`, `400` for invalid mode.

### Webapp

- Typecheck pass with extended `InteropMode` union.
- Mode selector renders 4 pills/tabs.
- Default mode remains `wallets_credentials`.
- Column headers render `subtitle` and `avatar_url` from new modes.

## Files changed

```
pkg/internal/apis/handlers/scoreboard_interop.go       # 2 new mode constants + configs, 1 new collection branch, rename function, AvatarURL on fallback
pkg/internal/apis/handlers/scoreboard_interop_test.go   # Extended mode/config/metadata tests
webapp/src/lib/scoreboard/interop/types.ts              # Extended InteropMode union, SUPPORTED_MODES array
webapp/src/routes/(public)/scoreboard/interop/+page.ts  # SUPPORTED_MODES includes new modes
webapp/messages/en.json                                  # 2 new i18n keys
```

No new files. No breaking changes.

## Compatibility and rollout

- Non-breaking API extension: existing modes unchanged.
- No structural changes to existing types or components.
- Frontend default mode unchanged (`wallets_credentials`).
- PocketBase schema already supports both relation fields — no migrations needed.

## Out of scope

- Additional axis pairings beyond these four modes.
- Wallet × Specs matrix.
- Cell click-through to filtered scoreboard/pipeline list.
- Response caching.
- Axis swap toggle.
