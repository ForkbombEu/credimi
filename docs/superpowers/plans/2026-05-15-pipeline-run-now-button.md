# Pipeline Run Now Button Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract pipeline “Run now” into `Pipeline.Runner.RunNowButton` and disable run with tooltip + `[Offline]` subtitle when the execution runner is known offline.

**Architecture:** Add `getExecutionRunnerPath` to `binding.ts` for global/specific runner resolution; new Svelte component owns run flow, modal, and offline UX using existing `Runners.status.isOnline`. Slim down `pipeline-card.svelte` to a single component tag.

**Tech Stack:** Svelte 5, Paraglide (`m.*`), Vitest, existing `Pipeline.Runners.status` coordinator, PocketBase types.

**Design spec:** `docs/superpowers/specs/2026-05-15-pipeline-run-now-button-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline/runner/binding.ts` | `getExecutionRunnerPath` helper |
| `webapp/src/lib/pipeline/runner/binding.test.ts` | Unit tests for path resolution |
| `webapp/src/lib/pipeline/runner/run-now-button.svelte` | Run UI, offline gate, modal |
| `webapp/src/lib/pipeline/runner/index.ts` | Export `RunNowButton` |
| `webapp/messages/en.json` | `Runner_offline_run_disabled` i18n key |
| `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte` | Consume `RunNowButton`, remove run logic |

---

### Task 1: `getExecutionRunnerPath` + unit tests

**Files:**
- Modify: `webapp/src/lib/pipeline/runner/binding.ts`
- Create: `webapp/src/lib/pipeline/runner/binding.test.ts`

- [ ] **Step 1: Write failing tests**

Create `binding.test.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';
import { beforeEach, describe, expect, it } from 'vitest';

import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

import * as Runner from './binding';

function pipeline(id: string, yaml: string): PipelinesResponse {
	return { id, yaml } as PipelinesResponse;
}

const runner = (path: string) =>
	({ id: 'runner-1', canonified_name: path.split('/').at(-1) } as MobileRunnersResponse);

const NO_MOBILE_YAML = `steps:
  - use: http-request
    id: step1`;

const GLOBAL_MOBILE_YAML = `steps:
  - use: mobile-automation
    id: ma1`;

const SPECIFIC_MOBILE_YAML = `steps:
  - use: mobile-automation
    id: ma1
    with:
      runner_id: org-a/my-runner`;

describe('getExecutionRunnerPath', () => {
	beforeEach(() => {
		localStorage.clear();
	});

	it('returns undefined when mobile-automation is not required', () => {
		expect(Runner.getExecutionRunnerPath(pipeline('p1', NO_MOBILE_YAML))).toBeUndefined();
	});

	it('returns undefined for global pipeline with no stored runner', () => {
		expect(Runner.getExecutionRunnerPath(pipeline('p2', GLOBAL_MOBILE_YAML))).toBeUndefined();
	});

	it('returns stored path for global pipeline with selected runner', () => {
		const p = pipeline('p3', GLOBAL_MOBILE_YAML);
		const r = runner('org-a/selected-runner');
		Runner.set(p, r);
		expect(Runner.getExecutionRunnerPath(p)).toBe(getPath(r));
	});

	it('returns runner_id from first mobile-automation step for specific pipeline', () => {
		expect(Runner.getExecutionRunnerPath(pipeline('p4', SPECIFIC_MOBILE_YAML))).toBe(
			'org-a/my-runner'
		);
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline/runner/binding.test.ts
```

Expected: FAIL — `getExecutionRunnerPath` is not exported.

- [ ] **Step 3: Implement `getExecutionRunnerPath`**

Add to `binding.ts` after `getType`:

```ts
export function getExecutionRunnerPath(pipeline: PipelinesResponse): string | undefined {
	const type = getType(pipeline);
	if (type === 'not-needed') return undefined;
	if (type === 'global') return get(pipeline.id);
	if (type === 'specific') {
		const yaml = parseYaml(pipeline.yaml);
		const step = (yaml?.steps ?? []).find((s) => s.use === 'mobile-automation');
		const runnerId = step && 'with' in step ? step.with?.runner_id : undefined;
		return typeof runnerId === 'string' ? runnerId : undefined;
	}
	return undefined;
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline/runner/binding.test.ts
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline/runner/binding.ts webapp/src/lib/pipeline/runner/binding.test.ts
git commit -m "feat(webapp): add getExecutionRunnerPath for pipeline run button"
```

---

### Task 2: i18n message

