# Scoreboard Interop Axes-Only Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace interop **mode** enums with a **hub-collection axis registry** on the API (`?row=&column=`) and curated featured-pair tabs on the frontend, using `EntityData` for labels.

**Architecture:** Backend: new `scoreboard_interop_axes.go` registry (`hub_collection`, `cache_field`, `path_based`); handler validates hub pair and rejects legacy `mode` param; scan/load generalizes path-based entities to either row or column side. Frontend: `interop-hub-collections.ts` + `interop-entity-data.ts` + `featured-pairs.ts`; delete `modes.ts`/`axes.ts`; page redirects bare URL to default pair.

**Tech Stack:** Go 1.24, PocketBase handlers/tests (`stretchr/testify`), SvelteKit 2 + Svelte 5, TypeScript, Paraglide i18n, Vitest.

**Spec:** `docs/superpowers/specs/2026-05-29-scoreboard-interop-axes-only-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/scoreboard_interop_axes.go` | **NEW** — axis registry, lookup helpers, usage hint |
| `pkg/internal/apis/handlers/scoreboard_interop.go` | Remove mode types/config; axis-pair handler; generalized scan/load; `buildInteropMatrix` takes axes |
| `pkg/internal/apis/handlers/scoreboard_interop_test.go` | Replace mode tests with axis/pair tests; update HTTP URLs |
| `webapp/src/lib/scoreboard/interop/interop-hub-collections.ts` | **NEW** — `INTEROP_HUB_COLLECTIONS`, `InteropHubCollection`, guard |
| `webapp/src/lib/scoreboard/interop/interop-entity-data.ts` | **NEW** — `Record<InteropHubCollection, EntityData>` |
| `webapp/src/lib/scoreboard/interop/featured-pairs.ts` | **NEW** — six curated pairs, default pair |
| `webapp/src/lib/scoreboard/interop/types.ts` | Drop `InteropMode` / `mode`; axis = `{ hub_collection, path_based }` |
| `webapp/src/lib/scoreboard/interop/to-view-matrix.ts` | Corner label from `interopEntityData`; drop `axisLabel` option |
| `webapp/src/lib/scoreboard/interop/matrix-grid.svelte` | Use entity data for corner label |
| `webapp/src/routes/(public)/scoreboard/interop/+page.ts` | Redirect + hub-pair fetch |
| `webapp/src/routes/(public)/scoreboard/interop/+page.svelte` | Featured pair tabs with EntityData labels |
| `webapp/messages/en.json` | Remove unused `interop_mode_*` keys (keep `interop_matrix_corner_label`) |
| **Delete** | `modes.ts`, `modes.test.ts`, `axes.ts`, `axes.test.ts` |

---

### Task 1: Backend axis registry

**Files:**
- Create: `pkg/internal/apis/handlers/scoreboard_interop_axes.go`
- Create: `pkg/internal/apis/handlers/scoreboard_interop_axes_test.go`

- [ ] **Step 1: Write failing axis registry tests**

```go
// scoreboard_interop_axes_test.go
package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInteropAxis_KnownHubs(t *testing.T) {
	t.Parallel()

	wallets, ok := getInteropAxis("wallets")
	require.True(t, ok)
	require.Equal(t, "wallets", wallets.HubCollection)
	require.Equal(t, "wallets", wallets.CacheField)
	require.False(t, wallets.PathBased)

	issuers, ok := getInteropAxis("credential_issuers")
	require.True(t, ok)
	require.Equal(t, "issuers", issuers.CacheField)
	require.False(t, issuers.PathBased)

	conformance, ok := getInteropAxis("conformance-checks")
	require.True(t, ok)
	require.Equal(t, "conformance_checks", conformance.CacheField)
	require.True(t, conformance.PathBased)
}

func TestGetInteropAxis_Unknown(t *testing.T) {
	t.Parallel()
	_, ok := getInteropAxis("bad_hub")
	require.False(t, ok)
}

func TestSupportedInteropHubCollections(t *testing.T) {
	t.Parallel()
	got := supportedInteropHubCollections()
	require.Len(t, got, 6)
	require.Contains(t, got, "wallets")
	require.Contains(t, got, "conformance-checks")
}

func TestInteropHubsUsageHint(t *testing.T) {
	t.Parallel()
	require.Contains(t, interopHubsUsageHint(), "row=")
	require.Contains(t, interopHubsUsageHint(), "wallets")
}
```

