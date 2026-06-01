# Scoreboard Interop Tiered Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a tiered public interoperability matrix (group/leaf rows and columns, existential-once vs Cartesian attribution) with a unified Go axis registry (`buildEntity`) and conformance suite grouping in one release.

**Architecture:** Extend `interopAxis` with optional `Tier` metadata and per-hub `buildEntity`. Scan each cache record into tier-aware coordinate sets per axis; aggregate cells by `(row_tier, row_key, col_tier, col_key)`. Return `row_groups` / `row_leaves` / `column_groups` / `column_leaves` plus tier-tagged `cells`. Frontend keeps expand state client-side and resolves visible rows/columns from group vs leaf lists.

**Tech Stack:** Go 1.24 (`stretchr/testify`, PocketBase test app), SvelteKit 2 + Svelte 5, TypeScript, Paraglide i18n, Vitest.

**Spec:** `docs/superpowers/specs/2026-06-01-scoreboard-interop-tiered-matrix-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/scoreboard_interop_tier.go` | **NEW** — `interopTier`, `interopAxisCoord`, tier key resolution from cache records, existential/Cartesian aggregation |
| `pkg/internal/apis/handlers/scoreboard_interop_tier_test.go` | **NEW** — attribution + tier scan unit tests |
| `pkg/internal/apis/handlers/scoreboard_interop_conformance_paths.go` | **NEW** — parse check paths, suite group ID/title (mirror TS `parsePath` / `groupPathsBySuite`) |
| `pkg/internal/apis/handlers/scoreboard_interop_conformance_paths_test.go` | **NEW** — path/suite grouping tests |
| `pkg/internal/apis/handlers/scoreboard_interop_entities.go` | **NEW** — `buildEntity` fns for wallets, issuers, verifiers, credentials, use cases |
| `pkg/internal/apis/handlers/scoreboard_interop_entities_test.go` | **NEW** — migrate resolver entity tests here |
| `pkg/internal/apis/handlers/scoreboard_interop_axes.go` | Add `Tier *interopAxisTier`, `Tiered() bool`, wire registry entries + `NoLeafSentinel` for wallets |
| `pkg/internal/apis/handlers/scoreboard_interop.go` | New DTOs; refactor scan → tier coords → aggregate → load groups/leaves; remove wallet label pre-pass |
| `pkg/internal/apis/handlers/scoreboard_interop_test.go` | Update matrix builder/handler tests for tiered response |
| **Delete** | `scoreboard_interop_resolvers.go`, `scoreboard_interop_resolvers_test.go` |
| `docs/public/API/openapi.yml` | Tiered response schemas |
| `webapp/src/lib/scoreboard/interop/types.ts` | Tier DTOs + `row_tier` / `column_tier` on cells |
| `webapp/src/lib/scoreboard/interop/to-view-matrix.ts` | Expand-aware visible rows/columns + tier cell keys |
| `webapp/src/lib/scoreboard/interop/to-view-matrix.test.ts` | Tier visibility + cell lookup tests |
| `webapp/src/lib/scoreboard/interop/matrix-grid.svelte` | Expand toggles, group headers with child count |
| `webapp/src/routes/(public)/scoreboard/interop/+page.ts` | Load `getStandardsWithTestSuites` for conformance enrichment |
| `webapp/src/routes/(public)/scoreboard/interop/+page.svelte` | Pass standards + hold expand `Set` state |
| `webapp/messages/en.json` (+ other locales if required) | `interop_matrix_version_undefined`, expand aria labels |
| **Delete** | `webapp/src/lib/scoreboard/interop/interop-entity-data.ts` if still pass-through only |

---

### Task 1: Tier types and existential aggregation

**Files:**
- Create: `pkg/internal/apis/handlers/scoreboard_interop_tier.go`
- Create: `pkg/internal/apis/handlers/scoreboard_interop_tier_test.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go` (replace flat cell key usage later in Task 5)

- [ ] **Step 1: Write failing aggregation tests**

