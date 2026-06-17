# Remove Manual Pipeline Routes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete `/new/manual` and `/edit/manual` routes; auto-open locked inline manual mode for `manual: true` pipelines and enrichment failures on the unified `/edit` page.

**Architecture:** Extend `StepsBuilder.enterManualMode` with `{ locked?: boolean }` and `isManualLocked`. `PipelineForm` constructor auto-enters locked manual mode when `pipeline.record.manual` or `startLockedManual` is set. `edit/+page.ts` stops redirecting and returns minimal pipeline data for manual/fallback cases.

**Tech Stack:** Svelte 5 runes, SvelteKit load functions, Vitest, Paraglide (`m.*`).

**Design spec:** `docs/superpowers/specs/2026-06-17-remove-manual-routes-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts` | `manualLocked` state, `enterManualMode` options, locked `exitManualMode` |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts` | Locked manual mode unit tests |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte` | Hide **Back to steps** when locked |
| `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts` | `startLockedManual` prop, auto-init, remove `manualEditHref` |
| `webapp/src/lib/pipeline-form/pipeline-form.test.ts` | Constructor auto-init tests |
| `webapp/src/lib/pipeline-form/pipeline-form.svelte` | Remove top-bar **Manual mode** button |
| `webapp/src/routes/my/pipelines/(group)/[...path]/edit/+page.ts` | Load minimal pipeline + `startLockedManual` |
| `webapp/src/routes/my/pipelines/(group)/[...path]/edit/+page.svelte` | Pass `startLockedManual` to `PipelineForm` |
| `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte` | Always link to `/edit` |
| `webapp/src/lib/pipeline/utils.ts` | Remove `getManualEditHref` |
| Delete: `webapp/src/routes/my/pipelines/(group)/new/manual/` | Old create-manual route |
| Delete: `webapp/src/routes/my/pipelines/(group)/[...path]/edit/manual/` | Old edit-manual route |
| Delete: `webapp/src/lib/pipeline-form/pipeline-form-manual.svelte` | Full-page manual form |

---

### Task 1: `StepsBuilder` locked manual mode

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts`

- [ ] **Step 1: Write failing tests**

Add to `steps-builder.test.ts`:

```ts
it('enterManualMode with locked sets isManualLocked', () => {
	const builder = createBuilder();
	builder.enterManualMode(VALID_YAML, { locked: true });

	expect(builder.isManualLocked).toBe(true);
	expect(builder.isManualMode).toBe(true);
	if (builder.mode.id === 'manual') builder.mode.editor.dispose();
});

it('exitManualMode is no-op when locked', async () => {
	const builder = createBuilder();
	builder.enterManualMode(VALID_YAML, { locked: true });
	if (builder.mode.id !== 'manual') throw new Error('expected manual mode');
	builder.mode.editor.yaml = `${VALID_YAML}\n`;

	const confirm = vi.fn();
	vi.stubGlobal('confirm', confirm);

	const ok = await builder.exitManualMode();

	expect(ok).toBe(true);
	expect(confirm).not.toHaveBeenCalled();
	expect(builder.mode.id).toBe('manual');
	expect(builder.isManualLocked).toBe(true);
	if (builder.mode.id === 'manual') builder.mode.editor.dispose();
});

