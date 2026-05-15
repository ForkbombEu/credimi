# Pipeline Execution Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show lifetime execution stats from `pipeline_scoreboard_cache` on pipeline cards and the pipeline detail page, reusing one shared Svelte component aligned with the scoreboard column.

**Architecture:** Extract `fromScoreboardRow()` (pure TS) and `pipeline-execution-stats.svelte` (three layouts). Refactor the scoreboard column to delegate to it. Wire the card’s existing client fetch and the detail page’s new server load. Remove incorrect paginated-workflow stat math on the detail page.

**Tech Stack:** Svelte 5, SvelteKit, Vitest, PocketBase scoreboard cache, existing i18n keys.

**Spec:** `docs/superpowers/specs/2026-05-15-pipeline-execution-stats-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/scoreboard/extras/from-scoreboard-row.ts` | Pure mapping `ScoreboardRow` → `ExecutionStats` |
| `webapp/src/lib/scoreboard/extras/from-scoreboard-row.test.ts` | Unit tests for mapper |
| `webapp/src/lib/scoreboard/extras/pipeline-execution-stats.svelte` | Shared UI (`inline`, `stat-box-success`, `stat-box-modes`) |
| `webapp/src/lib/scoreboard/columns/total-executions-successes-percentage.svelte` | Column definition + thin cell wrapper |
| `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte` | Card placement + `content` visibility |
| `webapp/src/routes/my/pipelines/[...pipeline_path]/+page.ts` | Server load scoreboard |
| `webapp/src/routes/my/pipelines/[...pipeline_path]/+page.svelte` | Two stat boxes |

---

### Task 1: `fromScoreboardRow` helper

**Files:**
- Create: `webapp/src/lib/scoreboard/extras/from-scoreboard-row.ts`
- Create: `webapp/src/lib/scoreboard/extras/from-scoreboard-row.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// webapp/src/lib/scoreboard/extras/from-scoreboard-row.test.ts
import { describe, expect, it } from 'vitest';

import type { ScoreboardRow } from '../types';

import { fromScoreboardRow, emptyExecutionStats } from './from-scoreboard-row';

describe('fromScoreboardRow', () => {
	it('returns undefined when row is undefined', () => {
		expect(fromScoreboardRow(undefined)).toBeUndefined();
	});

	it('maps scoreboard fields with defaults', () => {
		const row = {
			total_runs: 10,
			total_successes: 8,
			success_rate: 80,
			manually_executed_runs: 3,
			scheduled_runs: 2,
			CI_runs: 5
		} as ScoreboardRow;

		expect(fromScoreboardRow(row)).toEqual({
			total: 10,
			successes: 8,
			percent: 80,
			manual: 3,
			scheduled: 2,
			ci: 5
		});
	});

	it('coalesces missing optional numbers to zero', () => {
		const row = { total_runs: 0 } as ScoreboardRow;
		expect(fromScoreboardRow(row)).toEqual({
			total: 0,
			successes: 0,
			percent: 0,
			manual: 0,
			scheduled: 0,
			ci: 0
		});
	});
});

describe('emptyExecutionStats', () => {
	it('is all zeros', () => {
		expect(emptyExecutionStats).toEqual({
			total: 0,
			successes: 0,
			percent: 0,
			manual: 0,
			scheduled: 0,
			ci: 0
		});
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/scoreboard/extras/from-scoreboard-row.test.ts`

Expected: FAIL — module not found

- [ ] **Step 3: Implement helper**

```ts
// webapp/src/lib/scoreboard/extras/from-scoreboard-row.ts
import type { ScoreboardRow } from '../types';

export type ExecutionStats = {
	total: number;
	successes: number;
	percent: number;
	manual: number;
	scheduled: number;
	ci: number;
};

export const emptyExecutionStats: ExecutionStats = {
	total: 0,
	successes: 0,
	percent: 0,
	manual: 0,
	scheduled: 0,
	ci: 0
};

export function fromScoreboardRow(row: ScoreboardRow | undefined): ExecutionStats | undefined {
	if (!row) return undefined;

	return {
		total: row.total_runs ?? 0,
		successes: row.total_successes ?? 0,
		percent: row.success_rate ?? 0,
		manual: row.manually_executed_runs ?? 0,
		scheduled: row.scheduled_runs ?? 0,
		ci: row.CI_runs ?? 0
	};
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd webapp && bun run test:unit -- src/lib/scoreboard/extras/from-scoreboard-row.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/extras/from-scoreboard-row.ts \
        webapp/src/lib/scoreboard/extras/from-scoreboard-row.test.ts
git commit -m "feat(scoreboard): add fromScoreboardRow execution stats helper"
```

---

### Task 2: Shared `pipeline-execution-stats` component

**Files:**
- Create: `webapp/src/lib/scoreboard/extras/pipeline-execution-stats.svelte`

- [ ] **Step 1: Create component** (no unit test — presentational; column/card are visual checks)