```go
// scoreboard_interop_tier_test.go
package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAggregateInteropCells_LeafLeafCartesian(t *testing.T) {
	t.Parallel()
	inputs := []interopTieredCacheInput{{
		PipelineID:     "p1",
		TotalRuns:      10,
		TotalSuccesses: 8,
		RowCoords: []interopAxisCoord{
			{Tier: interopTierLeaf, Key: "v1"},
			{Tier: interopTierLeaf, Key: "v2"},
		},
		ColumnCoords: []interopAxisCoord{
			{Tier: interopTierLeaf, Key: "c1"},
		},
	}}
	cells := aggregateInteropCells(inputs)
	_, ok := cells[interopMatrixCellKey{rowTier: interopTierLeaf, rowKey: "v1", colTier: interopTierLeaf, colKey: "c1"}]
	require.True(t, ok)
	_, ok = cells[interopMatrixCellKey{rowTier: interopTierLeaf, rowKey: "v2", colTier: interopTierLeaf, colKey: "c1"}]
	require.True(t, ok)
}

func TestAggregateInteropCells_GroupLeafExistentialOnce(t *testing.T) {
	t.Parallel()
	// One cache record, two leaves under same wallet group — group×leaf must NOT double-count.
	inputs := []interopTieredCacheInput{{
		PipelineID:     "p1",
		TotalRuns:      100,
		TotalSuccesses: 80,
		RowCoords: []interopAxisCoord{
			{Tier: interopTierGroup, Key: "w1"},
			{Tier: interopTierLeaf, Key: "v1"},
			{Tier: interopTierLeaf, Key: "v2"},
		},
		ColumnCoords: []interopAxisCoord{
			{Tier: interopTierLeaf, Key: "i1"},
		},
	}}
	cells := aggregateInteropCells(inputs)
	g, ok := cells[interopMatrixCellKey{rowTier: interopTierGroup, rowKey: "w1", colTier: interopTierLeaf, colKey: "i1"}]
	require.True(t, ok)
	require.Equal(t, 100, g.totalRuns)
	// Summing leaf×leaf into group would yield 200 — must not happen.
}

func TestAggregateInteropCells_GroupGroupDoubleFold(t *testing.T) {
	t.Parallel()
	inputs := []interopTieredCacheInput{{
		PipelineID:     "p1",
		TotalRuns:      50,
		TotalSuccesses: 40,
		RowCoords: []interopAxisCoord{
			{Tier: interopTierGroup, Key: "w1"},
			{Tier: interopTierLeaf, Key: "v1"},
			{Tier: interopTierLeaf, Key: "v2"},
		},
		ColumnCoords: []interopAxisCoord{
			{Tier: interopTierGroup, Key: "issuer1"},
			{Tier: interopTierLeaf, Key: "cred1"},
			{Tier: interopTierLeaf, Key: "cred2"},
		},
	}}
	cells := aggregateInteropCells(inputs)
	g, ok := cells[interopMatrixCellKey{rowTier: interopTierGroup, rowKey: "w1", colTier: interopTierGroup, colKey: "issuer1"}]
	require.True(t, ok)
	require.Equal(t, 50, g.totalRuns)
}
```

- [ ] **Step 2: Run tests — expect compile fail**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestAggregateInteropCells' -v`  
Expected: FAIL — undefined types/functions.

- [ ] **Step 3: Implement tier types and aggregator**

```go
// scoreboard_interop_tier.go (core types)
type interopTier string

const (
	interopTierGroup interopTier = "group"
	interopTierLeaf  interopTier = "leaf"
)

type interopAxisCoord struct {
	Tier interopTier
	Key  string
}

type interopTieredCacheInput struct {
	PipelineID     string
	TotalRuns      int
	TotalSuccesses int
	RowCoords      []interopAxisCoord
	ColumnCoords   []interopAxisCoord
}

type interopMatrixCellKey struct {
	rowTier interopTier
	rowKey  string
	colTier interopTier
	colKey  string
}

func aggregateInteropCells(inputs []interopTieredCacheInput) map[interopMatrixCellKey]*interopCellAccumulator {
	out := map[interopMatrixCellKey]*interopCellAccumulator{}
	for _, in := range inputs {
		if in.TotalRuns <= 0 {
			continue
		}
		seen := map[interopMatrixCellKey]struct{}{}
		for _, r := range in.RowCoords {
			for _, c := range in.ColumnCoords {
				key := interopMatrixCellKey{rowTier: r.Tier, rowKey: r.Key, colTier: c.Tier, colKey: c.Key}
				if r.Tier == interopTierGroup || c.Tier == interopTierGroup {
					if _, ok := seen[key]; ok {
						continue
					}
					seen[key] = struct{}{}
				}
				acc := out[key]
				if acc == nil {
					acc = &interopCellAccumulator{pipelineIDs: map[string]struct{}{}}
					out[key] = acc
				}
				acc.pipelineIDs[in.PipelineID] = struct{}{}
				acc.totalRuns += in.TotalRuns
				acc.totalSuccesses += in.TotalSuccesses
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestAggregateInteropCells' -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_tier.go pkg/internal/apis/handlers/scoreboard_interop_tier_test.go
git commit -m "feat(interop): add tier-aware cell aggregation"
```