it('exitManualMode clears manualLocked when unlocked', async () => {
	const builder = createBuilder();
	builder.enterManualMode(VALID_YAML);

	const ok = await builder.exitManualMode();

	expect(ok).toBe(true);
	expect(builder.isManualLocked).toBe(false);
	expect(builder.mode.id).toBe('idle');
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/steps-builder.test.ts --run`  
Expected: FAIL — `isManualLocked` undefined / wrong signature on `enterManualMode`

- [ ] **Step 3: Implement locked manual mode**

In `steps-builder.svelte.ts`, extend `State`:

```ts
type State = {
	steps: EnrichedStep[];
	mode: BuilderMode;
	manualLocked: boolean;
};
```

Initialize in constructor / default state:

```ts
private state = $state<State>({
	steps: [],
	mode: { id: 'idle' },
	manualLocked: false
});
```

Add getter:

```ts
get isManualLocked() {
	return this.state.manualLocked;
}
```

Update `enterManualMode`:

```ts
enterManualMode(initialYaml: string, options?: { locked?: boolean }) {
	if (this.state.mode.id === 'form') {
		this.exitFormState();
	}
	const editor = new InlineManualEditor(initialYaml);
	this.stateManager.run((state) => {
		state.mode = { id: 'manual', editor };
		state.manualLocked = options?.locked ?? false;
	});
	void editor.validateNow();
}
```

Update `exitManualMode`:

```ts
async exitManualMode(): Promise<boolean> {
	if (this.state.mode.id !== 'manual') return true;
	if (this.state.manualLocked) return true;

	const { editor } = this.state.mode;
	if (editor.isDirty) {
		const confirmed = confirm(
			m.discard_manual_yaml_changes() + '\n' + m.Are_you_sure_you_want_to_exit_the_form()
		);
		if (!confirmed) return false;
	}
	editor.dispose();
	this.stateManager.run((state) => {
		state.mode = { id: 'idle' };
		state.manualLocked = false;
	});
	return true;
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/steps-builder.test.ts --run`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts
git commit -m "feat(pipeline-form): add locked manual mode to StepsBuilder"
```

---

### Task 2: Hide **Back to steps** when locked

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte`

- [ ] **Step 1: Gate Back to steps button**

In `steps-builder.svelte`, change the `titleRight` snippet for the right column:

```svelte
{#snippet titleRight()}
	{#if builder.mode.id === 'manual' && !builder.isManualLocked}
		<Button variant="outline" size="sm" onclick={() => void builder.exitManualMode()}>
			<BlocksIcon />
			{m.back_to_steps()}
		</Button>
	{:else if !builder.isManualMode}
		<YamlPreviewMenu {builder} initialYaml={builder.yamlPreview} />
	{/if}
{/snippet}
```

- [ ] **Step 2: Run check**

Run: `cd webapp && bun run check`  
Expected: no new errors in `steps-builder.svelte`

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte
git commit -m "feat(pipeline-form): hide Back to steps when manual mode is locked"
```

---

### Task 3: `PipelineForm` auto-init locked manual mode

**Files:**
- Create: `webapp/src/lib/pipeline-form/pipeline-form.test.ts`
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts`

- [ ] **Step 1: Write failing tests**

Create `pipeline-form.test.ts`:

```ts
import { afterEach, describe, expect, it, vi } from 'vitest';

vi.mock('$app/navigation', () => ({ beforeNavigate: vi.fn() }));
vi.mock('$lib', async () => {
	const { validateYaml } = await import('$lib/pipeline/validate-yaml');
	return { Pipeline: { validateYaml } };
});
vi.mock('./pipeline-form.svelte', () => ({ default: class {} }));
vi.mock('./steps-builder/steps-builder.svelte', () => ({ default: class {} }));
vi.mock('./metadata-form/metadata-form.svelte', () => ({ default: class {} }));
vi.mock('./runtime-options-form/runtime-options-form.svelte', () => ({ default: class {} }));
vi.mock('./execution-target/index.js', () => ({
	ExecutionTarget: { loadFromPipeline: vi.fn(), clear: vi.fn() }
}));

import { PipelineForm } from './pipeline-form.svelte.js';

const STORED_YAML = `name: manual-pipeline

steps:
  - use: debug
`;

const record = {
	id: 'rec1',
	name: 'manual-pipeline',
	description: '',
	yaml: STORED_YAML,
	manual: true
} as never;

describe('PipelineForm locked manual init', () => {
	afterEach(() => {
		if (PipelineForm.prototype) {
			/* dispose editors if needed */
		}
	});

	it('auto-enters locked manual mode when pipeline.manual is true', () => {
		const form = new PipelineForm({
			mode: 'edit',
			pipeline: { record, steps: [], runtime: undefined }
		});

		expect(form.stepsBuilder.isManualMode).toBe(true);
		expect(form.stepsBuilder.isManualLocked).toBe(true);
		if (form.stepsBuilder.mode.id === 'manual') {
			expect(form.stepsBuilder.mode.editor.yaml).toBe(STORED_YAML);
			form.stepsBuilder.mode.editor.dispose();
		}
	});

	it('auto-enters locked manual mode when startLockedManual is true', () => {
		const form = new PipelineForm({
			mode: 'edit',
			pipeline: { record: { ...record, manual: false }, steps: [], runtime: undefined },
			startLockedManual: true
		});

		expect(form.stepsBuilder.isManualMode).toBe(true);
		expect(form.stepsBuilder.isManualLocked).toBe(true);
		if (form.stepsBuilder.mode.id === 'manual') {
			expect(form.stepsBuilder.mode.editor.yaml).toBe(STORED_YAML);
			form.stepsBuilder.mode.editor.dispose();
		}
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/pipeline-form.test.ts --run`  
Expected: FAIL — `startLockedManual` not in Props / no auto-init

- [ ] **Step 3: Implement auto-init and remove `manualEditHref`**

In `pipeline-form.svelte.ts`:

Extend `Props`:

```ts
type Props = {
	mode: 'create' | 'edit';
	pipeline?: EnrichedPipeline;
	startLockedManual?: boolean;
};
```

Remove `manualEditHref` derived entirely.

After `this.stepsBuilder = new StepsBuilder({...})` in constructor, add:

```ts
const shouldStartLockedManual =
	props.pipeline?.record.manual === true || props.startLockedManual === true;

if (shouldStartLockedManual && props.pipeline?.record.yaml) {
	this.stepsBuilder.enterManualMode(props.pipeline.record.yaml, { locked: true });
}
```

Remove unused `Pipeline` import if it was only used for `getManualEditHref`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/pipeline-form.test.ts --run`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/pipeline-form.svelte.ts webapp/src/lib/pipeline-form/pipeline-form.test.ts
git commit -m "feat(pipeline-form): auto-init locked manual mode on load"
```

---

### Task 4: Edit page load — stop redirecting

**Files:**
- Modify: `webapp/src/routes/my/pipelines/(group)/[...path]/edit/+page.ts`
- Modify: `webapp/src/routes/my/pipelines/(group)/[...path]/edit/+page.svelte`

- [ ] **Step 1: Rewrite `+page.ts` load**

Replace entire file:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getEnrichedPipeline } from '$lib/pipeline-form/functions';

import type { EnrichedPipeline } from '$lib/pipeline-form/functions.js';

//

function minimalPipeline(record: EnrichedPipeline['record']): EnrichedPipeline {
	return { record, steps: [], runtime: undefined };
}

export const load = async ({ fetch, parent }) => {
	const { pipeline: record } = await parent();

	if (record.manual) {
		return {
			pipeline: minimalPipeline(record),
			startLockedManual: true as const
		};
	}

	try {
		const enriched = await getEnrichedPipeline(record.id, { fetch });
		return { pipeline: enriched };
	} catch {
		return {
			pipeline: minimalPipeline(record),
			startLockedManual: true as const
		};
	}
};
```

- [ ] **Step 2: Pass `startLockedManual` in `+page.svelte`**

```svelte
const { data } = $props();
const form = new PipelineForm({
	mode: 'edit',
	pipeline: data.pipeline,
	startLockedManual: data.startLockedManual
});
```

- [ ] **Step 3: Run check**

Run: `cd webapp && bun run check`  
Expected: PASS (no type errors on load return shape)

- [ ] **Step 4: Commit**

```bash
git add webapp/src/routes/my/pipelines/(group)/[...path]/edit/+page.ts webapp/src/routes/my/pipelines/(group)/[...path]/edit/+page.svelte
git commit -m "feat(pipelines): load edit page with inline locked manual fallback"
```

---

### Task 5: Remove top-bar **Manual mode** link

**Files:**
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.svelte`

- [ ] **Step 1: Remove Manual mode button and unused imports**

Delete the `PencilIcon` import if unused.

Remove from `topbarRight` snippet:

```svelte
<Button href={form.manualEditHref} variant="ghost">
	<PencilIcon />
	{m.manual_mode()}
</Button>
```

- [ ] **Step 2: Run check**

Run: `cd webapp && bun run check`  
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/pipeline-form.svelte
git commit -m "refactor(pipeline-form): remove top-bar Manual mode navigation link"
```

---

### Task 6: Simplify pipeline card edit link

**Files:**
- Modify: `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte`

- [ ] **Step 1: Always link to `/edit`**

Replace `editAction` href:

```svelte
<IconButton
	href={resolve('/my/pipelines/(group)/[...path]/edit', {
		path: getPath(pipeline, true)
	})}
	icon={Pencil}
	tooltip={pipeline.published ? m.pipeline_edit_disabled_while_published() : m.Edit()}
	disabled={pipeline.published}
/>
```

- [ ] **Step 2: Commit**

```bash
git add webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte
git commit -m "refactor(pipelines): pipeline card always links to unified edit route"
```

---

### Task 7: Delete manual routes and dead code

**Files:**
- Delete: `webapp/src/routes/my/pipelines/(group)/new/manual/+page.svelte`
- Delete: `webapp/src/routes/my/pipelines/(group)/[...path]/edit/manual/+page.svelte`
- Delete: `webapp/src/lib/pipeline-form/pipeline-form-manual.svelte`
- Modify: `webapp/src/lib/pipeline/utils.ts`

- [ ] **Step 1: Delete route files and `pipeline-form-manual.svelte`**

```bash
rm -rf webapp/src/routes/my/pipelines/\(group\)/new/manual
rm -rf webapp/src/routes/my/pipelines/\(group\)/\[...path\]/edit/manual
rm webapp/src/lib/pipeline-form/pipeline-form-manual.svelte
```

- [ ] **Step 2: Remove `getManualEditHref` from utils**

`webapp/src/lib/pipeline/utils.ts` should contain only:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { parse } from 'yaml';

import type { Pipeline } from './types';

//

export function parseYaml(yaml: string): Pipeline {
	return parse(yaml) as Pipeline;
}
```

Remove unused `getPath` and `PipelinesResponse` imports.

- [ ] **Step 3: Grep for remaining references**

Run: `rg "getManualEditHref|pipeline-form-manual|/manual" webapp/src`  
Expected: no matches (except unrelated strings like `manual_conformance`)

- [ ] **Step 4: Run check and tests**

Run: `cd webapp && bun run check && bun run test:unit -- --run`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add -A webapp/src/routes/my/pipelines webapp/src/lib/pipeline-form/pipeline-form-manual.svelte webapp/src/lib/pipeline/utils.ts
git commit -m "refactor(pipelines): remove manual routes and full-page manual form"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run lint and typecheck**

Run: `cd webapp && bun run lint && bun run check`  
Expected: PASS

- [ ] **Step 2: Run unit tests**

Run: `cd webapp && bun run test:unit -- --run`  
Expected: PASS

- [ ] **Step 3: Manual smoke test**

1. Open `/my/pipelines/new` → YAML ⋮ → **Edit manually** → **Back to steps** visible.
2. Save with valid YAML → pipeline has `manual: true`.
3. Re-open edit → locked manual mode, no **Back to steps**.
4. Edit a blocks pipeline → **Edit manually** → **Back to steps** works.
5. Top bar has no **Manual mode** link.
6. `/my/pipelines/new/manual` and `.../edit/manual` return 404.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Delete `/new/manual` and `/edit/manual` | Task 7 |
| Delete `pipeline-form-manual.svelte` | Task 7 |
| Remove `getManualEditHref` | Task 7 |
| Remove top-bar **Manual mode** link | Task 5 |
| `pipeline-card` always `/edit` | Task 6 |
| `manual: true` skip enrichment | Task 4 |
| Enrichment failure → locked manual | Task 4 |
| `enterManualMode` with `locked` option | Task 1 |
| `exitManualMode` no-op when locked | Task 1 |
| Hide **Back to steps** when locked | Task 2 |
| `PipelineForm` auto-init | Task 3 |
| Unit tests StepsBuilder locked | Task 1 |
| Unit tests PipelineForm init | Task 3 |
| No legacy URL redirects | Task 7 (routes deleted) |

---

## Out of scope (per spec)

- Redirects for legacy `/manual` URLs
- YAML → steps rehydration
- E2E tests
- Removing `blocks_mode` i18n key
