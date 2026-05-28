# Scoreboard Interop Conformance Checks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `wallets_conformance_checks` mode to the scoreboard interop API/page, reading `conformance_checks` JSON field from `pipeline_scoreboard_cache` as path-based column IDs, emitting minimal backend metadata, and enriching columns at render time from the already-loaded conformance store.

**Architecture:** Backend: add `ColumnIsPathBased` bool to `interopModeConfig`, extract column IDs from the JSON field via `getPathIDs()`, emit humanized check names as placeholders. Frontend: `resolveConformanceCheck()` enriches name/subtitle/logo from the conformance standards store. Aggregation semantics unchanged.

**Tech Stack:** Go 1.24, PocketBase handlers/tests (`stretchr/testify`), SvelteKit + TypeScript, Paraglide i18n, Tailwind.

**Spec:** `docs/superpowers/specs/2026-05-28-scoreboard-interop-conformance-checks-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/scoreboard_interop.go` | Add `ColumnIsPathBased` to config struct, add new mode constant + config, `getPathIDs()`, `conformanceCheckName()`, update loader for path-based column extraction |
| `pkg/internal/apis/handlers/scoreboard_interop_test.go` | Tests: mode validation, config, `getPathIDs`, `conformanceCheckName`, aggregation with path IDs, handler HTTP |
| `webapp/src/lib/scoreboard/interop/types.ts` | Add `'wallets_conformance_checks'` to `InteropMode` union |
| `webapp/src/lib/scoreboard/interop/resolve-conformance.ts` | NEW: `resolveConformanceCheck(path, standards)` → metadata enrichment |
| `webapp/src/routes/(public)/scoreboard/interop/+page.ts` | Add new mode to `SUPPORTED_MODES` |
| `webapp/src/routes/(public)/scoreboard/interop/+page.svelte` | Add mode tab entry, import/resolve conformance metadata |
| `webapp/src/lib/scoreboard/interop/matrix-grid.svelte` | Enrich column headers when mode is `wallets_conformance_checks` |
| `webapp/messages/en.json` | Add i18n key for new mode |

---

### Task 1: Backend mode config and path extraction helpers

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing tests for mode validation, config, path extraction, and name humanizing**

```go
func TestInteropModeValidation_ConformanceChecks(t *testing.T) {
	t.Parallel()
	require.True(t, isSupportedInteropMode(interopModeWalletsConformanceChecks))
	require.True(t, isSupportedInteropMode(interopModeWalletsCredentials))
	require.False(t, isSupportedInteropMode(interopMode("bad_mode")))
}

func TestInteropModeConfig_ConformanceChecks(t *testing.T) {
	t.Parallel()
	cfg, ok := getInteropModeConfig(interopModeWalletsConformanceChecks)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "conformance_checks", cfg.ColumnRelationField)
	require.True(t, cfg.ColumnIsPathBased)
	require.Equal(t, "wallet", cfg.RowAxis)
	require.Equal(t, "conformance_check", cfg.ColumnAxis)
}

func TestGetPathIDs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty array",
			input:    []interface{}{},
			expected: nil,
		},
		{
			name:     "string array",
			input:    []interface{}{"a/b/c", "d/e/f"},
			expected: []string{"a/b/c", "d/e/f"},
		},
		{
			name:     "mixed with empty",
			input:    []interface{}{"a/b/c", "", "d/e/f"},
			expected: []string{"a/b/c", "d/e/f"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := getPathIDs(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestConformanceCheckName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path     string
		expected string
	}{
		{
			path:     "openid4vp_wallet/1.0/webuild/WEBUILD-VP001-x509-direct_post-post-dcql-sd_jwt",
			expected: "Webuild Vp001 X509 Direct Post Post Dcql Sd Jwt",
		},
		{
			path:     "short",
			expected: "Short",
		},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, tc.expected, conformanceCheckName(tc.path))
		})
	}
}
```

- [ ] **Step 2: Run test — expect compile fail**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestInteropModeValidation_ConformanceChecks|TestInteropModeConfig_ConformanceChecks|TestGetPathIDs|TestConformanceCheckName' -v`  
Expected: FAIL with undefined `interopModeWalletsConformanceChecks`, `getPathIDs`, `conformanceCheckName`, `ColumnIsPathBased`.

- [ ] **Step 3: Implement mode constant, config field, and helpers**

Add to `interopModeConfig`:

