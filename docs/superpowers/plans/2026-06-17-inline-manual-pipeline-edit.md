# Inline Manual Pipeline Edit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add inline manual YAML editing inside the blocks pipeline composer, with debounced validation, disabled-but-scrollable side columns, and save semantics matching the full manual editor (`manual: true`).

**Architecture:** Extend `StepsBuilder` mode with `{ id: 'manual'; editor: InlineManualEditor }`. `InlineManualEditor` uses Runed `Debounced(() => Pipeline.validateYaml(this.yaml))`. Shared validation lives on `$lib/pipeline` as `Pipeline.validateYaml`. `PipelineForm` branches save/`hasChanges`/`canSave` when in manual mode.

**Tech Stack:** Svelte 5 runes, Runed `Debounced`, CodeMirror via `codeEditor.svelte`, AJV + `pipeline_schema.json`, Paraglide (`m.*`), Vitest.

**Design spec:** `docs/superpowers/specs/2026-06-17-inline-manual-pipeline-edit-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline/validate-yaml.ts` | `PipelineYamlValidation`, `validateYaml()` |
| `webapp/src/lib/pipeline/validate-yaml.test.ts` | Unit tests |
| `webapp/src/lib/pipeline/index.ts` | Re-export `validateYaml`, `PipelineYamlValidation` |
| `webapp/src/lib/pipeline-form/pipeline-form-manual.svelte` | Use `Pipeline.validateYaml` in Zod refine |
| `webapp/src/lib/pipeline-form/steps-builder/inline-manual-editor.svelte.ts` | `InlineManualEditor` class |
| `webapp/src/lib/pipeline-form/steps-builder/inline-manual-editor.test.ts` | Editor unit tests |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts` | `enterManualMode` / `exitManualMode`, mode type |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts` | Builder manual mode tests |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte` | Column layout swap, disabled overlays |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/column.svelte` | Optional `disabled` overlay prop |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/yaml-preview-menu.svelte` | Ellipsis dropdown → Edit manually |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/manual-editor-column.svelte` | CodeEditor, sticky error, Back to steps |
| `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts` | Manual save / hasChanges / canSave / validateExit |
| `webapp/src/lib/pipeline-form/pipeline-form.svelte` | Disabled top-bar controls + tooltips |
| `webapp/src/lib/pipeline-form/metadata-form/metadata-form.svelte.ts` | Optional `disabled` getter |
| `webapp/src/lib/pipeline-form/metadata-form/metadata-form.svelte` | Pass `disabled` to trigger Button |
| `webapp/src/lib/pipeline-form/runtime-options-form/runtime-options-form.svelte.ts` | Optional `disabled` getter |
| `webapp/src/lib/pipeline-form/runtime-options-form/runtime-options-form.svelte` | Pass `disabled` to trigger Button |
| `webapp/messages/en.json` (+ other locales) | New i18n keys |

---

### Task 1: `Pipeline.validateYaml`

**Files:**
- Create: `webapp/src/lib/pipeline/validate-yaml.ts`
- Create: `webapp/src/lib/pipeline/validate-yaml.test.ts`
- Modify: `webapp/src/lib/pipeline/index.ts`

- [ ] **Step 1: Write failing tests**

```ts
// validate-yaml.test.ts
import { describe, expect, it } from 'vitest';

import { validateYaml } from './validate-yaml';

const VALID_YAML = `name: test-pipeline

steps:
  - use: debug
`;

describe('validateYaml', () => {
	it('returns ok with value for valid pipeline yaml', () => {
		const result = validateYaml(VALID_YAML);
		expect(result).toEqual({ ok: true, value: VALID_YAML });
	});

	it('returns error for malformed yaml', () => {
		const result = validateYaml('name: [\n');
		expect(result.ok).toBe(false);
		if (!result.ok) expect(result.message).toMatch(/Invalid YAML/i);
	});

	it('returns error for schema violation', () => {
		const result = validateYaml('name: test\nsteps: []\n');
		expect(result.ok).toBe(false);
		if (!result.ok) expect(result.message.length).toBeGreaterThan(0);
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/validate-yaml.test.ts --run`  
Expected: FAIL — module not found