- [ ] **Step 2: Run tests — expect compile fail**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestGetInteropAxis|TestSupportedInteropHubCollections|TestInteropHubsUsageHint' -v`  
Expected: FAIL — undefined `getInteropAxis`, `supportedInteropHubCollections`, `interopHubsUsageHint`.

- [ ] **Step 3: Implement axis registry**

```go
// scoreboard_interop_axes.go
package handlers

import "sort"

type interopAxis struct {
	HubCollection string
	CacheField    string
	PathBased     bool
}

var interopAxisRegistry = map[string]interopAxis{
	"wallets": {
		HubCollection: "wallets",
		CacheField:    "wallets",
		PathBased:     false,
	},
	"credential_issuers": {
		HubCollection: "credential_issuers",
		CacheField:    "issuers",
		PathBased:     false,
	},
	"credentials": {
		HubCollection: "credentials",
		CacheField:    "credentials",
		PathBased:     false,
	},
	"verifiers": {
		HubCollection: "verifiers",
		CacheField:    "verifiers",
		PathBased:     false,
	},
	"use_cases_verifications": {
		HubCollection: "use_cases_verifications",
		CacheField:    "use_case_verifications",
		PathBased:     false,
	},
	"conformance-checks": {
		HubCollection: "conformance-checks",
		CacheField:    "conformance_checks",
		PathBased:     true,
	},
}

func getInteropAxis(hubCollection string) (interopAxis, bool) {
	axis, ok := interopAxisRegistry[hubCollection]
	return axis, ok
}

func supportedInteropHubCollections() []string {
	hubs := make([]string, 0, len(interopAxisRegistry))
	for hub := range interopAxisRegistry {
		hubs = append(hubs, hub)
	}
	sort.Strings(hubs)
	return hubs
}

func interopHubsUsageHint() string {
	return "use row= and column= with hub collections: " +
		joinStrings(supportedInteropHubCollections(), ", ")
}

func joinStrings(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	out := items[0]
	for i := 1; i < len(items); i++ {
		out += sep + items[i]
	}
	return out
}
```

Note: use `strings.Join` instead of custom `joinStrings` if you prefer stdlib — delete helper if redundant.

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestGetInteropAxis|TestSupportedInteropHubCollections|TestInteropHubsUsageHint' -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_axes.go pkg/internal/apis/handlers/scoreboard_interop_axes_test.go
git commit -m "feat(interop): add hub-collection axis registry"
```

---

### Task 2: Generalize scan/load and matrix builder

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Extend `interopCacheScan` for path-based rows**

In `scoreboard_interop.go`, add `rowEntities map[string]InteropMatrixEntity` alongside existing `columnEntities`.

- [ ] **Step 2: Replace `scanInteropCacheRecords` signature and implementation**

```go
func readAxisIDs(record *core.Record, axis interopAxis) ([]string, map[string]InteropMatrixEntity) {
	inline := map[string]InteropMatrixEntity{}
	if !axis.PathBased {
		ids := record.GetStringSlice(axis.CacheField)
		return ids, inline
	}
	var rawIDs []string
	if err := record.UnmarshalJSONField(axis.CacheField, &rawIDs); err != nil {
		return nil, inline
	}
	out := make([]string, 0, len(rawIDs))
	for _, id := range rawIDs {
		if id == "" {
			continue
		}
		out = append(out, id)
		if _, ok := inline[id]; ok {
			continue
		}
		inline[id] = InteropMatrixEntity{
			ID:   id,
			Name: conformanceCheckName(id),
			Path: id,
		}
	}
	return out, inline
}

