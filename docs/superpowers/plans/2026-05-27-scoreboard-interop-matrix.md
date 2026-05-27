# Scoreboard Interop Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a public Wallet×Issuer interoperability matrix at `/scoreboard/interop`, backed by `GET /api/scoreboard/interop?mode=wallets_issuers` aggregating `pipeline_scoreboard_cache` with run-weighted cell metrics.

**Architecture:** Pure Go matrix builder (unit-tested) loads cache rows, Cartesian-attributes pipelines to wallet×issuer cells, sums runs/successes, computes `success_rate` and `status`. Thin SvelteKit page fetches JSON and renders a sticky grid with status colors. Tracer bullet: backend first, then minimal UI, then polish.

**Tech Stack:** Go 1.24 + PocketBase test app (`stretchr/testify`), existing `routing.RouteGroup` + `apierror`, Svelte 5 / SvelteKit, Tailwind, Paraglide (`webapp/messages/en.json`).

**Spec:** `docs/superpowers/specs/2026-05-27-scoreboard-interop-matrix-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/scoreboard_interop.go` | DTOs, `buildInteropMatrix`, metadata resolution, HTTP handler, route group |
| `pkg/internal/apis/handlers/scoreboard_interop_test.go` | Unit tests for builder + handler HTTP |
| `pkg/internal/apis/RoutesRegistry.go` | Register `ScoreboardInteropPublicRoutes` in `RouteGroups` |
| `webapp/src/lib/scoreboard/interop/types.ts` | TS types mirroring API JSON |
| `webapp/src/lib/scoreboard/interop/status.ts` | `status` → Tailwind background/text classes |
| `webapp/src/lib/scoreboard/interop/matrix-cell.svelte` | Single cell (empty vs filled) |
| `webapp/src/lib/scoreboard/interop/matrix-grid.svelte` | Grid layout, sticky headers, legend slot |
| `webapp/src/routes/(public)/scoreboard/interop/+page.ts` | SSR `fetch` to API |
| `webapp/src/routes/(public)/scoreboard/interop/+page.svelte` | Page shell, footnote, legend |
| `webapp/src/routes/(public)/scoreboard/+page.svelte` | Link to interop matrix |
| `webapp/messages/en.json` (+ sync other locales or leave EN-only keys with EN fallback) | i18n strings |

---

### Task 1: Core matrix types and status bands

**Files:**
- Create: `pkg/internal/apis/handlers/scoreboard_interop.go` (first half)
- Create: `pkg/internal/apis/handlers/scoreboard_interop_test.go` (status tests)

- [ ] **Step 1: Write failing status tests**

```go
// pkg/internal/apis/handlers/scoreboard_interop_test.go
func TestInteropStatusFromRate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		rate   float64
		status interopStatus
	}{
		{90, interopStatusStable},
		{89.9, interopStatusFlaky},
		{70, interopStatusFlaky},
		{69.9, interopStatusFailing},
		{50, interopStatusFailing},
		{49.9, interopStatusBroken},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%.1f", tc.rate), func(t *testing.T) {
			require.Equal(t, tc.status, interopStatusFromRate(tc.rate))
		})
	}
}
```

- [ ] **Step 2: Run test — expect compile fail**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestInteropStatusFromRate -v`  
Expected: undefined `interopStatus` / `interopStatusFromRate`

- [ ] **Step 3: Implement types and status**

```go
// pkg/internal/apis/handlers/scoreboard_interop.go
type interopStatus string

const (
	interopStatusStable  interopStatus = "stable"
	interopStatusFlaky   interopStatus = "flaky"
	interopStatusFailing interopStatus = "failing"
	interopStatusBroken  interopStatus = "broken"
)

type interopMode string

const interopModeWalletsIssuers interopMode = "wallets_issuers"