---

### Task 2: Conformance path suite grouping (Go)

**Files:**
- Create: `pkg/internal/apis/handlers/scoreboard_interop_conformance_paths.go`
- Create: `pkg/internal/apis/handlers/scoreboard_interop_conformance_paths_test.go`

- [ ] **Step 1: Write failing suite grouping tests**

```go
func TestInteropSuiteGroupFromPath_Valid(t *testing.T) {
	t.Parallel()
	g, leaf, err := interopSuiteGroupFromPath("eu/a1/suite1/check1")
	require.NoError(t, err)
	require.Equal(t, "eu/a1/suite1", g.ID)
	require.Equal(t, "eu • a1 • suite1", g.Title)
	require.Equal(t, "eu/a1/suite1/check1", leaf)
}

func TestInteropSuiteGroupFromPath_Invalid(t *testing.T) {
	t.Parallel()
	_, _, err := interopSuiteGroupFromPath("bad/path")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run — expect FAIL**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestInteropSuiteGroupFromPath -v`

- [ ] **Step 3: Implement parser + grouper**

- Split path on `/`; require 4 segments (`standard`, `version`, `suite`, `test`).
- `group.ID = standard + "/" + version + "/" + suite`
- `group.Title = standard + " • " + version + " • " + suite` (match TS `groupPathsBySuite` title)
- `leaf` = full joined path (4 segments).

- [ ] **Step 4: Run — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_conformance_paths.go pkg/internal/apis/handlers/scoreboard_interop_conformance_paths_test.go
git commit -m "feat(interop): add conformance suite grouping for tiered paths"
```

---

### Task 3: Extend axis registry with tier metadata

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_axes.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_axes_test.go` (create if missing tests)

- [ ] **Step 1: Write failing tier registry tests**

```go
func TestGetInteropAxis_WalletsTiered(t *testing.T) {
	t.Parallel()
	axis, ok := getInteropAxis("wallets")
	require.True(t, ok)
	require.NotNil(t, axis.Tier)
	require.Equal(t, "wallet_versions", axis.Tier.LeafCacheField)
	require.Equal(t, "__no_version__", axis.Tier.NoLeafSentinel)
	require.True(t, axis.Tiered())
}

func TestGetInteropAxis_CredentialsFlat(t *testing.T) {
	t.Parallel()
	axis, ok := getInteropAxis("credentials")
	require.True(t, ok)
	require.Nil(t, axis.Tier)
	require.False(t, axis.Tiered())
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Add `interopAxisTier` and populate registry per spec table**

```go
type interopAxisTier struct {
	GroupCollection string
	LeafCollection  string
	LeafCacheField  string
	LeafParentField string
	NoLeafSentinel  string
}

func (a interopAxis) Tiered() bool { return a.Tier != nil }
```

Registry entries:

| Hub | Tier |
|-----|------|
| `wallets` | Leaf `wallet_versions`, parent field `wallet`, sentinel `__no_version__` |
| `credential_issuers` | Leaf `credentials`, FK `credential_issuer`, sentinel `__no_orphan__` or reuse pattern for inclusive issuer-only rows |
| `verifiers` | Leaf `use_case_verifications`, FK `verifier` |
| `credentials`, `use_cases_verifications` | `nil` |
| `conformance-checks` | Path tier: set `LeafCacheField` to `conformance_checks`, `GroupCollection` sentinel `conformance-checks` (path grouper handles IDs) |

- [ ] **Step 4: Run axis tests — PASS**

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_axes.go pkg/internal/apis/handlers/scoreboard_interop_axes_test.go
git commit -m "feat(interop): extend axis registry with tier metadata"
```