func scanInteropCacheRecords(
	records []*core.Record,
	rowAxis interopAxis,
	colAxis interopAxis,
) interopCacheScan {
	scan := interopCacheScan{
		rowIDs:              map[string]struct{}{},
		columnIDs:           map[string]struct{}{},
		rowEntities:         map[string]InteropMatrixEntity{},
		columnEntities:      map[string]InteropMatrixEntity{},
		walletVersionLabels: map[string]*string{},
	}
	for _, record := range records {
		rowIDs, rowInline := readAxisIDs(record, rowAxis)
		colIDs, colInline := readAxisIDs(record, colAxis)
		for id, entity := range rowInline {
			scan.rowEntities[id] = entity
		}
		for id, entity := range colInline {
			scan.columnEntities[id] = entity
		}
		scan.inputs = append(scan.inputs, interopCacheInput{
			PipelineID:     record.GetString("pipeline"),
			TotalRuns:      record.GetInt("total_runs"),
			TotalSuccesses: record.GetInt("total_successes"),
			RowIDs:         rowIDs,
			ColumnIDs:      colIDs,
		})
		for _, rowID := range rowIDs {
			scan.rowIDs[rowID] = struct{}{}
		}
		for _, colID := range colIDs {
			if !colAxis.PathBased {
				scan.columnIDs[colID] = struct{}{}
			}
		}
		if !rowAxis.PathBased {
			for _, rowID := range rowIDs {
				scan.rowIDs[rowID] = struct{}{}
			}
		}
	}
	return scan
}
```

Fix duplicate rowID loop — only track `scan.rowIDs` for non-path-based rows (mirror column logic):

```go
for _, rowID := range rowIDs {
	if !rowAxis.PathBased {
		scan.rowIDs[rowID] = struct{}{}
	}
}
```

- [ ] **Step 3: Update `buildInteropMatrix` to accept axes (remove mode/key defaults)**

```go
func buildInteropMatrix(
	inputs []interopCacheInput,
	rowAxis InteropAxis,
	colAxis InteropAxis,
	rowEntities map[string]InteropMatrixEntity,
	columnEntities map[string]InteropMatrixEntity,
) InteropMatrixResponse {
	// ... existing cell aggregation unchanged ...
	return InteropMatrixResponse{
		Row:     rowAxis,
		Column:  colAxis,
		Rows:    sortedInteropEntities(rowEntities, rowSeen),
		Columns: sortedInteropEntities(columnEntities, colSeen),
		Cells:   cells,
	}
}
```

Remove `Mode` field from `InteropMatrixResponse` struct and delete `Key` from `InteropAxis` if still present.

- [ ] **Step 4: Update `loadInteropMatrixFromCache`**

```go
func loadInteropMatrixFromCache(app core.App, rowAxis interopAxis, colAxis interopAxis) (InteropMatrixResponse, error) {
	// ... find collection, list records ...
	scan := scanInteropCacheRecords(records, rowAxis, colAxis)

	walletVersionsByID, err := loadWalletVersionsForCacheRecords(app, records)
	// ...
	if rowAxis.HubCollection == "wallets" {
		resolveWalletVersionLabels(scan, records, rowAxis, walletVersionsByID)
	}

	rowEntities, err := loadInteropEntitiesByIDs(app, rowAxis.HubCollection, scan.rowIDs, scan.walletVersionLabels)
	// merge scan.rowEntities into rowEntities for path-based side
	for id, entity := range scan.rowEntities {
		rowEntities[id] = entity
	}

	columnEntities := scan.columnEntities
	if !colAxis.PathBased {
		columnEntities, err = loadInteropEntitiesByIDs(app, colAxis.HubCollection, scan.columnIDs, nil)
		// ...
	}

	resp := buildInteropMatrix(
		scan.inputs,
		InteropAxis{HubCollection: rowAxis.HubCollection, PathBased: rowAxis.PathBased},
		InteropAxis{HubCollection: colAxis.HubCollection, PathBased: colAxis.PathBased},
		rowEntities,
		columnEntities,
	)
	return resp, nil
}
```

Update `resolveWalletVersionLabels` to take `interopAxis` and use `axis.CacheField` instead of `modeConfig.RowRelationField`.

- [ ] **Step 5: Update matrix builder unit test**

In `TestBuildInteropMatrix_CartesianAndSums`, change call to:

```go
got := buildInteropMatrix(
	inputs,
	InteropAxis{HubCollection: "wallets", PathBased: false},
	InteropAxis{HubCollection: "credential_issuers", PathBased: false},
	rowEntities,
	colEntities,
)
require.Equal(t, "wallets", got.Row.HubCollection)
require.Equal(t, "credential_issuers", got.Column.HubCollection)
```

Remove assertions on `got.Mode`, `got.Row.Key`, `got.Column.Key`.

- [ ] **Step 6: Run tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestBuildInteropMatrix' -v`  
Expected: PASS (fix compile errors from removed mode types incrementally).