type interopCacheInput struct {
	PipelineID     string
	TotalRuns      int
	TotalSuccesses int
	RowIDs         []string // wallet IDs for wallets_issuers mode
	ColumnIDs      []string // issuer IDs
}

type interopCellAccumulator struct {
	pipelineIDs    map[string]struct{}
	totalRuns      int
	totalSuccesses int
}

type InteropMatrixEntity struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Path          string  `json:"path"`
	VersionLabel  *string `json:"version_label,omitempty"`
}

type InteropMatrixCell struct {
	RowID          string        `json:"row_id"`
	ColumnID       string        `json:"column_id"`
	PipelineCount  int           `json:"pipeline_count"`
	TotalRuns      int           `json:"total_runs"`
	TotalSuccesses int           `json:"total_successes"`
	SuccessRate    float64       `json:"success_rate"`
	Status         interopStatus `json:"status"`
}

type InteropMatrixResponse struct {
	Mode       interopMode           `json:"mode"`
	RowAxis    string                `json:"row_axis"`
	ColumnAxis string                `json:"column_axis"`
	Rows       []InteropMatrixEntity `json:"rows"`
	Columns    []InteropMatrixEntity `json:"columns"`
	Cells      []InteropMatrixCell   `json:"cells"`
}

func interopStatusFromRate(rate float64) interopStatus {
	switch {
	case rate >= 90:
		return interopStatusStable
	case rate >= 70:
		return interopStatusFlaky
	case rate >= 50:
		return interopStatusFailing
	default:
		return interopStatusBroken
	}
}
```

- [ ] **Step 4: Run test — PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestInteropStatusFromRate -v`

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): add interop matrix types and status bands"
```

---

### Task 2: Pure matrix builder (aggregation)

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing builder tests**

```go
func TestBuildInteropMatrix_CartesianAndSums(t *testing.T) {
	t.Parallel()
	inputs := []interopCacheInput{
		{
			PipelineID: "p1", TotalRuns: 100, TotalSuccesses: 90,
			RowIDs: []string{"w1", "w2"}, ColumnIDs: []string{"i1"},
		},
		{
			PipelineID: "p2", TotalRuns: 84, TotalSuccesses: 66,
			RowIDs: []string{"w1"}, ColumnIDs: []string{"i1"},
		},
	}
	rows := map[string]InteropMatrixEntity{
		"w1": {ID: "w1", Name: "Wallet A", Path: "org/wallets/a"},
		"w2": {ID: "w2", Name: "Wallet B", Path: "org/wallets/b"},
	}
	cols := map[string]InteropMatrixEntity{
		"i1": {ID: "i1", Name: "Issuer X", Path: "org/issuers/x"},
	}

	matrix := buildInteropMatrix(inputs, rows, cols)

	require.Len(t, matrix.Cells, 2) // (w1,i1) and (w2,i1) only; p2 only w1

	var w1i1 *InteropMatrixCell
	for i := range matrix.Cells {
		if matrix.Cells[i].RowID == "w1" && matrix.Cells[i].ColumnID == "i1" {
			w1i1 = &matrix.Cells[i]
		}
	}
	require.NotNil(t, w1i1)
	require.Equal(t, 2, w1i1.PipelineCount)
	require.Equal(t, 184, w1i1.TotalRuns)
	require.Equal(t, 156, w1i1.TotalSuccesses)
	require.InDelta(t, 84.7826, w1i1.SuccessRate, 0.01)
	require.Equal(t, interopStatusFlaky, w1i1.Status)
}

func TestBuildInteropMatrix_SkipsEmptySides(t *testing.T) {
	t.Parallel()
	inputs := []interopCacheInput{
		{PipelineID: "p1", TotalRuns: 10, TotalSuccesses: 10, RowIDs: []string{"w1"}, ColumnIDs: nil},
	}
	matrix := buildInteropMatrix(inputs, map[string]InteropMatrixEntity{"w1": {ID: "w1", Name: "W"}}, nil)
	require.Empty(t, matrix.Cells)
}
```

- [ ] **Step 2: Run — FAIL** (`buildInteropMatrix` missing)

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestBuildInteropMatrix -v`

