# Scoreboard interop matrix extension (`wallets_credentials`)

**Date:** 2026-05-28  
**Status:** Approved (design)  
**Based on:** `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`, `docs/superpowers/plans/2026-05-27-scoreboard-interop-matrix.md`

## Goal

Extend `/scoreboard/interop` and `GET /api/scoreboard/interop` with a new interoperability mode:

- `wallets_credentials` (Wallet x Credential)

while keeping existing `wallets_issuers` support.

The new mode uses the `credentials` relation on `pipeline_scoreboard_cache` as the column axis.

## Scope decisions

1. Add a new mode on the same endpoint/page (do not replace existing mode).
2. Default UI mode becomes `wallets_credentials` when no explicit query parameter is present.
3. Keep existing run-weighted, Cartesian aggregation semantics.
4. Enrich credential column metadata in backend response (no extra UI fetches).
5. Use one generic metadata contract for all row/column entities across all modes.

## Metadata contract (all axes, all modes)

Both `rows[]` and `columns[]` use the same entity interface:

```json
{
  "id": "record_id",
  "name": "Display Name",
  "subtitle": "Optional subtitle",
  "avatar_url": "Optional absolute or API-resolved image URL",
  "path": "owner/entity/canonified-name"
}
```

Field rules:

- `id`: required (PocketBase record id)
- `name`: required
- `path`: required canonified path
- `subtitle`: optional
- `avatar_url`: optional

Mode-specific defaults:

- `wallets_credentials` credential columns:
  - `name`: credential name
  - `subtitle`: related credential issuer name (when relation exists)
  - `avatar_url`: credential avatar, fallback issuer avatar
- `wallets_issuers`:
  - same interface; values populated from wallet/issuer records where available

## API design

### Endpoint

```http
GET /api/scoreboard/interop?mode=wallets_credentials|wallets_issuers
```

- Public (no auth), same route group pattern as current scoreboard public endpoints.
- Invalid or missing mode -> `400`.

### Response shape

```json
{
  "mode": "wallets_credentials",
  "row_axis": "wallet",
  "column_axis": "credential",
  "rows": [
    {
      "id": "wallet_id",
      "name": "Wallet A",
      "subtitle": "v0.7.0",
      "avatar_url": "https://...",
      "path": "org/wallets/wallet-a"
    }
  ],
  "columns": [
    {
      "id": "credential_id",
      "name": "PID Credential",
      "subtitle": "German PID Issuer",
      "avatar_url": "https://...",
      "path": "org/credentials/pid-credential"
    }
  ],
  "cells": [
    {
      "row_id": "wallet_id",
      "column_id": "credential_id",
      "pipeline_count": 2,
      "total_runs": 184,
      "total_successes": 156,
      "success_rate": 84.7826,
      "status": "flaky"
    }
  ]
}
```

### Aggregation semantics (unchanged)

For each cache row:

- read row relation ids and column relation ids according to selected mode
- skip if either side is empty
- apply Cartesian attribution to row x column pairs
- add full row `total_runs` and `total_successes` to each pair
- track distinct pipeline ids for `pipeline_count`

Cell metrics:

```text
total_runs      = sum(total_runs)
total_successes = sum(total_successes)
success_rate    = total_successes / total_runs * 100
status          = stable|flaky|failing|broken by existing thresholds
```

Notes:

- Only emit cells with `total_runs > 0`.
- Rows/columns included only when they appear in at least one emitted cell.
- Sorting remains stable by name (case-insensitive).

## Backend architecture

Adopt a mode-driven configuration to avoid duplicated logic.

### Core pattern

Introduce an internal mode registry describing:

- row axis label
- column axis label
- row relation field name on `pipeline_scoreboard_cache`
- column relation field name on `pipeline_scoreboard_cache`
- row metadata resolver
- column metadata resolver

`buildInteropMatrix` remains generic and unchanged in behavior; it receives normalized inputs and metadata maps from mode-specific loaders.

### `wallets_credentials` metadata resolution

For credential columns:

1. Load credential records from relation ids.
2. Resolve `name`, `path`, and credential avatar candidate.
3. Resolve related issuer (if present) and issuer name/avatar.
4. Build metadata:
   - `subtitle = issuer.name` (if available)
   - `avatar_url = credential.avatar || issuer.avatar || empty`

Resolution must be defensive for missing/stale relations and continue with partial optional fields.

## Frontend architecture

### Routing and default mode

- Page stays `/scoreboard/interop`.
- Add/keep mode query parameter (`?mode=`).
- If mode is absent in URL, frontend requests `wallets_credentials` by default.

### Rendering

- Reuse one matrix grid and one cell renderer for all modes.
- Header components consume generic metadata (`name`, `subtitle`, `avatar_url`, `path`).
- Cell rendering unchanged (`%`, `successes/runs`, pipeline count, status color).

### Mode selector

Add mode pills/tabs:

- Wallet x Credential (default)
- Wallet x Issuer

Switch updates query parameter and refetches API mode.

## Error handling and edge cases

1. Unsupported mode -> `400` with clear mode guidance.
2. Missing credential issuer relation:
   - empty `subtitle`
   - avatar fallback only uses available sources.
3. Missing/deleted related records:
   - unresolved entities are skipped safely.
4. No data for mode:
   - render empty state consistent with existing scoreboard patterns.

## Testing strategy

### Go unit tests

- Mode validation accepts `wallets_credentials` and `wallets_issuers`, rejects others.
- Aggregation parity:
  - Cartesian behavior for multi-wallet/multi-credential rows
  - run-weighted sums and status bands unchanged
- Metadata resolver tests:
  - credential name/path mapping
  - subtitle from issuer name
  - avatar fallback order:
    1) credential avatar
    2) issuer avatar
    3) empty

### Go handler tests

- `200` for `wallets_credentials` with expected JSON shape and unified metadata keys.
- `200` for `wallets_issuers` remains valid.
- `400` for missing/invalid mode.

### Webapp checks

- Typecheck/lint for extended metadata types.
- Mode selector default behavior and query sync.
- Header rendering with/without `subtitle` and `avatar_url`.

## Compatibility and rollout

- Non-breaking API extension: existing mode remains.
- Shared entity interface adds optional fields; clients can ignore unknown keys.
- Frontend default mode change affects only `/scoreboard/interop` entry behavior when no mode parameter is supplied.

## Out of scope

- Additional modes beyond `wallets_credentials` and `wallets_issuers`.
- Click-through from matrix cells to filtered pipeline/result views.
- New caching/materialization strategy for matrix endpoint.
