# Scoreboard interop — tiered matrix & unified axis registry

**Date:** 2026-06-01  
**Status:** Approved  
**Supersedes (partial):** Tiering and aggregation deferred in `docs/superpowers/specs/2026-05-29-scoreboard-interop-axes-only-design.md`  
**Based on:** `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`, handoff `handoff-scoreboard-interop-tiered-matrix-20260530.md`

## Goal

Evolve the public interoperability matrix (`/scoreboard/interop`, `GET /api/scoreboard/interop`) from a **flat** entity×entity grid to a **tiered** matrix where **each axis** independently supports **group** (collapsed) and **leaf** (expanded) tiers. One **generic** engine drives all featured hub pairs; **conformance** uses the same tier machinery with path/suite grouping.

## Locked decisions

| Topic | Decision |
|-------|----------|
| Scope | **General per-axis model** — all six featured tabs; no pair-specific aggregation forks |
| Conformance | **D2** — suite→check tiering in the same release; Go suite grouping + conformance standards loaded for interop SSR |
| Tiering | Applies to **rows and columns**; four cell modes from `(row_tier, col_tier)` |
| Leaf×leaf attribution | **Cartesian** within each cache record (unchanged) |
| Any group tier | **Existential-once** per `(row_key, col_key)` per cache record — never sum child leaf cells into parent |
| Wallet rollup | **Inclusive** — wallet-only cache rows count toward **group** cells |
| Wallet expanded | **Synthetic leaf** per wallet when version unknown: `{wallet_id}::__no_version__`, `version_label` null; UI shows i18n **undefined version** label |
| Real wallet leaves | Leaf ID = `wallet_versions` record ID |
| Registry | Single `interopAxisRegistry` with optional `Tier` + `buildEntity`; delete separate resolver interface/map |
| API | `row_groups` / `row_leaves` / `column_groups` / `column_leaves` + tier-tagged `cells[]`; client expand state; default **collapsed**; no refetch on expand |
| Breaking change | Remove flat `rows` / `columns` from response |

## Out of scope

- `GET /api/scoreboard/interop/axes` metadata endpoint
- Hub catalog padding (entities with no cache data)
- Axis swap toggle, row/column pickers
- Deduplicating cache write logic in `scoreboard.go` (triplicated hub knowledge — separate cleanup)
- i18n cleanup beyond new keys for tier UI and undefined version

---

## Axis tier registry

Each hub collection declares **how that axis participates in tiering** when used as `row` or `column`. Pairs are `rowAxis.Tier × colAxis.Tier` only.

### Tier profile kinds

| Profile | `tiered` | Group key | Leaf keys | Notes |
|---------|----------|-----------|-----------|--------|
| **Flat** | `false` | — | IDs from `cache_field` | Current behavior for that hub on that axis |
| **PB hierarchical** | `true` | Parent record ID | Child IDs + FK | PocketBase load for entities |
| **Path hierarchical** | `true` | Suite composite key | Check path strings | `path_based`; suite parser in Go |

### Registry shape (Go)

```go
type interopAxis struct {
    HubCollection string
    CacheField    string
    PathBased     bool
    Tier          *interopAxisTier // nil = flat
    buildEntity   func(app core.App, axisRecord *core.Record, cacheRecord *core.Record) (InteropMatrixEntity, error)
}

type interopAxisTier struct {
    GroupCollection string // parent hub (same as HubCollection for PB hierarchies)
    LeafCollection  string
    LeafCacheField  string
    LeafParentField string // FK on leaf → group
    NoLeafSentinel  string // e.g. "__no_version__"
}
```

- `buildEntity` **nil** for path-based check leaves (inline at scan); suite **groups** built via path grouper.
- **Wallet `buildEntity`:** `version_label` from `cacheRecord` when non-nil; `cacheRecord=nil` when wallets is **column** axis (preserves axes-only behavior).
- **Credentials / use_cases_verifications:** parent enrichment inside `buildEntity` (replaces resolver + `loadInteropRelatedRecords`).

### Per-hub configuration

| Hub | Profile | Group | Leaf | Leaf cache field | Parent FK |
|-----|---------|-------|------|------------------|-----------|
| `wallets` | PB hierarchical | wallet | `wallet_versions` | `wallet_versions` | `wallet` on version |
| `credential_issuers` | PB hierarchical | issuer | `credentials` | `credentials` | `credential_issuer` |
| `credentials` | Flat | — | credential | `credentials` | — |
| `verifiers` | PB hierarchical | verifier | `use_cases_verifications` | `use_case_verifications` | `verifier` |
| `use_cases_verifications` | Flat | — | use case | `use_case_verifications` | — |
| `conformance-checks` | Path hierarchical | `standardUid/versionUid/suiteUid` | check path | `conformance_checks` | parsed from path |