- [ ] **Step 3: Implement `buildInteropMatrix`**

```go
func buildInteropMatrix(
	inputs []interopCacheInput,
	rowEntities map[string]InteropMatrixEntity,
	columnEntities map[string]InteropMatrixEntity,
) InteropMatrixResponse {
	acc := map[string]*interopCellAccumulator{}

	for _, in := range inputs {
		if len(in.RowIDs) == 0 || len(in.ColumnIDs) == 0 || in.TotalRuns <= 0 {
			continue
		}
		for _, rowID := range in.RowIDs {
			for _, colID := range in.ColumnIDs {
				key := rowID + "\x00" + colID
				cell, ok := acc[key]
				if !ok {
					cell = &interopCellAccumulator{pipelineIDs: map[string]struct{}{}}
					acc[key] = cell
				}
				cell.pipelineIDs[in.PipelineID] = struct{}{}
				cell.totalRuns += in.TotalRuns
				cell.totalSuccesses += in.TotalSuccesses
			}
		}
	}

	resp := InteropMatrixResponse{
		Mode:       interopModeWalletsIssuers,
		RowAxis:    "wallet",
		ColumnAxis: "issuer",
	}
	rowSeen := map[string]struct{}{}
	colSeen := map[string]struct{}{}

	for key, cell := range acc {
		if cell.totalRuns <= 0 {
			continue
		}
		parts := strings.SplitN(key, "\x00", 2)
		rowID, colID := parts[0], parts[1]
		rate := float64(cell.totalSuccesses) / float64(cell.totalRuns) * 100
		resp.Cells = append(resp.Cells, InteropMatrixCell{
			RowID: rowID, ColumnID: colID,
			PipelineCount: len(cell.pipelineIDs),
			TotalRuns: cell.totalRuns, TotalSuccesses: cell.totalSuccesses,
			SuccessRate: rate, Status: interopStatusFromRate(rate),
		})
		rowSeen[rowID] = struct{}{}
		colSeen[colID] = struct{}{}
	}

	resp.Rows = sortedInteropEntities(rowEntities, rowSeen)
	resp.Columns = sortedInteropEntities(columnEntities, colSeen)
	return resp
}

func sortedInteropEntities(
	all map[string]InteropMatrixEntity,
	seen map[string]struct{},
) []InteropMatrixEntity {
	out := make([]InteropMatrixEntity, 0, len(seen))
	for id := range seen {
		if e, ok := all[id]; ok {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}
```

- [ ] **Step 4: Run — PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestBuildInteropMatrix -v`

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(scoreboard): add interop matrix builder"
```

---

### Task 3: Load cache from PocketBase + HTTP handler

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`
- Modify: `pkg/internal/apis/RoutesRegistry.go`

- [ ] **Step 1: Write failing HTTP tests**

```go
func setupScoreboardInteropApp(t testing.TB) *tests.TestApp {
	app := setupPipelineApp(t)
	ScoreboardInteropPublicRoutes.Add(app)
	return app
}

func TestHandleInteropMatrix_MissingMode(t *testing.T) {
	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()
	// httptest GET /api/scoreboard/interop → 400, body contains mode
}

func TestHandleInteropMatrix_WalletsIssuers(t *testing.T) {
	// Insert pipeline_scoreboard_cache record with wallets+issuers relations
	// GET ?mode=wallets_issuers → 200, cells[0] matches expected sums
}
```

Use patterns from `scoreboard_test.go` (`HandleSaveScoreboardResults` fixture) to create `pipelines`, `wallets`, `credential_issuers`, and a cache row with `record.Set("wallets", ...)`, `record.Set("issuers", ...)`.

- [ ] **Step 2: Run — FAIL**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestHandleInteropMatrix -v`