---

### Task 4: Resolve tier coordinates from cache records

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_tier.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_tier_test.go`

- [ ] **Step 1: Write failing tier resolution tests**

```go
func TestResolveAxisCoords_WalletInclusiveOrphan(t *testing.T) {
	t.Parallel()
	rec := &core.Record{}
	rec.Set("wallets", []string{"w1"})
	rec.Set("wallet_versions", []string{})
	axis, _ := getInteropAxis("wallets")
	coords := resolveAxisCoords(rec, axis)
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierGroup, Key: "w1"})
	require.Contains(t, coords, interopAxisCoord{Tier: interopTierLeaf, Key: "w1::__no_version__"})
}

func TestResolveAxisCoords_WalletWithVersion(t *testing.T) {
	t.Parallel()
	// Use test app: version record with wallet FK, cache lists version id in wallet_versions and wallet in wallets.
	// Expect group w1 + leaf versionID (no synthetic orphan).
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement `resolveAxisCoords(record, axis) []interopAxisCoord`**

**Flat:** foreach ID in `cache_field` → `{leaf, id}`.

**PB hierarchical:**

1. Load leaf IDs from `tier.leaf_cache_field`; map each to group via `tier.leaf_parent_field` on loaded leaf record (batch load in scan pass).
2. Emit `{group, parentID}` for each distinct parent.
3. Emit `{leaf, leafID}` for each leaf.
4. For each parent ID in `cache_field` with no leaf on this record → emit synthetic `{leaf, parentID + sentinel}` and ensure `{group, parentID}`.

**Path hierarchical:** foreach path in JSON field → `interopSuiteGroupFromPath`; emit `{group, group.ID}` and `{leaf, path}`; on parse error emit flat leaf only (today’s fallback name path).

- [ ] **Step 4: Run tier tests — PASS**

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_tier.go pkg/internal/apis/handlers/scoreboard_interop_tier_test.go
git commit -m "feat(interop): resolve group/leaf coords from cache records"
```

---

### Task 5: Merge `buildEntity` registry (delete resolvers)

**Files:**
- Create: `pkg/internal/apis/handlers/scoreboard_interop_entities.go`
- Create: `pkg/internal/apis/handlers/scoreboard_interop_entities_test.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_axes.go` (attach `buildEntity` on registry entries)
- Delete: `pkg/internal/apis/handlers/scoreboard_interop_resolvers.go`
- Delete: `pkg/internal/apis/handlers/scoreboard_interop_resolvers_test.go`

- [ ] **Step 1: Copy failing tests from `scoreboard_interop_resolvers_test.go`**

Adjust to call `interopAxisRegistry["credentials"].buildEntity(...)` instead of resolver interface.

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Move resolver logic into `buildEntity` closures**

- `walletBuildEntity`: path, avatar, `version_label` from `walletVersionLabelFromCacheRecord` when `cacheRecord != nil`.
- `credentialBuildEntity`: load issuer via `findRecordsByIDs` when needed.
- `useCaseBuildEntity`: load verifier similarly.
- Simple hubs: same as `simpleInteropEntityResolver`.

Wire on registry in `scoreboard_interop_axes.go`.

- [ ] **Step 4: Run entity tests — PASS**

- [ ] **Step 5: Delete resolver files; fix compile in `scoreboard_interop.go` (temporary stubs if needed)**

- [ ] **Step 6: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_entities.go pkg/internal/apis/handlers/scoreboard_interop_entities_test.go pkg/internal/apis/handlers/scoreboard_interop_axes.go
git rm pkg/internal/apis/handlers/scoreboard_interop_resolvers.go pkg/internal/apis/handlers/scoreboard_interop_resolvers_test.go
git commit -m "refactor(interop): merge entity resolvers into axis buildEntity"
```

---

### Task 6: Tiered matrix build + HTTP handler

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Add new DTOs**

```go
type InteropMatrixTier string // "group" | "leaf"

type InteropMatrixGroup struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	ChildCount  int     `json:"child_count"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	Subtitle    *string `json:"subtitle,omitempty"`
}