**Files:**
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: Add message key**

Near other runner keys (~line 921), add:

```json
"Runner_offline_run_disabled": "The selected runner is currently offline. To run this pipeline, please select a different runner.",
```

- [ ] **Step 2: Regenerate Paraglide messages**

```bash
cd webapp && bunx @inlang/paraglide-js compile --project ./project.inlang --outdir ./src/paraglide
```

(Or run `make generate` / project i18n target if that is the repo convention after editing `messages/en.json`.)

- [ ] **Step 3: Verify `m.Runner_offline_run_disabled` exists**

```bash
cd webapp && bun run check
```

Expected: no errors referencing missing message key.

- [ ] **Step 4: Commit**

```bash
git add webapp/messages/en.json webapp/src/paraglide webapp/project.inlang 2>/dev/null || true
git commit -m "feat(webapp): add offline runner tooltip message"
```

Include only generated i18n artifacts that actually changed.

---

### Task 3: `RunNowButton` component

**Files:**
- Create: `webapp/src/lib/pipeline/runner/run-now-button.svelte`
- Modify: `webapp/src/lib/pipeline/runner/index.ts`

- [ ] **Step 1: Export from index**

```ts
import RunNowButton from './run-now-button.svelte';
import SelectInput from './runner-select-input.svelte';
import SelectModal from './runner-select-modal.svelte';

export * from './binding';
export { RunNowButton, SelectInput, SelectModal };
```

- [ ] **Step 2: Create `run-now-button.svelte`**

Move logic/UI from `pipeline-card.svelte` (lines 48–75, 154–183, 198–205). Full script structure:

```svelte
<script lang="ts">
	import { Pipeline } from '$lib';
	import { getRecordByCanonifiedPath } from '$lib/canonify';
	import { getPath } from '$lib/utils';
	import { Cog, PlayIcon } from '@lucide/svelte';
	import { isError } from 'effect/Predicate';

	import type { MobileRunnersResponse, PipelinesResponse } from '@/pocketbase/types';

	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import Tooltip from '@/components/ui-custom/tooltip.svelte';
	import * as ButtonGroup from '@/components/ui/button-group';
	import { m } from '@/i18n';

	import * as Runners from '../runners';
	import * as Runner from './binding';
	import SelectModal from './runner-select-modal.svelte';

	type Props = {
		pipeline: PipelinesResponse;
		onRun?: () => void;
	};

	let { pipeline, onRun }: Props = $props();

	let runnerSelectionDialogOpen = $state(false);
	let runPipelineAfterRunnerSelect = $state(false);

	const runnerType = $derived(Runner.getType(pipeline));
	const isRunnerSpecific = $derived(runnerType === 'specific');
	const executionPath = $derived(Runner.getExecutionRunnerPath(pipeline));
	const isRunnerOffline = $derived(
		executionPath !== undefined && Runners.status.isOnline(executionPath) === false
	);

	const runnerLabel = $derived.by(() => {
		const path = executionPath ?? Runner.get(pipeline.id);
		if (!path || !Runner.isRequired(pipeline)) return undefined;
		const name = path.split('/').at(-1);
		return isRunnerOffline ? `[Offline] ${name}` : name;
	});

	$effect(() => {
		const path = executionPath;
		if (!path || Runners.status.isOnline(path) !== undefined) return;

		const fromStore = Runners.store.read().find((r) => getPath(r) === path);

		if (fromStore) {
			Runners.status.probe([fromStore], { reason: 'visible' });
			return;
		}

		let cancelled = false;
		void getRecordByCanonifiedPath<MobileRunnersResponse>(path)
			.then((res) => {
				if (cancelled || isError(res)) return;
				Runners.status.probe([res], { reason: 'visible' });
			})
			.catch(console.error);

		return () => {
			cancelled = true;
		};
	});

	async function handleRunNow() {
		if (isRunnerOffline) return;

		if (!Runner.isRequired(pipeline)) {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (runnerType === 'specific') {
			await Pipeline.run(pipeline);
			onRun?.();
			return;
		}

		if (Runner.get(pipeline.id)) {
			await Pipeline.run(pipeline);
			onRun?.();
			runPipelineAfterRunnerSelect = false;
			return;
		}

		runPipelineAfterRunnerSelect = true;
		runnerSelectionDialogOpen = true;
	}
</script>

{#snippet runButtonGroup()}
	<ButtonGroup.Root>
		<Button
			onclick={handleRunNow}
			disabled={isRunnerOffline}
			class={{ 'w-[174px] justify-start': !Runner.isRequired(pipeline) }}
		>
			<PlayIcon />
			<div class="flex w-[90px] flex-col -space-y-0.5 text-left">
				<p>{m.Run_now()}</p>
				{#if runnerLabel}
					<small class="truncate text-[9px] opacity-80">{runnerLabel}</small>
				{/if}
			</div>
		</Button>
		{#if Runner.isRequired(pipeline)}
			<IconButton
				icon={Cog}
				variant="default"
				class="rounded-none rounded-r-md border-l border-l-slate-500"
				onclick={() => (runnerSelectionDialogOpen = true)}
				disabled={isRunnerSpecific}
				tooltip={isRunnerSpecific
					? m.Runner_configuration_not_available()
					: m.Configure_runner()}
			/>
		{/if}
	</ButtonGroup.Root>
{/snippet}

{#if isRunnerOffline}
	<Tooltip>
		<span class="inline-flex">
			{@render runButtonGroup()}
		</span>
		{#snippet content()}
			<p>{m.Runner_offline_run_disabled()}</p>
		{/snippet}
	</Tooltip>
{:else}
	{@render runButtonGroup()}
{/if}

<SelectModal
	{pipeline}
	bind:open={runnerSelectionDialogOpen}
	onSelect={() => {
		if (!runPipelineAfterRunnerSelect) return;
		void handleRunNow();
	}}
/>
```