- [ ] **Step 3: Implement loader + handler + routes**

```go
var ScoreboardInteropPublicRoutes = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:      http.MethodGet,
			Path:        "/scoreboard/interop",
			Handler:     HandleInteropMatrix,
			Description: "Wallet×Issuer (etc.) interoperability matrix from scoreboard cache",
		},
	},
}

func HandleInteropMatrix() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		mode := interopMode(e.Request.URL.Query().Get("mode"))
		if mode != interopModeWalletsIssuers {
			return apierror.New(http.StatusBadRequest, "mode", "unsupported or missing mode", "use mode=wallets_issuers").JSON(e)
		}
		resp, err := loadInteropMatrixFromCache(e.App, mode)
		if err != nil {
			return apierror.New(http.StatusInternalServerError, "scoreboard", "failed to build interop matrix", err.Error()).JSON(e)
		}
		return e.JSON(http.StatusOK, resp)
	}
}

func loadInteropMatrixFromCache(app core.App, mode interopMode) (InteropMatrixResponse, error) {
	coll, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	if err != nil {
		return InteropMatrixResponse{}, err
	}
	records, err := app.FindRecordsByFilter(coll.Id, "", "", -1, 0)
	if err != nil {
		return InteropMatrixResponse{}, err
	}

	var inputs []interopCacheInput
	rowEntities := map[string]InteropMatrixEntity{}
	columnEntities := map[string]InteropMatrixEntity{}

	for _, rec := range records {
		rowIDs := rec.GetStringSlice("wallets")
		colIDs := rec.GetStringSlice("issuers")
		pipelineID := rec.GetString("pipeline")
		inputs = append(inputs, interopCacheInput{
			PipelineID: pipelineID,
			TotalRuns: rec.GetInt("total_runs"),
			TotalSuccesses: rec.GetInt("total_successes"),
			RowIDs: rowIDs, ColumnIDs: colIDs,
		})
		if err := mergeWalletEntities(app, rec, rowIDs, rowEntities); err != nil {
			return InteropMatrixResponse{}, err
		}
		if err := mergeIssuerEntities(app, colIDs, columnEntities); err != nil {
			return InteropMatrixResponse{}, err
		}
	}
	return buildInteropMatrix(inputs, rowEntities, columnEntities), nil
}
```

Implement `mergeWalletEntities` / `mergeIssuerEntities`:
- Load `wallets` / `credential_issuers` by ID
- `Name` from record `name` field
- `Path` via `canonify.BuildPath(app, rec, template, rec.GetString("canonified_name"))` (mirror `pipeline_ci_helpers.go` / existing handler patterns)
- `VersionLabel`: if cache row has `wallet_versions`, load versions and set `version_label` to `tag` when `wallet_versions.wallet == wallet.id` (first match per wallet)

- [ ] **Step 4: Register route**

```go
// pkg/internal/apis/RoutesRegistry.go — add to RouteGroups slice:
handlers.ScoreboardInteropPublicRoutes,
```

- [ ] **Step 5: Run tests — PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestHandleInteropMatrix|TestBuildInteropMatrix|TestInteropStatus' -v`

- [ ] **Step 6: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go pkg/internal/apis/RoutesRegistry.go
git commit -m "feat(scoreboard): add public interop matrix API"
```

---

### Task 4: Frontend types and API fetch

**Files:**
- Create: `webapp/src/lib/scoreboard/interop/types.ts`
- Create: `webapp/src/lib/scoreboard/interop/status.ts`
- Create: `webapp/src/routes/(public)/scoreboard/interop/+page.ts`

- [ ] **Step 1: Add types**

```ts
// webapp/src/lib/scoreboard/interop/types.ts
export type InteropStatus = 'stable' | 'flaky' | 'failing' | 'broken';

export type InteropMatrixEntity = {
	id: string;
	name: string;
	path: string;
	version_label?: string;
};