type InteropMatrixLeaf struct {
	InteropMatrixEntity
	ParentID string `json:"parent_id,omitempty"`
}

type InteropMatrixCell struct {
	RowID          string            `json:"row_id"`
	ColumnID       string            `json:"column_id"`
	RowTier        InteropMatrixTier `json:"row_tier"`
	ColumnTier     InteropMatrixTier `json:"column_tier"`
	// ... existing metric fields
}

type InteropAxis struct {
	HubCollection string `json:"hub_collection"`
	PathBased     bool   `json:"path_based"`
	Tiered        bool   `json:"tiered"`
}

type InteropMatrixResponse struct {
	Row           InteropAxis            `json:"row"`
	Column        InteropAxis            `json:"column"`
	RowGroups     []InteropMatrixGroup   `json:"row_groups"`
	RowLeaves     []InteropMatrixLeaf    `json:"row_leaves"`
	ColumnGroups  []InteropMatrixGroup   `json:"column_groups"`
	ColumnLeaves  []InteropMatrixLeaf    `json:"column_leaves"`
	Cells         []InteropMatrixCell    `json:"cells"`
}
```

- [ ] **Step 2: Write failing `buildInteropMatrix` tiered test**

Update `TestBuildInteropMatrix_CartesianAndSums` to use tier coords (leaf×leaf only) and assert `RowLeaves` / `ColumnLeaves` instead of `Rows`/`Columns`.

- [ ] **Step 3: Refactor pipeline**

1. `scanInteropCacheRecords` → produce `[]interopTieredCacheInput` via `resolveAxisCoords` per record.
2. `aggregateInteropCells` → convert accumulators to `[]InteropMatrixCell` with tiers.
3. Collect unique group/leaf IDs per axis; load entities:
   - Groups: `buildEntity` with `cacheRecord=nil` (or group-specific loader).
   - Leaves: first referencing cache record per leaf ID.
4. Compute `child_count` per group from leaf list.
5. Sort groups/leaves for stable UI (name order).

Remove: `resolveWalletVersionLabels`, `loadInteropEntitiesByIDs` resolver path, flat `readAxisIDs` for aggregation (keep path inline entities for conformance leaves).

- [ ] **Step 4: Update HTTP tests** (`TestScoreboardInterop_*`) for new JSON shape.

- [ ] **Step 5: Run full handler package tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'Interop|interop' -count=1`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(interop): tiered matrix response and handler"
```

---

### Task 7: OpenAPI

**Files:**
- Modify: `docs/public/API/openapi.yml`

- [ ] **Step 1: Add schemas** `HandlersInteropMatrixGroup`, `HandlersInteropMatrixLeaf`, extend `HandlersInteropAxis` with `tiered`, `HandlersInteropMatrixCell` with `row_tier`/`column_tier`.

- [ ] **Step 2: Update `HandlersInteropMatrixResponse`** — replace `rows`/`columns` with four tier arrays.

- [ ] **Step 3: Regenerate or verify OpenAPI** if project uses `make generate` for spec — run `make generate` only if Makefile documents OpenAPI generation from Go; otherwise hand-edit.

- [ ] **Step 4: Commit**

```bash
git add docs/public/API/openapi.yml
git commit -m "docs(api): tiered interop matrix OpenAPI schemas"
```

---

### Task 8: Frontend types + conformance load

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/types.ts`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.ts`
- Delete: `webapp/src/lib/scoreboard/interop/interop-entity-data.ts` (update imports to `$lib/global/entities` in `+page.svelte` / `to-view-matrix.ts`)

- [ ] **Step 1: Update TypeScript types** to match new API (breaking).

- [ ] **Step 2: Load standards in `+page.ts`**

```typescript
import { getStandardsWithTestSuites } from '$lib/standards';

const conformanceChecks = await getStandardsWithTestSuites({ fetch, forPipeline: true });
if (conformanceChecks instanceof Error) {
	error(500, 'Failed to load conformance standards');
}
return { matrix, row, column, standards: conformanceChecks };
```

- [ ] **Step 3: Fix imports** after deleting `interop-entity-data.ts` — use `EntityData` from `$lib/global/entities` keyed by `InteropHubCollection`.

- [ ] **Step 4: Run typecheck**

Run: `cd webapp && bun run check`  
Expected: errors in `to-view-matrix` / `matrix-grid` until Task 9 — note remaining failures.

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/types.ts webapp/src/routes/(public)/scoreboard/interop/+page.ts webapp/src/lib/scoreboard/interop/
git commit -m "feat(interop): tiered matrix types and load conformance standards"
```