- [ ] **Step 3: Implement `validateYaml`**

```ts
// validate-yaml.ts
import PipelineSchema from '$root/schemas/pipeline/pipeline_schema.json';
import Ajv from 'ajv/dist/2020';
import { parse as parseYaml } from 'yaml';

import { getExceptionMessage } from '@/utils/errors';

export type PipelineYamlValidation =
	| { ok: true; value: string }
	| { ok: false; message: string };

const ajv = new Ajv({ allowUnionTypes: true, dynamicRef: true });
const validatePipeline = ajv.compile(PipelineSchema);

export function validateYaml(yaml: string): PipelineYamlValidation {
	let parsed: unknown;
	try {
		parsed = parseYaml(yaml);
	} catch (e) {
		return { ok: false, message: `Invalid YAML document: ${getExceptionMessage(e)}` };
	}

	if (!validatePipeline(parsed)) {
		return { ok: false, message: `Invalid YAML document: ${ajv.errorsText(validatePipeline.errors)}` };
	}

	return { ok: true, value: yaml };
}
```

- [ ] **Step 4: Export from pipeline index**

```ts
// index.ts — add:
export { validateYaml, type PipelineYamlValidation } from './validate-yaml';
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/validate-yaml.test.ts --run`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline/validate-yaml.ts webapp/src/lib/pipeline/validate-yaml.test.ts webapp/src/lib/pipeline/index.ts
git commit -m "feat(pipeline): add Pipeline.validateYaml for schema validation"
```

---

### Task 2: Refactor full manual route to use `Pipeline.validateYaml`

**Files:**
- Modify: `webapp/src/lib/pipeline-form/pipeline-form-manual.svelte`

- [ ] **Step 1: Replace inline AJV with `Pipeline.validateYaml`**

Remove local `ajv`, `validatePipeline`, `parseYaml` imports. Add:

```ts
import { Pipeline } from '$lib';
```

Replace `refineAsPipelineYaml` body:

```ts
function refineAsPipelineYaml(schema: z.ZodString | z.ZodOptional<z.ZodString>) {
	return schema.superRefine((v, ctx) => {
		if (!v) return;
		const result = Pipeline.validateYaml(v);
		if (!result.ok) {
			ctx.addIssue({ code: z.ZodIssueCode.custom, message: result.message });
		}
	});
}
```

- [ ] **Step 2: Smoke-check types**

Run: `cd webapp && bun run check`  
Expected: no new errors in `pipeline-form-manual.svelte`

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/pipeline-form-manual.svelte
git commit -m "refactor(pipeline-form): use Pipeline.validateYaml in manual route"
```

---

### Task 3: `InlineManualEditor` class

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps-builder/inline-manual-editor.svelte.ts`
- Create: `webapp/src/lib/pipeline-form/steps-builder/inline-manual-editor.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
import { describe, expect, it, vi } from 'vitest';

import { InlineManualEditor } from './inline-manual-editor.svelte.js';

const VALID_YAML = `name: test

steps:
  - use: debug
`;

describe('InlineManualEditor', () => {
	it('tracks dirty state', () => {
		const editor = new InlineManualEditor(VALID_YAML);
		expect(editor.isDirty).toBe(false);
		editor.yaml = `${VALID_YAML}\n`;
		expect(editor.isDirty).toBe(true);
		editor.dispose();
	});

	it('validateNow flushes debounce and returns validation', async () => {
		const editor = new InlineManualEditor(VALID_YAML);
		const result = await editor.validateNow();
		expect(result.ok).toBe(true);
		if (result.ok) expect(result.value).toBe(VALID_YAML);
		editor.dispose();
	});

	it('dispose cancels without throwing', () => {
		const editor = new InlineManualEditor(VALID_YAML);
		editor.yaml = 'broken: [';
		expect(() => editor.dispose()).not.toThrow();
	});
});
```

Note: Svelte 5 rune classes need the `.svelte.ts` extension; adjust import path if vitest resolves `.svelte.ts` directly.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/inline-manual-editor.test.ts --run`  
Expected: FAIL