### Cache scan: resolve tier keys per axis per record

For each `pipeline_scoreboard_cache` record:

1. **Flat axis:** emit leaf tier only — one key per ID in `cache_field` (or inline path entity).
2. **PB hierarchical:**
   - Emit **leaf** keys from `leaf_cache_field` (map leaf → `group_id` via `leaf_parent_field`).
   - **Inclusive orphan:** if `cache_field` contains parent ID with no matching leaf on this record, emit synthetic leaf `{parent_id}{no_leaf_sentinel}`.
   - Emit **group** keys for every parent present (from leaves’ parents and/or bare `cache_field` entries when inclusive).
3. **Path hierarchical:** parse each path in `conformance_checks`; assign suite **group** key + **leaf** path; malformed paths: same fallback as today (`conformanceCheckName`, skip grouping if unparsable).

**Entity load:** one `buildEntity` per unique leaf ID; pass **first cache record** that references that ID as representative `cacheRecord`. Group entities loaded by group ID.

### Code cleanup

- Remove `scoreboard_interop_resolvers.go` interface + `interopEntityResolvers` map (logic → registry `buildEntity`, optional `scoreboard_interop_entities.go`).
- Remove `resolveWalletVersionLabels` pre-pass.
- Frontend: delete `interop-entity-data.ts` if still a pass-through; keep `interop-hub-collections.ts` for labels only.

---

## Aggregation

### Cell coordinate

```text
(row_tier, row_key, column_tier, column_key)
```

| Tier | Key |
|------|-----|
| `group` | Parent / suite group ID |
| `leaf` | Leaf record ID, check path, or synthetic sentinel |

API precomputes cells for all four tier combinations needed for the pair. UI filters by expand state.

### Per cache record

1. Build set `R` of `(row_tier, row_key)` from row axis scan.
2. Build set `C` of `(column_tier, column_key)` from column axis scan.
3. For each `(r, c) ∈ R × C`:

| Row tier | Col tier | Rule |
|----------|----------|------|
| `leaf` | `leaf` | **Cartesian** — one accumulator add per pair |
| `group` | `leaf` | **Existential-once** — at most one add per cache record |
| `leaf` | `group` | **Existential-once** |
| `group` | `group` | **Existential-once** |

**Existential-once:** increment cell at most once per cache record for that coordinate; do not multiply by leaf count under a group; do not sum child leaf cells into parent group cells.

**Metrics per cell** (unchanged): distinct `pipeline_id`, summed `total_runs` / `total_successes`, `success_rate`, `status` band. Omit empty cells; UI shows “not tested”.

### Product notes

- **Leaf×leaf + wallets row:** row keys are **version** IDs (or synthetic no-version), not wallet IDs.
- **Double-fold:** wallet group × issuer group counts once per cache record when both sides match, not once per version×credential pair.

---

## API

### Request

```http
GET /api/scoreboard/interop?row=<hub>&column=<hub>
```

Unchanged validation: both required; distinct hubs; unknown hub → 400; `mode` param → 400 with hub hint.

### Response (breaking)

```json
{
  "row": { "hub_collection": "wallets", "path_based": false, "tiered": true },
  "column": { "hub_collection": "credential_issuers", "path_based": false, "tiered": true },
  "row_groups": [
    {
      "id": "wallet_rec_id",
      "name": "EUDI Reference Wallet",
      "path": "owner/wallets/eudi",
      "child_count": 10,
      "avatar_url": "https://…"
    }
  ],
  "row_leaves": [
    {
      "id": "version_rec_id",
      "parent_id": "wallet_rec_id",
      "name": "EUDI Reference Wallet",
      "path": "owner/wallets/eudi/versions/…",
      "version_label": "v0.7.0",
      "avatar_url": "https://…"
    },
    {
      "id": "wallet_rec_id::__no_version__",
      "parent_id": "wallet_rec_id",
      "name": "EUDI Reference Wallet",
      "path": "owner/wallets/eudi",
      "version_label": null
    }
  ],
  "column_groups": [],
  "column_leaves": [],
  "cells": [
    {
      "row_id": "wallet_rec_id",
      "column_id": "issuer_rec_id",
      "row_tier": "group",
      "column_tier": "group",
      "pipeline_count": 1,
      "total_runs": 100,
      "total_successes": 80,
      "success_rate": 80.0,
      "status": "flaky"
    }
  ]
}
```