- [ ] **Step 7: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "refactor(interop): generalize scan and matrix builder for axis pairs"
```

---

### Task 3: HTTP handler — row/column query params

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing handler validation tests**

Replace `TestInteropModeValidation`, `TestSupportedInteropModeStrings`, `TestInteropModeConfigRelations`, and `TestHandleInteropMatrix_ModeValidationReturnsBadRequest` with:

```go
func TestHandleInteropMatrix_PairValidationReturnsBadRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		url  string
	}{
		{name: "missing row", url: "/api/scoreboard/interop?column=credentials"},
		{name: "missing column", url: "/api/scoreboard/interop?row=wallets"},
		{name: "unknown row", url: "/api/scoreboard/interop?row=bad&column=credentials"},
		{name: "equal axes", url: "/api/scoreboard/interop?row=wallets&column=wallets"},
		{name: "legacy mode param", url: "/api/scoreboard/interop?mode=wallets_credentials"},
		{name: "mode with row column", url: "/api/scoreboard/interop?mode=x&row=wallets&column=credentials"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()
			err = HandleInteropMatrix()(&core.RequestEvent{
				App: app,
				Event: router.Event{Request: req, Response: rec},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}
```

- [ ] **Step 2: Run test — expect FAIL** (handler still uses `mode`)

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestHandleInteropMatrix_PairValidation -v`

- [ ] **Step 3: Implement handler and route metadata**

Remove all `interopMode*` types, constants, `interopModeConfigs`, and mode helpers from `scoreboard_interop.go`.

Update `InteropMatrixResponse` — delete `Mode` field. Update `InteropAxis` — delete `Key` field.

```go
func HandleInteropMatrix() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		q := e.Request.URL.Query()
		if q.Has("mode") {
			return apierror.New(
				http.StatusBadRequest,
				"mode",
				"mode query param is no longer supported",
				interopHubsUsageHint(),
			).JSON(e)
		}
		rowHub := q.Get("row")
		colHub := q.Get("column")
		if rowHub == "" || colHub == "" {
			return apierror.New(
				http.StatusBadRequest,
				"row",
				"missing row or column hub collection",
				interopHubsUsageHint(),
			).JSON(e)
		}
		if rowHub == colHub {
			return apierror.New(
				http.StatusBadRequest,
				"row",
				"row and column must differ",
				interopHubsUsageHint(),
			).JSON(e)
		}
		rowAxis, ok := getInteropAxis(rowHub)
		if !ok {
			return apierror.New(http.StatusBadRequest, "row", "unknown row hub collection", interopHubsUsageHint()).JSON(e)
		}
		colAxis, ok := getInteropAxis(colHub)
		if !ok {
			return apierror.New(http.StatusBadRequest, "column", "unknown column hub collection", interopHubsUsageHint()).JSON(e)
		}

		resp, err := loadInteropMatrixFromCache(e.App, rowAxis, colAxis)
		// ... existing error handling without unsupportedInteropModeError ...
		return e.JSON(http.StatusOK, resp)
	}
}
```

Update `ScoreboardInteropPublicRoutes` query attributes to required `row` and `column` (remove `mode`). Delete `unsupportedInteropModeError`.

- [ ] **Step 4: Update all integration test URLs**

Global replace in `scoreboard_interop_test.go`:

| Old | New |
|-----|-----|
| `?mode=wallets_credentials` | `?row=wallets&column=credentials` |
| `?mode=wallets_issuers` | `?row=wallets&column=credential_issuers` |
| `?mode=wallets_verifiers` | `?row=wallets&column=verifiers` |
| `?mode=wallets_use_case_verifications` | `?row=wallets&column=use_cases_verifications` |
| `?mode=wallets_conformance_checks` | `?row=wallets&column=conformance-checks` |
| `?mode=use_case_verifications_conformance_checks` | `?row=use_cases_verifications&column=conformance-checks` |

Remove assertions on `resp.Mode`, `resp.Row.Key`, `resp.Column.Key`.

- [ ] **Step 5: Add path-based row test**

```go
func TestHandleInteropMatrix_ConformanceChecksAsRow(t *testing.T) {
	// cache with conformance_checks JSON + wallets relation
	// GET ?row=conformance-checks&column=wallets
	// assert 200, resp.Row.PathBased == true, cell present
}
```

- [ ] **Step 6: Run full handler test package**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'Interop|HandleInteropMatrix|BuildInterop' -v`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(interop): replace mode query param with row/column hub pairs"
```

---

### Task 4: Frontend hub collections and entity data

**Files:**
- Create: `webapp/src/lib/scoreboard/interop/interop-hub-collections.ts`
- Create: `webapp/src/lib/scoreboard/interop/interop-hub-collections.test.ts`
- Create: `webapp/src/lib/scoreboard/interop/interop-entity-data.ts`
- Create: `webapp/src/lib/scoreboard/interop/interop-entity-data.test.ts`
- Create: `webapp/src/lib/scoreboard/interop/featured-pairs.ts`

- [ ] **Step 1: Write failing hub collection tests**

```ts
// interop-hub-collections.test.ts
import { describe, expect, it } from 'vitest';
import { INTEROP_HUB_COLLECTIONS, isInteropHubCollection } from './interop-hub-collections';

describe('interop hub collections', () => {
	it('lists six unique hub collections', () => {
		expect(new Set(INTEROP_HUB_COLLECTIONS).size).toBe(6);
		expect(INTEROP_HUB_COLLECTIONS).toHaveLength(6);
	});

	it('guards known hubs', () => {
		expect(isInteropHubCollection('wallets')).toBe(true);
		expect(isInteropHubCollection('conformance-checks')).toBe(true);
		expect(isInteropHubCollection('bad')).toBe(false);
	});
});
```

- [ ] **Step 2: Implement hub collections**

```ts
// interop-hub-collections.ts
export const INTEROP_HUB_COLLECTIONS = [
	'wallets',
	'credential_issuers',
	'credentials',
	'verifiers',
	'use_cases_verifications',
	'conformance-checks'
] as const;

export type InteropHubCollection = (typeof INTEROP_HUB_COLLECTIONS)[number];

export function isInteropHubCollection(value: string): value is InteropHubCollection {
	return (INTEROP_HUB_COLLECTIONS as readonly string[]).includes(value);
}
```

- [ ] **Step 3: Write failing entity data test**

```ts
// interop-entity-data.test.ts
import { describe, expect, it } from 'vitest';
import { INTEROP_HUB_COLLECTIONS } from './interop-hub-collections';
import { interopEntityData } from './interop-entity-data';

describe('interopEntityData', () => {
	it('covers every hub collection', () => {
		for (const hub of INTEROP_HUB_COLLECTIONS) {
			expect(interopEntityData[hub].labels.singular.length).toBeGreaterThan(0);
		}
	});
});
```

- [ ] **Step 4: Implement entity data and featured pairs**

```ts
// interop-entity-data.ts
import { entities, type EntityData } from '$lib/global/entities';
import type { InteropHubCollection } from './interop-hub-collections';

export const interopEntityData: Record<InteropHubCollection, EntityData> = {
	wallets: entities.wallets,
	credential_issuers: entities.credential_issuers,
	credentials: entities.credentials,
	verifiers: entities.verifiers,
	use_cases_verifications: entities.use_cases_verifications,
	'conformance-checks': entities.conformance_checks
};
```

```ts
// featured-pairs.ts
import type { InteropHubCollection } from './interop-hub-collections';

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

- [ ] **Step 5: Run webapp unit tests**

Run: `cd webapp && bun run test:unit -- --run src/lib/scoreboard/interop/interop-hub-collections.test.ts src/lib/scoreboard/interop/interop-entity-data.test.ts`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/interop-hub-collections.ts webapp/src/lib/scoreboard/interop/interop-hub-collections.test.ts webapp/src/lib/scoreboard/interop/interop-entity-data.ts webapp/src/lib/scoreboard/interop/interop-entity-data.test.ts webapp/src/lib/scoreboard/interop/featured-pairs.ts
git commit -m "feat(interop): add hub collections, entity data map, and featured pairs"
```

---

### Task 5: Frontend types and view matrix

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/types.ts`
- Modify: `webapp/src/lib/scoreboard/interop/to-view-matrix.ts`
- Modify: `webapp/src/lib/scoreboard/interop/to-view-matrix.test.ts`
- Modify: `webapp/src/lib/scoreboard/interop/matrix-grid.svelte`

- [ ] **Step 1: Update types**

```ts
// types.ts — remove InteropMode import/export and mode field
export type InteropAxis = {
	hub_collection: string;
	path_based: boolean;
};

export type InteropMatrixResponse = {
	row: InteropAxis;
	column: InteropAxis;
	rows: InteropMatrixEntity[];
	columns: InteropMatrixEntity[];
	cells: InteropMatrixCell[];
};
```

- [ ] **Step 2: Update `to-view-matrix.ts`**

```ts
import { interopEntityData } from './interop-entity-data';
import type { InteropHubCollection } from './interop-hub-collections';
import { m } from '@/i18n';

export type ToViewMatrixOptions = {
	standards: readonly Standard[];
};

function hubLabel(hub: string, plural: boolean): string {
	if (!(hub in interopEntityData)) return hub;
	const data = interopEntityData[hub as InteropHubCollection];
	return plural ? (data.labels.plural ?? data.labels.singular) : data.labels.singular;
}

export function toViewMatrix(
	response: InteropMatrixResponse,
	{ standards }: ToViewMatrixOptions
): ViewMatrix {
	const rowLabel = hubLabel(response.row.hub_collection, false);
	const columnLabel = hubLabel(response.column.hub_collection, false);

	return {
		cornerLabel: m.interop_matrix_corner_label({ row: rowLabel, column: columnLabel }),
		// ... rest unchanged ...
	};
}
```

- [ ] **Step 3: Update tests**

Remove `mode` and `key` from fixtures; drop `axisLabel` / `cornerLabel` from `viewOptions`:

```ts
const viewOptions = { standards };
// minimalMatrix without mode/key on axes
expect(view.cornerLabel).toContain('Wallet'); // or match m.interop_matrix_corner_label output
```

Update `hubHref` tests to axes without `key`.

- [ ] **Step 4: Update `matrix-grid.svelte`**

```ts
const view = $derived(
	toViewMatrix(matrix, {
		standards: getConformanceStore().standards
	})
);
```

Remove `interopAxisLabel` import.

- [ ] **Step 5: Run tests and check**

Run: `cd webapp && bun run test:unit -- --run src/lib/scoreboard/interop/to-view-matrix.test.ts && bun run check`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/types.ts webapp/src/lib/scoreboard/interop/to-view-matrix.ts webapp/src/lib/scoreboard/interop/to-view-matrix.test.ts webapp/src/lib/scoreboard/interop/matrix-grid.svelte
git commit -m "refactor(interop): derive matrix labels from interopEntityData"
```

---

### Task 6: Page load, tabs, cleanup

**Files:**
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.ts`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.svelte`
- Delete: `webapp/src/lib/scoreboard/interop/modes.ts`, `modes.test.ts`, `axes.ts`, `axes.test.ts`
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: Update `+page.ts`**

```ts
import { redirect } from '@sveltejs/kit';
import { DEFAULT_INTEROP_PAIR } from '$lib/scoreboard/interop/featured-pairs';
import { isInteropHubCollection } from '$lib/scoreboard/interop/interop-hub-collections';
import type { InteropMatrixResponse } from '$lib/scoreboard/interop/types';
import { error } from '@sveltejs/kit';

export const load = async ({ fetch, url }) => {
	const row = url.searchParams.get('row');
	const column = url.searchParams.get('column');

	if (!row || !column) {
		redirect(302, `/scoreboard/interop?row=${DEFAULT_INTEROP_PAIR.row}&column=${DEFAULT_INTEROP_PAIR.column}`);
	}
	if (!isInteropHubCollection(row) || !isInteropHubCollection(column)) {
		error(400, 'Invalid interoperability matrix axes');
	}

	const res = await fetch(`/api/scoreboard/interop?row=${row}&column=${column}`);
	if (!res.ok) {
		error(res.status, 'Failed to load interoperability matrix');
	}
	const matrix = (await res.json()) as InteropMatrixResponse;
	return { matrix, row, column };
};
```

- [ ] **Step 2: Update `+page.svelte`**

```svelte
<script lang="ts">
	import { FEATURED_INTEROP_PAIRS } from '$lib/scoreboard/interop/featured-pairs';
	import { interopEntityData } from '$lib/scoreboard/interop/interop-entity-data';
	// ...

	function pairLabel(row: string, column: string): string {
		const rowData = interopEntityData[row as keyof typeof interopEntityData];
		const colData = interopEntityData[column as keyof typeof interopEntityData];
		const rowLabel = rowData.labels.plural ?? rowData.labels.singular;
		const colLabel = colData.labels.plural ?? colData.labels.singular;
		return `${rowLabel} × ${colLabel}`;
	}
</script>

{#each FEATURED_INTEROP_PAIRS as pair (pair.row + pair.column)}
	<a
		href={resolve(localizeHref(`/scoreboard/interop?row=${pair.row}&column=${pair.column}`) as '/')}
		aria-current={data.row === pair.row && data.column === pair.column ? 'page' : undefined}
	>
		{pairLabel(pair.row, pair.column)}
	</a>
{/each}
```

Use typed `InteropHubCollection` in `pairLabel` params instead of `keyof typeof` if preferred.

- [ ] **Step 3: Delete obsolete files and i18n keys**

Remove from `webapp/messages/en.json`:

- `interop_mode_wallets_credentials`
- `interop_mode_wallets_issuers`
- `interop_mode_wallets_verifiers`
- `interop_mode_wallets_use_case_verifications`
- `interop_mode_wallets_conformance_checks`
- `interop_mode_use_case_verifications_conformance_checks`

Run Paraglide compile if the project requires it after message changes (`cd webapp && bun run predev` or project’s i18n script).

- [ ] **Step 4: Grep for stale imports**

Run: `rg "interop/modes|interop/axes|InteropMode|interop_mode_" webapp`  
Expected: no matches outside docs.

- [ ] **Step 5: Final verification**

Run:
```bash
go test -tags=unit ./pkg/internal/apis/handlers/ -run 'Interop|HandleInteropMatrix|BuildInterop' -v
cd webapp && bun run test:unit -- --run src/lib/scoreboard/interop && bun run check
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add webapp/src/routes/(public)/scoreboard/interop/ webapp/src/lib/scoreboard/interop/ webapp/messages/en.json
git rm webapp/src/lib/scoreboard/interop/modes.ts webapp/src/lib/scoreboard/interop/modes.test.ts webapp/src/lib/scoreboard/interop/axes.ts webapp/src/lib/scoreboard/interop/axes.test.ts
git commit -m "feat(interop): switch page to hub-pair URLs and EntityData tab labels"
```

---

### Task 7: Docs touch-up (optional, same PR)

**Files:**
- Modify: `docs/src/content/docs/software-architecture/scoreboard.md` (if it references `?mode=`)

- [ ] **Step 1: Search and update API examples**

Run: `rg 'scoreboard/interop\\?mode' docs`  
Replace with `?row=wallets&column=credentials` examples.

- [ ] **Step 2: Commit**

```bash
git add docs/
git commit -m "docs: update interop matrix API to row/column hub params"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Axis registry with cache_field | Task 1 |
| Open API any valid pair | Task 3 |
| Reject `mode` param | Task 3 |
| Required row+column | Task 3 |
| path_based on row or column | Task 2, 3 |
| Drop `mode` from JSON | Task 2, 3 |
| Featured six tabs | Task 4, 6 |
| Bare URL redirect | Task 6 |
| InteropHubCollection type file | Task 4 |
| Record hub → EntityData | Task 4 |
| Remove modes.ts / axes.ts | Task 6 |
| i18n cleanup | Task 6 |

## Success criteria (from spec)

- [ ] `GET /api/scoreboard/interop?row=wallets&column=credentials` — no `mode` in response
- [ ] `?mode=` → 400
- [ ] Missing row/column → 400
- [ ] `/scoreboard/interop` → redirect to default pair
- [ ] Six tabs with EntityData labels
- [ ] `row=conformance-checks&column=wallets` works
- [ ] `go test -tags=unit` + `bun run check` pass