```svelte
<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV
SPDX-License-Identifier: AGPL-3.0-or-later
-->
<script lang="ts">
	import { ClockIcon, CogIcon, HandIcon } from '@lucide/svelte';

	import type { IconComponent } from '@/components/types';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';

	import type { ExecutionStats } from './from-scoreboard-row';

	type Layout = 'inline' | 'stat-box-success' | 'stat-box-modes';

	type Props = {
		stats: ExecutionStats;
		layout: Layout;
		label?: string;
	};

	let { stats, layout, label }: Props = $props();

	type ExecutionModeCount = {
		icon: IconComponent;
		count: number;
		label: string;
	};

	const executionTypes: ExecutionModeCount[] = $derived([
		{ icon: HandIcon, count: stats.manual, label: m.Executed_manually() },
		{ icon: ClockIcon, count: stats.scheduled, label: m.Executed_via_scheduling() },
		{ icon: CogIcon, count: stats.ci, label: m.Executed_via_ci() }
	]);

	const successClass = $derived(
		stats.percent >= 70 ? 'text-emerald-600' : undefined
	);
</script>

{#snippet successLine(className?: string)}
	<p class={['font-bold', className, successClass]}>
		{stats.successes}/{stats.total} ({stats.percent}%)
	</p>
{/snippet}

{#snippet modesLine(className?: string)}
	<p class={className}>
		{#each executionTypes as executionType, index (executionType.label)}
			<Tooltip>
				<span>
					{executionType.count}
					<executionType.icon class="-ml-0.5 inline-block size-3 -translate-y-px" />
				</span>
				{#snippet content()}
					<p>
						<executionType.icon class="inline-block size-3 -translate-y-px" />
						{executionType.label}
					</p>
				{/snippet}
			</Tooltip>
			{#if index < executionTypes.length - 1}
				<span class="pr-1 pl-0.5">/</span>
			{/if}
		{/each}
	</p>
{/snippet}

{#if layout === 'inline'}
	<div class="pr-3 text-right">
		{@render successLine('text-sm')}
		{@render modesLine('text-xs text-muted-foreground opacity-80')}
	</div>
{:else if layout === 'stat-box-success'}
	<div class="flex h-20 w-[140px] flex-col items-start justify-between rounded-lg border p-3">
		<T tag="h2" class={['mb-0! pb-0!', successClass]}>{stats.successes}/{stats.total} ({stats.percent}%)</T>
		<T class="text-sm">{label}</T>
	</div>
{:else}
	<div class="flex h-20 w-[140px] flex-col items-start justify-between rounded-lg border p-3">
		<div class="text-lg font-semibold leading-tight">
			{@render modesLine('text-sm')}
		</div>
		<T class="text-sm">{label}</T>
	</div>
{/if}
```

- [ ] **Step 2: Run Svelte check on the new file**

Run: `cd webapp && bun run check 2>&1 | head -40`

Expected: no errors for `pipeline-execution-stats.svelte`

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/scoreboard/extras/pipeline-execution-stats.svelte
git commit -m "feat(scoreboard): add pipeline-execution-stats shared component"
```

---

### Task 3: Refactor scoreboard column

**Files:**
- Modify: `webapp/src/lib/scoreboard/columns/total-executions-successes-percentage.svelte`

- [ ] **Step 1: Replace cell markup with shared component**

Keep `<script module>` `column` definition unchanged (same `fn` return shape).

Replace `<script>` + template with:

```svelte
<script lang="ts">
	import PipelineExecutionStats from '../extras/pipeline-execution-stats.svelte';

	import * as Column from '../column';

	let { value }: Column.Props<typeof column> = $props();
</script>

<PipelineExecutionStats stats={value} layout="inline" />
```

Remove unused imports (`ClockIcon`, `HandIcon`, `Tooltip`, etc.) from this file.

- [ ] **Step 2: Verify scoreboard still typechecks**

Run: `cd webapp && bun run check 2>&1 | head -40`

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/scoreboard/columns/total-executions-successes-percentage.svelte
git commit -m "refactor(scoreboard): delegate success-rate column to shared stats component"
```

---

### Task 4: Pipeline card

**Files:**
- Modify: `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte`

- [ ] **Step 1: Add imports and derived stats**

```ts
import PipelineExecutionStats from '$lib/scoreboard/extras/pipeline-execution-stats.svelte';
import {
	emptyExecutionStats,
	fromScoreboardRow
} from '$lib/scoreboard/extras/from-scoreboard-row';
```

Add derived values:

```ts
const executionStats = $derived(fromScoreboardRow(scoreboardResults));
const showExecutionStats = $derived(executionStats !== undefined);
const showCardContent = $derived(showContent || showExecutionStats);
```

Change `DashboardCard` prop: `content={showCardContent ? content : undefined}` (replace `showContent`).

- [ ] **Step 2: Update `content` snippet**