---

### Task 9: View model + grid expand UI

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/to-view-matrix.ts`
- Modify: `webapp/src/lib/scoreboard/interop/to-view-matrix.test.ts`
- Modify: `webapp/src/lib/scoreboard/interop/matrix-grid.svelte`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.svelte`
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: Write failing Vitest tests**

```typescript
it('shows group rows when collapsed', () => { /* ... */ });
it('shows leaf rows when group expanded', () => { /* ... */ });
it('resolves cell with tier-aware key', () => { /* ... */ });
it('uses undefined version i18n when version_label is null', () => { /* ... */ });
```

- [ ] **Step 2: Run — expect FAIL**

Run: `cd webapp && bun run test:unit -- --run src/lib/scoreboard/interop/to-view-matrix.test.ts`

- [ ] **Step 3: Implement `buildVisibleMatrix` in `to-view-matrix.ts`**

- Inputs: `InteropMatrixResponse`, `standards`, `expandedRowGroups`, `expandedColumnGroups`.
- Visible row: if `row.tiered` && group expanded → leaves with `parent_id`; else → `row_groups`.
- Cell key: `` `${rowTier}:${rowId}:${colTier}:${colId}` `` from visible row/col context.
- `displaySubtitle`: `subtitleOrVersion(subtitle, version_label)`; if `version_label === null` on wallet leaf → `m.interop_matrix_version_undefined()`.

- [ ] **Step 4: Update `matrix-grid.svelte`**

- `$state` expand sets (or bind from page).
- Group header: chevron button, `(child_count)` suffix.
- Pass `rowTier`/`colTier` into cell lookup.

- [ ] **Step 5: Wire `+page.svelte`** — pass `data.standards`, manage expand sets.

- [ ] **Step 6: Add i18n keys** — `interop_matrix_version_undefined`, `interop_matrix_expand_group`, `interop_matrix_collapse_group`.

- [ ] **Step 7: Run tests + check**

Run: `cd webapp && bun run test:unit -- --run src/lib/scoreboard/interop/`  
Run: `cd webapp && bun run check`  
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/ webapp/src/routes/(public)/scoreboard/interop/ webapp/messages/
git commit -m "feat(interop): tiered matrix expand UI and view model"
```

---

### Task 10: Manual verification + lint

- [ ] **Step 1: Run Go unit suite**

Run: `make test` or `go test -tags=unit ./pkg/internal/apis/handlers/...`

- [ ] **Step 2: Run webapp lint**

Run: `cd webapp && bun run lint`

- [ ] **Step 3: Manual checklist (dev stack running)**

- [ ] `/scoreboard/interop?row=wallets&column=credential_issuers` — collapsed group×group; expand wallet → version rows; expand issuer → credential columns.
- [ ] Wallet with cache-only wallet (no version) — group cell + expanded `<undefined>` version row.
- [ ] `row=wallets&column=conformance-checks` — suite group columns; expand to check paths; subtitles from standards.
- [ ] All six featured tabs load without 500.

- [ ] **Step 4: Commit any fixups**

```bash
git commit -m "chore(interop): tiered matrix verification fixups"
```

---

## Spec coverage checklist (self-review)

| Spec requirement | Task |
|------------------|------|
| Per-axis tier profiles | Task 3–4 |
| Existential-once vs Cartesian | Task 1 |
| Inclusive wallet + synthetic leaf | Task 4 |
| Unified `buildEntity` registry | Task 5 |
| Conformance suite grouping (D2) | Task 2, 4 |
| Breaking API shape | Task 6–7 |
| Frontend expand + SSR standards | Task 8–9 |
| OpenAPI | Task 7 |
| Delete resolver file | Task 5 |
| i18n undefined version | Task 9 |

## Out of scope (confirmed)

- `/axes` endpoint, hub padding, axis swap — not in tasks.
- `scoreboard.go` cache write deduplication — not in tasks.