export type InteropMatrixCell = {
	row_id: string;
	column_id: string;
	pipeline_count: number;
	total_runs: number;
	total_successes: number;
	success_rate: number;
	status: InteropStatus;
};

export type InteropMatrixResponse = {
	mode: string;
	row_axis: string;
	column_axis: string;
	rows: InteropMatrixEntity[];
	columns: InteropMatrixEntity[];
	cells: InteropMatrixCell[];
};
```

- [ ] **Step 2: Add status classes**

```ts
// webapp/src/lib/scoreboard/interop/status.ts
import type { InteropStatus } from './types';

const STATUS_STYLES: Record<InteropStatus, { bg: string; text: string }> = {
	stable: { bg: 'bg-emerald-100', text: 'text-emerald-800' },
	flaky: { bg: 'bg-amber-100', text: 'text-amber-800' },
	failing: { bg: 'bg-orange-100', text: 'text-orange-800' },
	broken: { bg: 'bg-red-100', text: 'text-red-800' }
};

export function interopStatusStyles(status: InteropStatus) {
	return STATUS_STYLES[status];
}
```

- [ ] **Step 3: Add `+page.ts` loader**

```ts
// webapp/src/routes/(public)/scoreboard/interop/+page.ts
import type { InteropMatrixResponse } from '$lib/scoreboard/interop/types';

export const load = async ({ fetch }) => {
	const res = await fetch('/api/scoreboard/interop?mode=wallets_issuers');
	if (!res.ok) {
		throw new Error(`interop matrix: ${res.status}`);
	}
	const matrix: InteropMatrixResponse = await res.json();
	return { matrix };
};
```

- [ ] **Step 4: Verify types**

Run: `cd webapp && bun run check`

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/ webapp/src/routes/(public)/scoreboard/interop/+page.ts
git commit -m "feat(webapp): add interop matrix types and loader"
```

---

### Task 5: Matrix UI components

**Files:**
- Create: `webapp/src/lib/scoreboard/interop/matrix-cell.svelte`
- Create: `webapp/src/lib/scoreboard/interop/matrix-grid.svelte`
- Create: `webapp/src/routes/(public)/scoreboard/interop/+page.svelte`

- [ ] **Step 1: `matrix-cell.svelte`**

Filled cell:
- `Math.round(cell.success_rate)` + `%` headline
- `{total_successes}/{total_runs}` subline
- `{pipeline_count}` + i18n pipe label
- Apply `interopStatusStyles(cell.status)` on container

Empty: muted “Not tested” (`m.interop_matrix_not_tested()`).

- [ ] **Step 2: `matrix-grid.svelte`**

- Build `Map<string, InteropMatrixCell>` keyed by `` `${row_id}:${column_id}` ``
- `<table>` with `sticky` first row/column (Tailwind `sticky left-0 top-0 z-*`)
- Corner: `m.interop_matrix_corner_label()`
- Row header: name + optional `version_label` in muted text
- Column header: name; link to `/hub/{path}` via `resolve()` if hub routes use path (match `getPocketbaseEntityHref` pattern from scoreboard entity-display)
- Accept `legend` snippet prop for page-level legend

- [ ] **Step 3: `+page.svelte`**

```svelte
<script lang="ts">
	import PublicPageHeader from '@/components/layout/public-page-header.svelte';
	import MatrixGrid from '$lib/scoreboard/interop/matrix-grid.svelte';
	import { interopStatusStyles } from '$lib/scoreboard/interop/status';
	import { m } from '@/i18n';

	let { data } = $props();
</script>

<div class="grow bg-secondary pb-20">
	<PublicPageHeader
		title={m.interop_matrix_title()}
		description={m.interop_matrix_description()}
	/>
	<!-- legend row: Broken, Failing, Flaky, Stable with colored dots -->
	<MatrixGrid matrix={data.matrix} />
	<p class="px-4 text-sm text-muted-foreground">{m.interop_matrix_footnote()}</p>
</div>
```