**Fix before commit:** Replace erroneous `</motion.div>` with `</motion.div>` → `</motion.div>` should be `</motion.div>` — actually use `</div>` (typo in plan snippet). Use `getPath` from `$lib/utils` in the `$effect` store lookup (not `Runner.getPath`).

- [ ] **Step 3: Run Svelte check + autofixer**

```bash
cd webapp && bun run check
```

Use Svelte MCP `svelte-autofixer` on `run-now-button.svelte` if available.

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline/runner/run-now-button.svelte webapp/src/lib/pipeline/runner/index.ts
git commit -m "feat(webapp): add RunNowButton with offline runner gate"
```

---

### Task 4: Wire into `pipeline-card.svelte`

**Files:**
- Modify: `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte`

- [ ] **Step 1: Remove run-related code**

Delete:
- Imports: `PlayIcon`, `Cog`, `Button`, `ButtonGroup`, run-only state
- `runnerSelectionDialogOpen`, `runPipelineAfterRunnerSelect`, `runnerType`, `isRunnerSpecific`, `handleRunNow`
- `actions` snippet ButtonGroup block (lines 155–183)
- `<Pipeline.Runner.SelectModal ...>` block (lines 198–205)

- [ ] **Step 2: Add `RunNowButton` in actions**

```svelte
{#snippet actions()}
	<Pipeline.Runner.RunNowButton {pipeline} {onRun} />

	{#if !schedule}
		<SchedulePipelineForm {pipeline} />
	{:else}
		...
	{/if}
{/snippet}
```

- [ ] **Step 3: Verify**

```bash
cd webapp && bun run check && bun run lint
```

- [ ] **Step 4: Commit**

```bash
git add webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte
git commit -m "refactor(webapp): use RunNowButton in pipeline card"
```

---

### Task 5: Final verification

- [ ] **Step 1: Run unit tests**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline/runner/binding.test.ts
```

Expected: PASS.

- [ ] **Step 2: Manual UAT** (dev stack running: `make dev`)

On `http://localhost:8090/my/pipelines` (or proxied UI port):

1. Global pipeline + offline selected runner → Run disabled, tooltip, `[Offline]` subtitle.
2. Global pipeline + no runner → Run enabled, opens modal.
3. Specific pipeline + offline YAML runner → same disable UX.
4. Runner back online after ~30s poll → Run enabled, subtitle without `[Offline]`.

- [ ] **Step 3: Commit plan completion note** (optional — skip if no further file changes)

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `getExecutionRunnerPath` | Task 1 |
| Offline disable when `isOnline === false` | Task 3 |
| Probe when status unknown | Task 3 `$effect` |
| Tooltip + i18n | Task 2, Task 3 |
| `[Offline]` subtitle | Task 3 `runnerLabel` |
| Global no-runner opens modal | Task 3 `handleRunNow` |
| `pipeline-card` slimmed | Task 4 |
| Unit tests | Task 1 |
| Manual UAT | Task 5 |

## Out of scope (do not implement)

- E2E tests
- Disable while status `undefined`
- Force runner selection before first global run