```svelte
{#snippet content()}
	<div class="space-y-3">
		{#if showExecutionStats && executionStats}
			{#if workflows && workflows.length > 0}
				<div class="space-y-3">
					<div class="flex items-center justify-between gap-2">
						<T class="shrink-0 text-sm font-medium">{m.Recent_workflows()}</T>
						<PipelineExecutionStats stats={executionStats} layout="inline" />
						<BlueButton
							compact
							href={resolve('/my/pipelines/[...pipeline_path]', {
								pipeline_path: getPath(pipeline, true)
							})}
						>
							{m.view_all()}
							<ArrowRightIcon />
						</BlueButton>
					</div>
					<Pipeline.Workflows.SmallTable {workflows} />
				</div>
			{:else}
				<PipelineExecutionStats stats={executionStats} layout="inline" />
			{/if}
		{:else if workflows && workflows.length > 0}
			<div class="space-y-3">
				<div class="flex items-center justify-between gap-1">
					<T class="text-sm font-medium">{m.Recent_workflows()}</T>
					<BlueButton compact href={...}>
						{m.view_all()}
						<ArrowRightIcon />
					</BlueButton>
				</div>
				<Pipeline.Workflows.SmallTable {workflows} />
			</div>
		{/if}
	</div>
{/snippet}
```

Use the existing `resolve` / `getPath` href from the current file for `BlueButton`.

**Layout note:** For the header row with three items, use `flex items-center justify-between gap-2` and give the stats `shrink-0` wrapper if needed so “View all” does not wrap awkwardly on narrow cards.

- [ ] **Step 3: Run check**

Run: `cd webapp && bun run check 2>&1 | head -40`

- [ ] **Step 4: Commit**

```bash
git add webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte
git commit -m "feat(pipelines): show scoreboard execution stats on pipeline cards"
```

---

### Task 5: Pipeline detail page loader

**Files:**
- Modify: `webapp/src/routes/my/pipelines/[...pipeline_path]/+page.ts`

- [ ] **Step 1: Load scoreboard in `load`**

```ts
import { Pipeline, Scoreboard } from '$lib';
```

After `pipeline` is resolved:

```ts
const scoreboard = await Scoreboard.Records.loadForPipeline(pipeline.id, { fetch });
```

Return `{ pipeline, workflows, pagination, scoreboard }`.

- [ ] **Step 2: Commit**

```bash
git add webapp/src/routes/my/pipelines/[...pipeline_path]/+page.ts
git commit -m "feat(pipelines): load scoreboard cache on pipeline detail page"
```

---

### Task 6: Pipeline detail page UI

**Files:**
- Modify: `webapp/src/routes/my/pipelines/[...pipeline_path]/+page.svelte`

- [ ] **Step 1: Remove paginated stat derivations**

Delete:

```ts
const nonCanceledWorkflows = $derived(...);
const totalRuns = $derived(...);
const totalSuccesses = $derived(...);
const successRate = $derived(...);
```

Delete `{#snippet numberBox ...}` if no longer used.

- [ ] **Step 2: Add scoreboard-backed stat boxes**

```ts
import PipelineExecutionStats from '$lib/scoreboard/extras/pipeline-execution-stats.svelte';
import {
	emptyExecutionStats,
	fromScoreboardRow
} from '$lib/scoreboard/extras/from-scoreboard-row';

let { data } = $props();
let { pipeline, pagination, scoreboard } = $derived(data);

const executionStats = $derived(
	fromScoreboardRow(scoreboard) ?? emptyExecutionStats
);
```

Replace the three `numberBox` calls with:

```svelte
<div class="flex flex-wrap gap-2 md:flex-nowrap">
	<PipelineExecutionStats
		stats={executionStats}
		layout="stat-box-success"
		label={m.scoreboard_success_rate()}
	/>
	<PipelineExecutionStats
		stats={executionStats}
		layout="stat-box-modes"
		label={m.Execution_mode()}
	/>
</div>
```

- [ ] **Step 3: Run check + lint**

Run:
```bash
cd webapp && bun run check 2>&1 | head -40
cd webapp && bun run lint 2>&1 | tail -20
```

- [ ] **Step 4: Commit**

```bash
git add webapp/src/routes/my/pipelines/[...pipeline_path]/+page.svelte
git commit -m "feat(pipelines): show lifetime execution stats on pipeline detail page"
```

---

### Task 7: Manual verification

- [ ] **Step 1: Card with cache + recent workflows** — stats between title and “View all”; numbers match scoreboard row for that pipeline.

- [ ] **Step 2: Card with cache, no recent workflows** — stats shown in content area without table.

- [ ] **Step 3: Card without cache** — no stats in content; `afterDescription` unchanged.

- [ ] **Step 4: Detail page** — two boxes; change status filter / page — stats unchanged.

- [ ] **Step 5: Scoreboard table** — success-rate column visually unchanged.

---

## Spec self-review (plan vs spec)

| Spec requirement | Task |
|------------------|------|
| Shared component + `fromScoreboardRow` | 1, 2 |
| Column refactor | 3 |
| Card rule B + placement | 4 |
| Detail loader + two boxes | 5, 6 |
| Zeros when no cache | 1 (`emptyExecutionStats`), 6 |
| No new i18n | all tasks use existing keys |
| Out of scope (hub, summary) | not in plan |

No placeholders remain in task steps.