```go
type interopModeConfig struct {
	RowRelationField    string
	ColumnRelationField string
	RowAxis             string
	ColumnAxis          string
	RowCollection       string
	ColumnCollection    string
	ColumnIsPathBased   bool // NEW: true when column IDs are path strings from a JSON field (not PB relation IDs)
}
```

Add mode constant:

```go
const (
	interopModeWalletsIssuers              interopMode = "wallets_issuers"
	interopModeWalletsCredentials          interopMode = "wallets_credentials"
	interopModeWalletsVerifiers            interopMode = "wallets_verifiers"
	interopModeWalletsUseCaseVerifications interopMode = "wallets_use_case_verifications"
	interopModeWalletsConformanceChecks    interopMode = "wallets_conformance_checks" // NEW
)
```

Add mode config entry:

```go
interopModeWalletsConformanceChecks: {
	RowRelationField:    "wallets",
	ColumnRelationField: "conformance_checks",
	RowAxis:             "wallet",
	ColumnAxis:          "conformance_check",
	RowCollection:       "wallets",
	ColumnCollection:    "",
	ColumnIsPathBased:   true,
},
```

Add helpers:

```go
func getPathIDs(raw interface{}) []string {
	switch v := raw.(type) {
	case []interface{}:
		ids := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok || s == "" {
				continue
			}
			ids = append(ids, s)
		}
		return ids
	case []string:
		ids := make([]string, 0, len(v))
		for _, s := range v {
			if s == "" {
				continue
			}
			ids = append(ids, s)
		}
		return ids
	}
	return nil
}

func conformanceCheckName(pathID string) string {
	parts := strings.Split(pathID, "/")
	last := parts[len(parts)-1]
	ext := strings.LastIndex(last, ".")
	if ext >= 0 {
		last = last[:ext]
	}
	name := strings.NewReplacer("-", " ", "_", " ").Replace(last)
	name = strings.TrimSpace(name)
	if name == "" {
		return pathID
	}
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}
```

- [ ] **Step 4: Run tests — PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestInteropModeValidation_ConformanceChecks|TestInteropModeConfig_ConformanceChecks|TestGetPathIDs|TestConformanceCheckName' -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): add conformance_checks interop mode config and path helpers"
```

---

### Task 2: Update loader for path-based column extraction

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing aggregation test with path-based column IDs**

```go
func TestBuildInteropMatrix_PathBasedColumns(t *testing.T) {
	t.Parallel()
	inputs := []interopCacheInput{
		{
			PipelineID: "p1", TotalRuns: 100, TotalSuccesses: 80,
			RowIDs:    []string{"w1"},
			ColumnIDs: []string{"openid4vp_wallet/1.0/webuild/check-a"},
		},
		{
			PipelineID: "p2", TotalRuns: 50, TotalSuccesses: 30,
			RowIDs:    []string{"w1"},
			ColumnIDs: []string{"openid4vp_wallet/1.0/webuild/check-b"},
		},
	}
	rowEntities := map[string]InteropMatrixEntity{
		"w1": {ID: "w1", Name: "Wallet", Path: "org/wallets/w1"},
	}
	colEntities := map[string]InteropMatrixEntity{
		"openid4vp_wallet/1.0/webuild/check-a": {ID: "openid4vp_wallet/1.0/webuild/check-a", Name: "Check A", Path: "openid4vp_wallet/1.0/webuild/check-a"},
		"openid4vp_wallet/1.0/webuild/check-b": {ID: "openid4vp_wallet/1.0/webuild/check-b", Name: "Check B", Path: "openid4vp_wallet/1.0/webuild/check-b"},
	}

	resp := buildInteropMatrix(inputs, rowEntities, colEntities)
	require.Len(t, resp.Cells, 2)
	require.Len(t, resp.Columns, 2)
	require.Len(t, resp.Rows, 1)
}
```

- [ ] **Step 2: Run — FAIL** (test depends on loader integration, may need handler test)

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestBuildInteropMatrix_PathBasedColumns -v`  
Expected: FAIL for missing handler-level integration (will pass once builder is wired through loader).

- [ ] **Step 3: Update `loadInteropMatrixFromCache` for path-based columns**

In the cache row loop, branch on `ColumnIsPathBased`:

```go
for _, record := range records {
	rowIDs := record.GetStringSlice(modeConfig.RowRelationField)

	var colIDs []string
	if modeConfig.ColumnIsPathBased {
		colIDs = getPathIDs(record.Get(modeConfig.ColumnRelationField))
		if len(colIDs) > 0 {
			for _, pathID := range colIDs {
				if _, ok := columnEntities[pathID]; !ok {
					columnEntities[pathID] = InteropMatrixEntity{
						ID:   pathID,
						Name: conformanceCheckName(pathID),
						Path: pathID,
					}
				}
			}
		}
	} else {
		colIDs = record.GetStringSlice(modeConfig.ColumnRelationField)
	}

	inputs = append(inputs, interopCacheInput{
		PipelineID:     record.GetString("pipeline"),
		TotalRuns:      record.GetInt("total_runs"),
		TotalSuccesses: record.GetInt("total_successes"),
		RowIDs:         rowIDs,
		ColumnIDs:      colIDs,
	})

	if err := mergeInteropEntities(app, modeConfig.RowCollection, record, rowIDs, rowEntities); err != nil {
		return InteropMatrixResponse{}, err
	}
	if !modeConfig.ColumnIsPathBased {
		if err := mergeInteropEntities(app, modeConfig.ColumnCollection, nil, colIDs, columnEntities); err != nil {
			return InteropMatrixResponse{}, err
		}
	}
}
```

Also update handler error message (description) in route definition and `HandleInteropMatrix()`:

```go
// Route definition
Description: "Matrix pair mode. Supports wallets_credentials, wallets_issuers, wallets_verifiers, wallets_use_case_verifications, or wallets_conformance_checks.",

// Handler error
"use mode=wallets_credentials, wallets_issuers, wallets_verifiers, wallets_use_case_verifications, or wallets_conformance_checks",
```

- [ ] **Step 4: Run tests — PASS**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestHandleInteropMatrix|TestBuildInteropMatrix|TestInteropMode' -v`  
Expected: Existing tests still PASS, new test PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): load conformance_checks as path-based interop columns"
```

---

### Task 3: Frontend types and resolve-conformance utility

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/types.ts`
- Create: `webapp/src/lib/scoreboard/interop/resolve-conformance.ts`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.ts`

- [ ] **Step 1: Add new mode to types and SUPPORTED_MODES**

```ts
// webapp/src/lib/scoreboard/interop/types.ts
export type InteropMode =
	| 'wallets_credentials'
	| 'wallets_issuers'
	| 'wallets_verifiers'
	| 'wallets_use_case_verifications'
	| 'wallets_conformance_checks'; // NEW
```

```ts
// webapp/src/routes/(public)/scoreboard/interop/+page.ts
const SUPPORTED_MODES: InteropMode[] = [
	'wallets_credentials',
	'wallets_issuers',
	'wallets_verifiers',
	'wallets_use_case_verifications',
	'wallets_conformance_checks' // NEW
];
```

- [ ] **Step 2: Create `resolve-conformance.ts`**

```ts
// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Standard } from '$lib/conformance/types';

export type ConformanceMetadata = {
	name: string;
	subtitle?: string;
	avatar_url?: string;
};

export function resolveConformanceCheck(
	path: string,
	standards: readonly Standard[]
): ConformanceMetadata | undefined {
	const parts = path.split('/');
	if (parts.length < 4) return undefined;

	const [standardUid, versionUid, suiteUid, ...checkParts] = parts;
	const checkName = checkParts.join('/');

	const standard = standards.find((s) => s.uid === standardUid);
	const version = standard?.versions.find((v) => v.uid === versionUid);
	const suite = version?.suites.find((s) => s.uid === suiteUid);

	if (!suite) return undefined;

	return {
		name: checkName,
		subtitle: suite.name,
		avatar_url: suite.logo
	};
}
```

- [ ] **Step 3: Run check**

Run: `cd webapp && bun run check`  
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/types.ts webapp/src/lib/scoreboard/interop/resolve-conformance.ts webapp/src/routes/(public)/scoreboard/interop/+page.ts
git commit -m "feat(webapp): add conformance_checks mode type and path resolver"
```

---

### Task 4: Frontend mode selector and grid enrichment