Use Svelte MCP autofixer on new `.svelte` files before commit.

- [ ] **Step 4: Manual smoke**

Run: `make dev` (or API + webapp), open `/scoreboard/interop`, confirm grid renders with local `pipeline_scoreboard_cache` data.

- [ ] **Step 5: Commit**

```bash
git commit -am "feat(webapp): add interop matrix page and components"
```

---

### Task 6: i18n and navigation

**Files:**
- Modify: `webapp/messages/en.json`
- Modify: `webapp/src/routes/(public)/scoreboard/+page.svelte`

- [ ] **Step 1: Add English keys**

```json
"interop_matrix_title": "Interop matrix",
"interop_matrix_description": "Cross-testing results between wallets and issuers from published pipelines.",
"interop_matrix_corner_label": "WALLET ↓ / ISSUER →",
"interop_matrix_not_tested": "Not tested",
"interop_matrix_footnote": "Metrics aggregate pipelines whose last successful run included both entities. Multiple wallets or issuers on one pipeline count toward each matching pair.",
"interop_matrix_legend_broken": "Broken",
"interop_matrix_legend_failing": "Failing",
"interop_matrix_legend_flaky": "Flaky",
"interop_matrix_legend_stable": "Stable",
"interop_matrix_pipeline_count_one": "{count} pipe.",
"interop_matrix_pipeline_count_other": "{count} pipes.",
"scoreboard_view_interop_matrix": "Interop matrix"
```

Run project i18n generation if required (`cd webapp && bun run paraglide` or whatever `package.json` documents).

- [ ] **Step 2: Link from scoreboard**

In `scoreboard/+page.svelte`, below `PublicPageHeader`, add a secondary button:

```svelte
<Button variant="outline" href={resolve('/scoreboard/interop')}>
	{m.scoreboard_view_interop_matrix()}
</Button>
```

- [ ] **Step 3: Commit**

```bash
git commit -am "feat(webapp): interop matrix i18n and scoreboard nav link"
```

---

### Task 7: Verification and docs touch-up

**Files:**
- Modify: `docs/src/content/docs/software-architecture/scoreboard.md` (short section linking endpoint + page)

- [ ] **Step 1: Run Go unit tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run Interop -v`  
Expected: all PASS

- [ ] **Step 2: Run webapp check**

Run: `cd webapp && bun run check && bun run lint`

- [ ] **Step 3: Add docs snippet**

Under scoreboard.md **Frontend**, add:

- Page `/scoreboard/interop`
- API `GET /api/scoreboard/interop?mode=wallets_issuers`
- Cell metric definition (Σ successes / Σ runs)

- [ ] **Step 4: Commit**

```bash
git commit -am "docs: document scoreboard interop matrix API and page"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `GET /api/scoreboard/interop?mode=wallets_issuers` | Task 3 |
| Run-weighted `success_rate` + matching fraction | Task 2, 5 |
| Cartesian attribution | Task 2 |
| Status bands on `success_rate` | Task 1, 5 |
| `/scoreboard/interop` public page | Task 4–5 |
| Data-driven rows/columns | Task 3 (`sortedInteropEntities` + seen set) |
| Link from `/scoreboard` | Task 6 |
| Footnote semantics | Task 6 |
| Wallet×Specs deferred | — |
| Future modes in query enum | Task 3 validates only `wallets_issuers` for now |

---

## GitNexus / impact notes

Before editing handlers, run `gitnexus_impact` on `HandleInteropMatrix` / new symbols once registered. Before commit, run `gitnexus_detect_changes()`. New route group is additive; low blast radius.

---

## Out of scope (follow-up PRs)

- `wallets_verifiers`, `issuers_verifiers` modes
- Wallet×Specs matrix
- Cell click-through to filtered scoreboard
- API response caching
- Homepage promo block