- [ ] **Step 3: Implement `InlineManualEditor`**

```ts
// inline-manual-editor.svelte.ts
import { Pipeline } from '$lib';
import { Debounced } from 'runed';

export class InlineManualEditor {
	yaml = $state('');
	readonly baselineYaml: string;

	private debouncedValidation = new Debounced(
		() => Pipeline.validateYaml(this.yaml),
		400
	);

	constructor(initialYaml: string) {
		this.yaml = initialYaml;
		this.baselineYaml = initialYaml;
	}

	get validation() {
		return this.debouncedValidation.current;
	}

	get isDirty() {
		return this.yaml !== this.baselineYaml;
	}

	get isValid() {
		return this.debouncedValidation.current.ok;
	}

	async validateNow() {
		await this.debouncedValidation.updateImmediately();
		return this.debouncedValidation.current;
	}

	dispose() {
		this.debouncedValidation.cancel();
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/inline-manual-editor.test.ts --run`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/inline-manual-editor.svelte.ts webapp/src/lib/pipeline-form/steps-builder/inline-manual-editor.test.ts
git commit -m "feat(pipeline-form): add InlineManualEditor with debounced validation"
```

---

### Task 4: `StepsBuilder` manual mode API

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts`
- Create or modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts`

- [ ] **Step 1: Extend `BuilderMode` and add methods**

```ts
import { InlineManualEditor } from './inline-manual-editor.svelte.js';
import { m } from '@/i18n';

type BuilderMode =
	| { id: 'idle' }
	| { id: 'form'; ... }
	| { id: 'manual'; editor: InlineManualEditor };

get isManualMode() {
	return this.state.mode.id === 'manual';
}

enterManualMode(initialYaml: string) {
	if (this.state.mode.id === 'form') {
		this.exitFormState();
	}
	const editor = new InlineManualEditor(initialYaml);
	this.stateManager.run((state) => {
		state.mode = { id: 'manual', editor };
	});
	void editor.validateNow();
}

async exitManualMode(): Promise<boolean> {
	if (this.state.mode.id !== 'manual') return true;
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
	});
	return true;
}
```

- [ ] **Step 2: Write builder tests**

Test `enterManualMode` closes form mode and sets `manual`. Test `exitManualMode` returns to `idle` when not dirty. Mock `confirm` for dirty exit.

- [ ] **Step 3: Run tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/steps-builder.test.ts --run`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts
git commit -m "feat(pipeline-form): add StepsBuilder manual mode enter/exit"
```

---

### Task 5: Column disabled overlay + partials

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/_partials/column.svelte`
- Create: `webapp/src/lib/pipeline-form/steps-builder/_partials/yaml-preview-menu.svelte`
- Create: `webapp/src/lib/pipeline-form/steps-builder/_partials/manual-editor-column.svelte`

- [ ] **Step 1: Add `disabled` prop to `column.svelte`**

```svelte
type Props = {
	// ...existing
	disabled?: boolean;
};

let { ..., disabled = false }: Props = $props();
```

Wrap content area:

```svelte
<div class={['relative flex grow flex-col overflow-y-scroll', contentClass]}>
	{#if disabled}
		<div class="absolute inset-0 z-10 bg-white/40" aria-hidden="true"></div>
	{/if}
	<div class={disabled ? 'opacity-60' : ''}>
		{@render children?.()}
	</div>
</div>
```

- [ ] **Step 2: Create `yaml-preview-menu.svelte`**

```svelte
<script lang="ts">
	import { EllipsisIcon, PencilIcon } from '@lucide/svelte';
	import DropdownMenu from '@/components/ui-custom/dropdown-menu.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';
	import type { StepsBuilder } from '../steps-builder.svelte.js';

	type Props = {
		builder: StepsBuilder;
		initialYaml: string;
	};
	let { builder, initialYaml }: Props = $props();
</script>

<DropdownMenu
	items={[{ label: m.edit_manually(), icon: PencilIcon, onclick: () => builder.enterManualMode(initialYaml) }]}
>
	{#snippet trigger({ props })}
		<IconButton {...props} icon={EllipsisIcon} size="xs" variant="ghost" />
	{/snippet}
</DropdownMenu>
```