| Field | Rules |
|-------|--------|
| `row_groups` / `column_groups` | Present when `tiered: true` on that axis; includes `child_count` |
| `row_leaves` / `column_leaves` | All leaves for matrix; flat axis fills only leaves |
| `cells[].row_tier` / `column_tier` | `"group"` \| `"leaf"` |
| Group entity | No `version_label`; may include `subtitle` / enriched fields per hub |
| Leaf entity | `parent_id` when tiered; wallet leaves use `version_label` or null |

Update `docs/public/API/openapi.yml`.

### Featured pair behavior (derived, not special-cased)

| Pair | Row axis | Column axis | Default visible grid |
|------|----------|-------------|----------------------|
| wallets × credential_issuers | tiered | tiered | group × group |
| wallets × credentials | tiered | flat | group × leaf |
| wallets × verifiers | tiered | tiered | group × group |
| wallets × use_cases_verifications | tiered | flat | group × leaf |
| wallets × conformance-checks | tiered | path tiered | group × group |
| credential_issuers × verifiers | tiered | tiered | group × group |

---

## Frontend

### Load

- `+page.ts`: fetch matrix; **load conformance standards** (same source as other scoreboard/conformance pages) so `resolveConformanceCheck` works on SSR.
- Pass standards into `toViewMatrix` via page data or initialized store.

### Types (`webapp/src/lib/scoreboard/interop/types.ts`)

- `InteropAxis`: add `tiered: boolean`.
- `InteropMatrixGroup`, `InteropMatrixLeaf` (leaf includes `parent_id?`).
- `InteropMatrixCell`: add `row_tier`, `column_tier`.
- `InteropMatrixResponse`: replace `rows`/`columns` with four tier arrays.

### View model (`to-view-matrix.ts`)

- Inputs: response, `expandedRowGroups: Set<string>`, `expandedColumnGroups: Set<string>`, standards.
- Cell map key: `` `${rowTier}:${rowId}:${colTier}:${colId}` ``.
- **Visible rows (tiered):** collapsed → `row_groups` entries; expanded group → that group’s `row_leaves` only (group row hidden while expanded).
- **Visible columns:** same pattern.
- **Undefined version:** when `version_label` is null on wallet leaf, `displaySubtitle` = `m.interop_matrix_version_undefined()` (or equivalent).

### Grid (`matrix-grid.svelte`)

- Chevron on group headers toggles expand (client state, default empty → all collapsed).
- Group header subtitle: `(N)` child count from `child_count`.
- Cells resolve via tier-aware key from visible row/column context.

### i18n

- New key for undefined wallet version label (not literal `<undefined>` in API).

---

## Conformance (D2)

### Go

- Port suite grouping rules from `webapp/src/lib/conformance/group.ts` (`parsePath`, group by `standard • version • suite` title).
- Suite **group id:** stable composite of `standardUid`, `versionUid`, `suiteUid`.
- Group display name aligns with TS `GroupedPathsBySuite.title`.

### SSR

- Interop page must not read an uninitialized conformance store (fixes current `matrix-grid.svelte` gap).

---

## Testing

### Go (`-tags=unit`)

| Area | Cases |
|------|--------|
| Attribution | Leaf×leaf Cartesian; group×leaf / leaf×group / group×group existential-once; reject sum-of-leaves into group |
| Tier scan | Wallet inclusive orphan + synthetic leaf; issuer+credential mapping; path suite grouping |
| Registry | `buildEntity` parity (migrate tests from `scoreboard_interop_resolvers_test.go`) |

### Webapp (Vitest)

| Area | Cases |
|------|--------|
| `to-view-matrix` | Collapsed vs expanded row/column lists; tier-aware cell lookup |
| Display | Null `version_label` → undefined i18n subtitle |

### Manual

- Each featured tab: collapse/expand row and column groups; double-fold; conformance suite expand; wallet no-version leaf row.

---

## Implementation order

1. Extend `interopAxis` registry + merge `buildEntity` (delete resolver file).
2. Tier scan + aggregation (unit tests first for §3 cases).
3. Response DTO + handler + OpenAPI.
4. Frontend types, load standards, `to-view-matrix`, `matrix-grid` expand UI.
5. Conformance Go grouper + end-to-end manual pass on wallets×conformance-checks.

---

## References

- `pkg/internal/apis/handlers/scoreboard_interop.go` — current scan/aggregate
- `pkg/internal/apis/handlers/scoreboard_interop_axes.go` — axis registry
- `pkg/internal/apis/handlers/scoreboard_interop_resolvers.go` — to be removed
- `webapp/src/lib/conformance/group.ts` — suite grouping precedent
- Architecture review: `architecture-review-20260530-scoreboard-interop.html` (triplication, SSR gap)