**Files:**
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.svelte`
- Modify: `webapp/src/lib/scoreboard/interop/matrix-grid.svelte`
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: Add mode tab**

In `+page.svelte`, add to `modeTabs` array:

```svelte
{ value: 'wallets_conformance_checks', label: () => m.interop_mode_wallets_conformance_checks() },
```

- [ ] **Step 2: Add i18n key**

In `webapp/messages/en.json`:

```json
"interop_mode_wallets_conformance_checks": "Wallet x Conformance Checks"
```

- [ ] **Step 3: Enrich column headers in matrix-grid.svelte**

Import the resolver and conformance store in `matrix-grid.svelte`:

```svelte
<script lang="ts">
	import { get as getConformanceStore } from '$lib/conformance/store';
	import { resolveConformanceCheck } from './resolve-conformance';

	// ... existing code ...

	const isConformanceMode = $derived(matrix.mode === 'wallets_conformance_checks');

	function enrichedColumn(column: InteropMatrixEntity) {
		if (!isConformanceMode) return column;
		const resolved = resolveConformanceCheck(column.id, getConformanceStore().standards);
		if (!resolved) return column;
		return {
			...column,
			name: resolved.name,
			subtitle: resolved.subtitle ?? undefined,
			avatar_url: resolved.avatar_url ?? undefined
		};
	}

	function hubHref(collection: 'wallets' | 'credential_issuers' | 'credentials', path: string) {
		return `/hub/${collection}/${path}`;
	}
</script>
```

In the `{#each matrix.columns as column}` block, use `enrichedColumn(column)` for display props (name, subtitle, avatar) but keep `column.path` and `column.id` for link targets:

```svelte
{#each matrix.columns as column (column.id)}
	{@const enriched = enrichedColumn(column)}
	{@const columnSubtitle = getSubtitleOrVersion(enriched.subtitle, enriched.version_label)}
	<th class="sticky top-0 z-10 min-w-32 border-b bg-muted/60 px-3 py-3 text-center font-semibold">
		{#if isConformanceMode}
			<!-- Conformance mode: link to /hub/{path} directly using the check path -->
			<a class="inline-flex max-w-44 flex-col items-center gap-1 hover:underline" href={`/hub/${column.path}`}>
				{#if enriched.avatar_url}
					<img src={enriched.avatar_url} alt={enriched.name} class="size-6 rounded-full object-cover" loading="lazy" />
				{/if}
				<span>{enriched.name}</span>
				{#if columnSubtitle}
					<span class="text-xs font-normal text-muted-foreground">{columnSubtitle}</span>
				{/if}
			</a>
		{:else}
			<a class="inline-flex max-w-44 flex-col items-center gap-1 hover:underline" href={hubHref(columnCollection, column.path)}>
				{#if column.avatar_url}
					<img src={column.avatar_url} alt={column.name} class="size-6 rounded-full object-cover" loading="lazy" />
				{/if}
				<span>{column.name}</span>
				{#if columnSubtitle}
					<span class="text-xs font-normal text-muted-foreground">{columnSubtitle}</span>
				{/if}
			</a>
		{/if}
	</th>
{/each}
```

- [ ] **Step 4: Run check and lint**

Run: `cd webapp && bun run check && bun run lint`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webapp/src/routes/(public)/scoreboard/interop/+page.svelte webapp/src/lib/scoreboard/interop/matrix-grid.svelte webapp/messages/en.json
git commit -m "feat(webapp): render conformance_checks interop mode with store enrichment"
```

---

### Task 5: Verification

- [ ] **Step 1: Run Go unit tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run Interop -v`  
Expected: ALL PASS.

- [ ] **Step 2: Run frontend checks**

Run: `cd webapp && bun run check && bun run lint`  
Expected: PASS.

- [ ] **Step 3: Commit verification**

```bash
git commit -m "chore(scoreboard): verify wallets_conformance_checks interop mode" --allow-empty
```

---

## Spec coverage checklist

| Spec requirement | Plan task |
|------------------|-----------|
| `wallets_conformance_checks` mode on API | Tasks 1, 2 |
| `ColumnIsPathBased` config field | Task 1 |
| `getPathIDs()` JSON extraction | Task 1 |
| `conformanceCheckName()` humanizing | Task 1 |
| Path-based column ID extraction in loader | Task 2 |
| Mode validation includes new mode | Tasks 1, 2 |
| Minimal backend column metadata | Task 2 |
| `resolveConformanceCheck()` utility | Task 3 |
| Grid enrichment from conformance store | Task 4 |
| Mode selector tab | Task 4 |
| Hub link to full check path | Task 4 |
| Same metadata contract, existing modes preserved | Tasks 1, 2, 3, 4 |
| Run-weighted + Cartesian semantics unchanged | Task 2 |

---

## Self-review

1. **Spec coverage:** All approved design requirements mapped; no missing feature slice.
2. **Placeholder scan:** No TODO/TBD markers; each task has concrete files, commands, and expected outcomes.
3. **Type consistency:** Mode name `wallets_conformance_checks` consistent across Go, TypeScript, and i18n.