- [ ] **Step 3: Create `manual-editor-column.svelte`**

```svelte
<script lang="ts">
	import { BlocksIcon } from '@lucide/svelte';
	import CodeEditor from '@/components/ui-custom/codeEditor.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';
	import type { InlineManualEditor } from '../inline-manual-editor.svelte.js';
	import type { StepsBuilder } from '../steps-builder.svelte.js';
	import Column from './column.svelte';

	type Props = { builder: StepsBuilder; editor: InlineManualEditor };
	let { builder, editor }: Props = $props();
</script>

<Column title={m.manual_edit()} class="card basis-2 min-w-0 overflow-hidden">
	{#snippet titleRight()}
		<Button variant="outline" size="sm" onclick={() => void builder.exitManualMode()}>
			<BlocksIcon />
			{m.back_to_steps()}
		</Button>
	{/snippet}

	<div class="relative flex min-h-0 min-w-0 grow flex-col">
		<CodeEditor
			lang="yaml"
			bind:value={editor.yaml}
			minHeight={null}
			class="min-h-0 grow rounded-none"
			hideCopyButton
		/>
		{#if !editor.validation.ok}
			<div class="sticky bottom-0 border-t bg-destructive/10 px-4 py-2 text-sm text-destructive">
				{editor.validation.message}
			</div>
		{/if}
	</div>
</Column>
```

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/_partials/
git commit -m "feat(pipeline-form): add manual editor column and yaml preview menu partials"
```

---

### Task 6: `steps-builder.svelte` layout

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte`

- [ ] **Step 1: Wire manual mode layout**

- Pass `yamlPreview` string from parent via new prop or use `builder.yamlPreview` + pass to menu.
- First two columns: `disabled={builder.isManualMode}`.
- Third column: `{#if builder.isManualMode}` → `<ManualEditorColumn />` `{:else}` → existing YAML preview + `<YamlPreviewMenu />` in `titleRight`.
- Add `min-w-0` on manual column pane for flex overflow (CodeEditor warning).

- [ ] **Step 2: Run Svelte autofixer / check**

Run: `cd webapp && bun run check`  
Run svelte-autofixer on changed `.svelte` files.

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte
git commit -m "feat(pipeline-form): swap yaml preview for manual editor column"
```

---

### Task 7: `PipelineForm` save and change detection

**Files:**
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts`

- [ ] **Step 1: Branch `save()` for manual mode**

```ts
async save() {
	// ... metadata gate unchanged ...

	let yaml: string;
	let manual = false;

	if (this.stepsBuilder.isManualMode && this.stepsBuilder.mode.id === 'manual') {
		const result = await this.stepsBuilder.mode.editor.validateNow();
		if (!result.ok) {
			showPipelineFormError(new Error(result.message));
			return;
		}
		yaml = result.value;
		manual = true;
	} else {
		try {
			yaml = this.yamlString;
		} catch (e) {
			showPipelineFormError(e);
			return;
		}
	}

	const data = { ...this.metadataForm.value, yaml, ...(manual ? { manual: true } : {}) };
	// ... rest unchanged ...
}
```

- [ ] **Step 2: Update `hasChanges` and `canSave`**

```ts
hasChanges = $derived.by(() => {
	if (this.stepsBuilder.isManualMode && this.stepsBuilder.mode.id === 'manual') {
		return this.stepsBuilder.mode.editor.isDirty || /* metadata/runtime diffs */;
	}
	// existing blocks logic
});

canSave = $derived.by(() => {
	if (this.stepsBuilder.isManualMode && this.stepsBuilder.mode.id === 'manual') {
		return this.hasChanges && this.stepsBuilder.mode.editor.isValid;
	}
	return this.hasChanges && this.stepsBuilder.steps.length > 0;
});
```

- [ ] **Step 3: Update `validateExit()`**

