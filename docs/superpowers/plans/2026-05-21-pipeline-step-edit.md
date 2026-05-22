# Pipeline Step Edit (Composer) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let users edit an existing pipeline step by reopening the same `initForm` in the left composer column (prefilled, explicit Save), replacing the Sheet-based `EditComponent` flow.

**Architecture:** Extend `Config.initForm({ intent, initial })` and `BaseForm` with `commit()`, `canSave()`, and `getSubmitData()`. `StepsBuilder` gains `initEditStep(index)` with mode `{ id: 'form', intent, stepIndex, form }`; submit updates the tuple in place. Step cards call `initEditStep` and highlight the active index.

**Tech Stack:** Svelte 5 runes, Paraglide (`m.*`), Vitest, existing `StateManager`, `EnrichedStep` tuples.

**Design spec:** `docs/superpowers/specs/2026-05-21-pipeline-step-edit-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps/types.ts` | `FormIntent`, `InitFormOptions`, `BaseForm.commit/canSave/getSubmitData`; remove `EditComponent` |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts` | Mode shape, `initEditStep`, shared form lifecycle, edit vs add submit |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte` | Edit column title, Save footer, pass `editingIndex` |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/step-card.svelte` | Pencil → `initEditStep`, highlight, remove Sheet |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/step-card-display.svelte` | Optional `editing` ring class |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/utils.ts` | `isStepEditable(step)` helper |
| `webapp/src/lib/pipeline-form/steps/hub-item/*` | Prefill, summary UI, intent branching |
| `webapp/src/lib/pipeline-form/steps/wallet-action/*` | Prefill, ready action UI, intent branching |
| `webapp/src/lib/pipeline-form/steps/conformance-check/*` | Prefill, ready test chip, intent branching |
| `webapp/src/lib/pipeline-form/steps/utils-steps/*` | Prefill, Save vs Add_step label |
| `webapp/messages/en.json` | `Edit_step` key |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts` | Builder edit save / discard tests |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts` | Edit does not auto-submit on `selectAction` |

**Delete:** `steps/wallet-action/edit-component.svelte`, `steps/hub-item/edit-component.svelte`

---

### Task 1: Form contract (`types.ts`)

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/types.ts`

- [ ] **Step 1: Replace `EditComponent` types with intent/options**

Remove `EditComponent`, `EditComponentProps`. Add:

```ts
export type FormIntent = 'add' | 'edit';

export type InitFormOptions<Deserialized = unknown> = {
	intent?: FormIntent;
	initial?: Deserialized;
};

export interface Config<ID extends string = string, Serialized = unknown, Deserialized = unknown> {
	use: ID;
	serialize: (step: Deserialized) => Serialized;
	deserialize: (step: Serialized) => Promise<Deserialized>;
	display: EntityData;
	initForm: (opts?: InitFormOptions<Deserialized>) => Form<Deserialized>;
	cardData: (data: Deserialized) => CardData;
	CardDetailsComponent?: Component<CardDetailsComponentProps<Deserialized>>;
	makeId: (data: Serialized) => string;
	linkProcedure?: (serialized: Serialized, previousSteps: PipelineStep[]) => void;
}
```

Extend `Form` interface:

```ts
export interface Form<Deserialized = unknown, T = any> extends Renderable<T> {
	readonly intent: FormIntent;
	onSubmit: (handler: (step: Deserialized) => void) => void;
	canSave(): boolean;
	getSubmitData(): Deserialized | undefined;
	commit(data?: Deserialized): void;
}
```

- [ ] **Step 2: Implement on `BaseForm`**

```ts
export abstract class BaseForm<Deserialized, T> implements Form<Deserialized, T> {
	abstract Component: Renderable<T>['Component'];

	readonly intent: FormIntent;
	protected handleSubmit: (step: Deserialized) => void = () => {};

	constructor(opts?: InitFormOptions<Deserialized>) {
		this.intent = opts?.intent ?? 'add';
		if (opts?.initial !== undefined) {
			this.applyInitial(opts.initial);
		}
	}

	protected applyInitial(_initial: Deserialized): void {
		// overridden per form
	}

	onSubmit(handler: (data: Deserialized) => void) {
		this.handleSubmit = handler;
	}

	commit(data?: Deserialized) {
		const payload = data ?? this.getSubmitData();
		if (payload !== undefined) {
			this.handleSubmit(payload);
		}
	}

	abstract canSave(): boolean;
	abstract getSubmitData(): Deserialized | undefined;
}
```

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/types.ts
git commit -m "refactor(pipeline-form): add FormIntent and BaseForm commit contract"
```

---

### Task 2: `StepsBuilder` edit lifecycle

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/_partials/utils.ts`

- [ ] **Step 1: Add `isStepEditable`**

In `utils.ts`:

```ts
export function isStepEditable(step: EnrichedStep): boolean {
	if (step[0].use === 'debug') return false;
	return getStepData(step) !== undefined && getStepConfig(step) !== undefined;
}
```

- [ ] **Step 2: Update `State` / mode type and shared `openForm` helper**

In `steps-builder.svelte.ts`:

```ts
type BuilderMode =
	| { id: 'idle' }
	| { id: 'form'; intent: pipelinestep.FormIntent; stepIndex?: number; form: pipelinestep.Form };

type State = {
	steps: EnrichedStep[];
	mode: BuilderMode;
};
```

Add private field `private formEffectCleanup: (() => void) | null = null;`

Replace duplicated effect logic with:

```ts
private openForm(
	intent: pipelinestep.FormIntent,
	config: pipelinestep.AnyConfig,
	opts: { initial?: GenericRecord; stepIndex?: number }
) {
	this.exitFormState();

	this.stateManager.run((state) => {
		const effectCleanup = $effect.root(() => {
			const form = config.initForm({
				intent,
				initial: opts.initial as never
			});
			form.onSubmit((formData) => {
				this.stateManager.run((inner) => {
					if (inner.mode.id !== 'form') return;

					if (inner.mode.intent === 'add') {
						const step: PipelineStep = {
							use: config.use as never,
							id: '',
							continue_on_error: false,
							with: config.serialize(formData)
						};
						inner.steps.push([step, formData as GenericRecord]);
					} else {
						const index = inner.mode.stepIndex;
						if (index === undefined) return;
						const tuple = inner.steps[index];
						if (!tuple || tuple[0].use === 'debug') return;
						const raw = tuple[0];
						raw.with = config.serialize(formData);
						tuple[1] = formData as GenericRecord;
					}

					this.exitFormState();
				});
			});
			inner.mode = {
				id: 'form',
				intent,
				stepIndex: opts.stepIndex,
				form
			};
		});
		this.formEffectCleanup = effectCleanup;
	});
}
```

Fix: use `state` parameter inside run callback correctly (the snippet above uses `inner` in onSubmit — mirror existing `initAddStep` pattern with `this.stateManager.run`).

- [ ] **Step 3: Refactor `initAddStep` and add `initEditStep`**

```ts
initAddStep(type: string) {
	if (this.state.mode.id === 'form') this.exitFormState();
	const config = pipelinestep.configs.find((c) => c.use === type);
	if (!config) return;
	this.openForm('add', config, {});
}

initEditStep(index: number) {
	if (this.state.mode.id === 'form') this.exitFormState();
	const step = this.state.steps[index];
	if (!step || !isStepEditable(step)) return;
	const config = getStepConfig(step);
	const data = getStepData(step);
	if (!config || !data) return;
	this.openForm('edit', config, { initial: data, stepIndex: index });
}
```

- [ ] **Step 4: Update `exitFormState` to run cleanup**

```ts
exitFormState() {
	this.formEffectCleanup?.();
	this.formEffectCleanup = null;
	this.stateManager.run((state) => {
		if (state.mode.id !== 'form') return;
		state.mode = { id: 'idle' };
	});
}
```

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts \
  webapp/src/lib/pipeline-form/steps-builder/_partials/utils.ts
git commit -m "feat(pipeline-form): add StepsBuilder initEditStep and form mode intent"
```

---

### Task 3: Composer UI + step card

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/_partials/step-card.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/_partials/step-card-display.svelte`
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: i18n**

In `webapp/messages/en.json` (alphabetically near `Edit_*` or `Add_step`):

```json
"Edit_step": "Edit step",
```

Run message compile if project uses `bun run predev` / paraglide sync per repo convention.

- [ ] **Step 2: Column title + Save footer in `steps-builder.svelte`**

Import `isStepEditable` is not needed here. Add derived:

```ts
const formMode = $derived(builder.mode.id === 'form' ? builder.mode : null);
const editingIndex = $derived(
	formMode?.intent === 'edit' ? formMode.stepIndex : undefined
);
```

Change first column title:

```svelte
<Column title={formMode?.intent === 'edit' ? m.Edit_step() : m.Add_step()}>
```

Wrap form render:

```svelte
{:else if builder.mode.id == 'form'}
	<div class="flex grow flex-col" in:fly>
		<Render item={builder.mode.form} />
		{#if builder.mode.intent === 'edit'}
			<div class="mt-auto border-t p-4">
				<Button
					class="w-full"
					disabled={!builder.mode.form.canSave()}
					onclick={() => builder.mode.form.commit()}
				>
					{m.Save()}
				</Button>
			</div>
		{/if}
	</div>
{/if}
```

Pass highlight to cards:

```svelte
<StepCard {builder} {step} {index} editing={editingIndex === index} />
```

- [ ] **Step 3: Simplify `step-card.svelte`**

Remove `Sheet`, `EditComponent`, `isEditSheetOpen`.

```svelte
import { isStepEditable } from './utils.js';

type Props = {
	index: number;
	step: EnrichedStep;
	builder: StepsBuilder;
	editing?: boolean;
};

let { builder, step, index, editing = false }: Props = $props();

const editable = $derived(isStepEditable(step));
```

Pencil block:

```svelte
{#if editable}
	<IconButton
		icon={PencilIcon}
		variant="ghost"
		size="xs"
		onclick={() => builder.initEditStep(index)}
	/>
{/if}
```

Pass `editing` to `StepCardDisplay`.

- [ ] **Step 4: Highlight in `step-card-display.svelte`**

Add prop `editing?: boolean` and extend root `class`:

```svelte
editing && 'ring-2 ring-primary'
```

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte \
  webapp/src/lib/pipeline-form/steps-builder/_partials/step-card.svelte \
  webapp/src/lib/pipeline-form/steps-builder/_partials/step-card-display.svelte \
  webapp/messages/en.json
git commit -m "feat(pipeline-form): composer edit UI with Save footer and card highlight"
```

---

### Task 4: Utils forms (email, HTTP)

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/utils-steps/email-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/utils-steps/http-request-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/utils-steps/email-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/utils-steps/http-request-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/utils-steps/index.ts`

- [ ] **Step 1: Email form class**

```ts
export class EmailStepForm extends BaseForm<EmailFormData, EmailStepForm> {
	// ...

	constructor(opts?: InitFormOptions<EmailFormData>) {
		super(opts);
	}

	protected applyInitial(initial: EmailFormData) {
		this.data = { ...initial };
	}

	canSave() {
		return this.isValid;
	}

	getSubmitData() {
		return this.isValid ? this.data : undefined;
	}

	submit() {
		this.commit();
	}
}
```

Update `initForm` in `index.ts`:

```ts
initForm: (opts) => new EmailStepForm(opts),
```

- [ ] **Step 2: HTTP form class** — same pattern as email (`applyInitial`, `canSave`, `getSubmitData`, `commit` in `submit()`).

- [ ] **Step 3: Button labels in `.svelte` files**

```svelte
<T>{form.intent === 'edit' ? m.Save() : m.Add_step()}</T>
```

Hide bottom button when `intent === 'edit'` (composer footer owns Save) **OR** keep button hidden in edit — **preferred: hide inline button in edit**:

```svelte
{#if form.intent === 'add'}
	<Button class="w-full" disabled={!form.isValid} onclick={() => form.submit()}>
		<T>{m.Add_step()}</T>
	</Button>
{/if}
```

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/utils-steps/
git commit -m "feat(pipeline-form): utils step forms support edit intent and prefill"
```

---

### Task 5: Hub item steps

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/index.ts`
- Delete: `webapp/src/lib/pipeline-form/steps/hub-item/edit-component.svelte`

- [ ] **Step 1: Form class**

```ts
export class HubItemStepForm extends BaseForm<HubItem, HubItemStepForm> {
	selectedItem = $state<HubItem | undefined>(undefined);

	constructor(private props: Props, opts?: InitFormOptions<HubItem>) {
		super(opts);
	}

	protected applyInitial(initial: HubItem) {
		this.selectedItem = initial;
	}

	canSave() {
		return this.selectedItem !== undefined;
	}

	getSubmitData() {
		return this.selectedItem;
	}

	async selectItem(item: HubItem) {
		this.selectedItem = item;
		if (this.intent === 'add') {
			this.commit(item);
		}
	}

	discardSelection() {
		this.selectedItem = undefined;
	}
}
```

Update each hub config `initForm`:

```ts
initForm: (opts) =>
	new HubItemStepForm({ collection: 'credentials', entityData: entities.credentials }, opts),
```

Remove `EditComponent` import and property from all three configs in `index.ts`.

- [ ] **Step 2: Summary UI in `hub-item-step-form.svelte`**

When `form.selectedItem`:

```svelte
{#if form.selectedItem}
	<div class="border-b p-4">
		<WithLabel label={labels.singular}>
			<ItemCard
				avatar={getHubItemLogo(form.selectedItem)}
				title={form.selectedItem.name}
				subtitle={form.selectedItem.organization_name}
				onDiscard={() => form.discardSelection()}
			/>
		</WithLabel>
	</div>
{/if}
```

Keep search list below (unchanged).

- [ ] **Step 3: Delete `edit-component.svelte`**

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/hub-item/
git rm webapp/src/lib/pipeline-form/steps/hub-item/edit-component.svelte
git commit -m "feat(pipeline-form): hub steps edit via initForm; remove EditComponent"
```

---

### Task 6: Wallet action step

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/index.ts`
- Delete: `webapp/src/lib/pipeline-form/steps/wallet-action/edit-component.svelte`

- [ ] **Step 1: Constructor + applyInitial**

```ts
constructor(opts?: InitFormOptions<WalletActionStepData>) {
	super(opts);
	if (!opts?.initial && ExecutionTarget.state.current) {
		this.data = { ...ExecutionTarget.state.current, action: undefined };
	}
}

protected applyInitial(initial: WalletActionStepData) {
	this.data = { ...initial };
}
```

Change `initForm` in `index.ts`:

```ts
initForm: (opts) => new WalletActionStepForm(opts),
```

Remove `EditComponent` import and property.

- [ ] **Step 2: Intent-aware `selectAction`**

```ts
selectAction(action: WalletActionsResponse) {
	ExecutionTarget.state.current = {
		wallet: this.data.wallet!,
		version: this.data.version!,
		runner: this.data.runner!
	};
	const payload = { ...this.data, action } as WalletActionStepData;
	this.data.action = action;
	if (this.intent === 'add') {
		this.commit(payload);
	}
}
```

Add `canSave()` → `this.state === 'ready'`.

Add `getSubmitData()` → when ready, return full `WalletActionStepData`.

Add `removeAction()` setting `action` undefined (for discard from ready summary).

- [ ] **Step 3: Ready UI for action in `.svelte`**

Inside `{#if form.data.wallet}` summary block, after runner:

```svelte
{#if form.data.action}
	<WithLabel label={m.Wallet_action()}>
		<ItemCard
			title={form.data.action.name}
			onDiscard={() => form.removeAction()}
		/>
	</WithLabel>
{/if}
```

When `form.state === 'ready'` and no action picker visible, do not show `select-action` list unless user discarded action (state falls back via `removeAction` clearing action → `select-action`).

- [ ] **Step 4: Delete `edit-component.svelte`**

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/
git rm webapp/src/lib/pipeline-form/steps/wallet-action/edit-component.svelte
git commit -m "feat(pipeline-form): wallet step edit via initForm; remove EditComponent"
```

---

### Task 7: Conformance check step

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/index.ts`

- [ ] **Step 1: Form class**

```ts
constructor(opts?: InitFormOptions<FormData>) {
	super(opts);
}

protected applyInitial(initial: FormData) {
	this.data = { ...initial };
}

canSave() {
	return this.state === 'ready';
}

getSubmitData() {
	return this.state === 'ready' ? (this.data as FormData) : undefined;
}
```

```ts
selectTest(test: Test) {
	this.data.test = test;
	if (this.intent === 'add') {
		this.commit({ ...this.data, test } as FormData);
	}
}

discardTest() {
	this.data.test = undefined;
}
```

Update `initForm: (opts) => new ConformanceCheckStepForm(opts)`.

- [ ] **Step 2: Test chip in summary (`.svelte`)**

After suite chip:

```svelte
{#if form.data.test}
	<WithLabel label={m.Test()}>
		<ItemCard
			title={form.data.test.split('/').at(-1)?.replaceAll('+', ' ') ?? form.data.test}
			onDiscard={() => form.discardTest()}
		/>
	</WithLabel>
{/if}
```

When `form.state === 'ready'`, hide the picker list (only show summary + Save in column footer).

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/conformance-check/
git commit -m "feat(pipeline-form): conformance step edit via initForm"
```

---

### Task 8: Unit tests

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts`
- Create: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts`

- [ ] **Step 1: `steps-builder.test.ts`**

```ts
import { describe, expect, it, vi } from 'vitest';

import type { EnrichedStep } from './types';
import { StepsBuilder } from './steps-builder.svelte.js';

const emailStep: EnrichedStep = [
	{
		use: 'email',
		id: 'email-user-0001',
		continue_on_error: true,
		with: { recipient: 'a@b.com', subject: 'Hi', body: '', sender: '' }
	},
	{ recipient: 'a@b.com', subject: 'Hi', body: '', sender: '' }
];

describe('StepsBuilder initEditStep', () => {
	it('updates step on edit save and preserves id and continue_on_error', () => {
		const builder = new StepsBuilder({ steps: [emailStep], yamlPreview: () => '' });
		builder.initEditStep(0);
		expect(builder.mode.id).toBe('form');
		if (builder.mode.id !== 'form') throw new Error('expected form mode');
		expect(builder.mode.intent).toBe('edit');

		const form = builder.mode.form;
		// mutate via form API — EmailStepForm sets data from initial
		form.commit({
			recipient: 'c@d.com',
			subject: 'Updated',
			body: '',
			sender: ''
		});

		expect(builder.mode.id).toBe('idle');
		expect(builder.steps[0][0].id).toBe('email-user-0001');
		expect(builder.steps[0][0].continue_on_error).toBe(true);
		expect(builder.steps[0][1]).toMatchObject({ recipient: 'c@d.com', subject: 'Updated' });
	});

	it('exitFormState discards without mutating steps', () => {
		const builder = new StepsBuilder({ steps: [emailStep], yamlPreview: () => '' });
		builder.initEditStep(0);
		if (builder.mode.id === 'form') {
			const form = builder.mode.form as { data: { recipient: string } };
			if ('data' in form) form.data.recipient = 'mutated@x.com';
		}
		builder.exitFormState();
		expect(builder.mode.id).toBe('idle');
		expect(builder.steps[0][1]).toMatchObject({ recipient: 'a@b.com' });
	});
});
```

Adjust test if `EmailStepForm` instance is accessed differently (use public `getSubmitData` + `commit` with explicit payload).

- [ ] **Step 2: `wallet-action-step-form.test.ts`**

Mock PB/search dependencies minimally: test class with `intent: 'edit'`, prefilled data, call `selectAction`, assert `handleSubmit` not called until `commit()`:

```ts
import { describe, expect, it, vi } from 'vitest';

import { WalletActionStepForm } from './wallet-action-step-form.svelte.js';

describe('WalletActionStepForm edit intent', () => {
	it('selectAction does not commit until commit()', () => {
		const onSubmit = vi.fn();
		const form = new WalletActionStepForm({
			intent: 'edit',
			initial: {
				wallet: { id: 'w1', name: 'W' } as never,
				version: 'installed_from_external_source' as never,
				runner: 'global' as never,
				action: { id: 'a1', name: 'Old' } as never
			}
		});
		form.onSubmit(onSubmit);
		form.selectAction({ id: 'a2', name: 'New' } as never);
		expect(onSubmit).not.toHaveBeenCalled();
		form.commit();
		expect(onSubmit).toHaveBeenCalledOnce();
	});
});
```

Stub or skip if constructor triggers network — use minimal partial mocks and `vi.mock` for `pb` if needed.

- [ ] **Step 3: Run tests**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/steps-builder.test.ts src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts \
  webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts
git commit -m "test(pipeline-form): cover step edit save and wallet edit commit gate"
```

---

### Task 9: Verification

- [ ] **Step 1: Typecheck and lint**

```bash
cd webapp && bun run check && bun run lint
```

Expected: no errors

- [ ] **Step 2: Manual smoke (dev server running)**

1. Open pipeline create/edit with existing steps.
2. Click pencil on email step → left column shows **Edit step**, fields prefilled.
3. Change recipient → **Save** → card/YAML update.
4. **Back** without Save → step unchanged.
5. Wallet step → ready summary with action; change action → Save.
6. Hub step → summary card + search; no Sheet.
7. Debug step → no pencil.

- [ ] **Step 3: Final commit if fixes needed**

```bash
git add -A && git commit -m "fix(pipeline-form): address check/lint from step edit feature"
```

---

## Spec coverage (self-review)

| Spec requirement | Task |
|------------------|------|
| Left column edit via `initForm` | 2, 3 |
| Remove `EditComponent` / Sheet | 5, 6, 3 |
| `initForm({ intent, initial })` | 1, 4–7 |
| Explicit Save on edit | 1, 3, 4–7 |
| Back discards | 2 (`exitFormState`) |
| Preserve `id` / `continue_on_error` | 2 |
| Debug / enrich no edit | 2 (`isStepEditable`), 3 |
| Wizard ready view | 6, 7 |
| Hub summary | 5 |
| Highlight card | 3 |
| i18n `Edit_step` | 3 |
| Tests | 8 |

`linkProcedure` runs at YAML generation (`createPipelineYaml`) — no builder change required.