Include manual editor dirty state in unsaved-changes check.

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/pipeline-form.svelte.ts
git commit -m "feat(pipeline-form): save and change detection for inline manual mode"
```

---

### Task 8: Top bar disabled controls + tooltips

**Files:**
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/metadata-form/metadata-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/metadata-form/metadata-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/runtime-options-form/runtime-options-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/runtime-options-form/runtime-options-form.svelte`

- [ ] **Step 1: Add `disabled` to MetadataForm and RuntimeOptionsForm**

```ts
disabled = $state(false);
```

Pass `disabled` to trigger `<Button disabled={form.disabled}>`.

- [ ] **Step 2: Wire `pipeline-form.svelte`**

```svelte
const manualMode = $derived(builder.isManualMode);
const manualTooltip = m.unavailable_in_manual_edit();

$effect(() => {
	metadata.disabled = manualMode;
	activityOptions.disabled = manualMode;
});
```

Wrap Undo, Redo, Info, Parameters in `Tooltip` with a `<span class="inline-flex">` wrapper around disabled buttons (disabled buttons don't receive hover events).

```svelte
<Tooltip disabled={!manualMode}>
	{#snippet child({ props })}
		<span {...props} class="inline-flex">
			<Button variant="ghost" disabled={manualMode} onclick={() => builder.undo()}>
				<UndoIcon />{m.Undo()}
			</Button>
		</span>
	{/snippet}
	{#snippet content()}{manualTooltip}{/snippet}
</Tooltip>
```

Repeat for Redo, metadata Render (or pass disabled into form), runtime Render.

- [ ] **Step 3: Run check + svelte-autofixer**

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/pipeline-form.svelte webapp/src/lib/pipeline-form/metadata-form/ webapp/src/lib/pipeline-form/runtime-options-form/
git commit -m "feat(pipeline-form): disable top-bar controls in manual mode with tooltips"
```

---

### Task 9: i18n keys

**Files:**
- Modify: `webapp/messages/en.json` (+ `it.json`, `de.json`, etc. per project convention)

- [ ] **Step 1: Add keys**

```json
"edit_manually": "Edit manually",
"manual_edit": "Manual edit",
"back_to_steps": "Back to steps",
"discard_manual_yaml_changes": "Discard manual YAML changes?",
"unavailable_in_manual_edit": "Unavailable while editing YAML manually"
```

- [ ] **Step 2: Run i18n compile if required**

Run: `cd webapp && bun run check`

- [ ] **Step 3: Commit**

```bash
git add webapp/messages/
git commit -m "i18n: add inline manual pipeline edit strings"
```

---

### Task 10: Final verification

- [ ] **Step 1: Run unit tests**

Run: `cd webapp && bun run test:unit -- --run`  
Expected: PASS

- [ ] **Step 2: Run lint and typecheck**

Run: `cd webapp && bun run lint && bun run check`  
Expected: PASS

- [ ] **Step 3: Manual smoke test**

1. Open pipeline edit page in blocks mode.
2. YAML preview ⋮ → Edit manually → editor appears, side columns grayed but scrollable.
3. Edit YAML → sticky error on invalid content.
4. Back to steps → confirm when dirty.
5. Save in manual mode → persists with `manual: true`.
6. Undo/Redo/Info/Parameters disabled with tooltips in manual mode.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `Pipeline.validateYaml` with `{ ok: true, value }` | Task 1 |
| Dropdown Edit manually | Task 5, 6 |
| `InlineManualEditor` + Runed Debounced | Task 3 |
| Disabled scrollable columns | Task 5, 6 |
| Manual editor column + CodeEditor | Task 5, 6 |
| Sticky bottom error only | Task 5 |
| Back to steps + confirm discard | Task 4, 5 |
| Save `manual: true` | Task 7 |
| Top bar disabled + tooltips (all) | Task 8 |
| Auto-close step form on enter | Task 4 |
| Full manual route uses shared validation | Task 2 |
| Unit tests | Tasks 1, 3, 4 |

---

## Out of scope (per spec)

- YAML → steps rehydration
- Removing `/edit/manual` route
- E2E tests
