# Pipeline Custom Integration Config Form Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the generic `HubItemStepForm` for `custom-check` pipeline steps with a dedicated form that picks from `custom_checks` (owned + published), renders a JSON schema config form when the integration defines one, and serializes/deserializes `with.config`.

**Architecture:** New `custom-integration/` step module with `CustomIntegrationStepForm` (state machine + json schema validation), Svelte view using `StepCollectionPicker` + `JsonSchemaFormComponent`, and updated `customCheckStepConfig` with typed serialize/deserialize. `HubItemStepForm` remains for credential-offer and use-case-verification only.

**Tech Stack:** Svelte 5 runes, `@sjsf/form`, existing `StepCollectionPicker` / `CollectionManager`, Paraglide (`m.*`), Vitest.

**Design spec:** `docs/superpowers/specs/2026-06-06-pipeline-custom-integration-config-form-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps/custom-integration/index.ts` | **Create** — `customCheckStepConfig` with serialize/deserialize/cardData |
| `webapp/src/lib/pipeline-form/steps/custom-integration/index.test.ts` | **Create** — serialize/deserialize unit tests |
| `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte.ts` | **Create** — form class, state machine, json schema form lifecycle |
| `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts` | **Create** — selection/commit/canSave unit tests |
| `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte` | **Create** — picker + config UI |
| `webapp/src/lib/pipeline-form/steps/index.ts` | **Modify** — import from `custom-integration` instead of `hub-item` |
| `webapp/src/lib/pipeline-form/steps/hub-item/index.ts` | **Modify** — remove `customCheckStepConfig` |
| `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte.ts` | **Modify** — remove `custom_checks` from `HubStepCollection` union |

**Unchanged:** backend workflow, run page `CustomCheckConfigEditor`, pipeline schema.

---

### Task 1: Step config serialize/deserialize

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps/custom-integration/index.test.ts`
- Create: `webapp/src/lib/pipeline-form/steps/custom-integration/index.ts`

- [ ] **Step 1: Write failing serialize/deserialize tests**

```typescript
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CustomChecksResponse } from '@/pocketbase/types';

import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/canonify/index.js', () => ({
	getRecordByCanonifiedPath: vi.fn()
}));

vi.mock('$lib/utils', () => ({
	getPath: vi.fn((record: { canonified_name?: string }, trim?: boolean) =>
		trim ? record.canonified_name ?? '' : record.canonified_name ?? ''
	)
}));

vi.mock('@/i18n/index.js', () => ({
	m: {
		Pipeline_form_missing_check_id: () => 'Missing check ID'
	}
}));

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';

import { customCheckStepConfig } from './index.js';

const integration = {
	id: 'ci1',
	name: 'My Integration',
	canonified_name: 'org/my-integration',
	logo: 'logo.png'
} as CustomChecksResponse;

describe('customCheckStepConfig', () => {
	beforeEach(() => {
		vi.mocked(getRecordByCanonifiedPath).mockReset();
	});

	it('serialize emits check_id only when config is absent', () => {
		expect(customCheckStepConfig.serialize({ integration })).toEqual({
			check_id: 'org/my-integration'
		});
	});

	it('serialize includes config when non-empty', () => {
		expect(
			customCheckStepConfig.serialize({
				integration,
				config: { apiKey: 'secret' }
			})
		).toEqual({
			check_id: 'org/my-integration',
			config: { apiKey: 'secret' }
		});
	});

	it('serialize omits config when empty object', () => {
		expect(
			customCheckStepConfig.serialize({
				integration,
				config: {}
			})
		).toEqual({
			check_id: 'org/my-integration'
		});
	});

	it('deserialize loads integration and config', async () => {
		vi.mocked(getRecordByCanonifiedPath).mockResolvedValue(integration);

		const result = await customCheckStepConfig.deserialize({
			check_id: 'org/my-integration',
			config: { apiKey: 'secret' }
		});

		expect(getRecordByCanonifiedPath).toHaveBeenCalledWith('org/my-integration');
		expect(result).toEqual({
			integration,
			config: { apiKey: 'secret' }
		});
	});

	it('deserialize throws when check_id is missing', async () => {
		await expect(customCheckStepConfig.deserialize({})).rejects.toThrow('Missing check ID');
	});

	it('deserialize propagates lookup errors', async () => {
		const err = new Error('not found');
		vi.mocked(getRecordByCanonifiedPath).mockResolvedValue(err);

		await expect(
			customCheckStepConfig.deserialize({ check_id: 'org/missing' })
		).rejects.toThrow('not found');
	});

	it('cardData uses integration name and public run URL', () => {
		const card = customCheckStepConfig.cardData({ integration });
		expect(card.title).toBe('My Integration');
		expect(card.copyText).toBe('org/my-integration');
		expect(card.publicUrl).toContain('/my/custom-integrations/org/my-integration/run');
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/index.test.ts --run`

Expected: FAIL — module `./index.js` not found

- [ ] **Step 3: Implement `index.ts`**

```typescript
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getRecordByCanonifiedPath } from '$lib/canonify/index.js';
import { entities } from '$lib/global/entities.js';
import { getCustomCheckPublicUrl } from '$lib/hub/utils.js';
import { getPath } from '$lib/utils';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { m } from '@/i18n/index.js';
import { pb } from '@/pocketbase';

import type { TypedConfig } from '../types';

import { getLastPathSegment } from '../_partials/misc';
import {
	CustomIntegrationStepForm,
	type CustomIntegrationStepFormData
} from './custom-integration-step-form.svelte.js';

export type { CustomIntegrationStepFormData };

export const customCheckStepConfig: TypedConfig<'custom-check', CustomIntegrationStepFormData> = {
	use: 'custom-check',
	display: entities.custom_checks,
	initForm: (opts) => new CustomIntegrationStepForm(opts),
	serialize: ({ integration, config }) => {
		const serialized: { check_id: string; config?: Record<string, unknown> } = {
			check_id: getPath(integration, true)
		};
		if (config && Object.keys(config).length > 0) {
			serialized.config = config;
		}
		return serialized;
	},
	deserialize: async ({ check_id, config }) => {
		if (!check_id) throw new Error(m.Pipeline_form_missing_check_id());
		const integration = await getRecordByCanonifiedPath<CustomChecksResponse>(check_id);
		if (integration instanceof Error) throw integration;
		return {
			integration,
			config: config as Record<string, unknown> | undefined
		};
	},
	cardData: ({ integration }) => ({
		title: integration.name,
		copyText: getPath(integration, true),
		avatar: integration.logo ? pb.files.getURL(integration, integration.logo) : undefined,
		publicUrl: getCustomCheckPublicUrl(integration)
	}),
	makeId: ({ check_id }) => getLastPathSegment(check_id ?? 'custom-check-unknown')
};
```

- [ ] **Step 4: Add minimal form class stub so tests compile**

Create `custom-integration-step-form.svelte.ts` with:

```typescript
import { BaseForm, type InitFormOptions } from '../types';
import Component from './custom-integration-step-form.svelte';

export type CustomIntegrationStepFormData = {
	integration: import('@/pocketbase/types').CustomChecksResponse;
	config?: Record<string, unknown>;
};

export class CustomIntegrationStepForm extends BaseForm<
	CustomIntegrationStepFormData,
	CustomIntegrationStepForm
> {
	readonly Component = Component;
	canSave() {
		return false;
	}
	getSubmitData() {
		return undefined;
	}
	constructor(_opts?: InitFormOptions<CustomIntegrationStepFormData>) {
		super(_opts);
	}
}
```

Create empty `custom-integration-step-form.svelte`:

```svelte
<script lang="ts">
	import type { SelfProp } from '$lib/renderable';
	import type { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';
	let { self: _form }: SelfProp<CustomIntegrationStepForm> = $props();
</script>
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/index.test.ts --run`

Expected: PASS (7 tests)

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/custom-integration/
git commit -m "feat(pipeline-form): add custom integration step config serialize/deserialize"
```

---

### Task 2: `CustomIntegrationStepForm` class

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte.ts`
- Create: `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts`

- [ ] **Step 1: Write failing form class tests**

```typescript
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { CustomChecksResponse } from '@/pocketbase/types';

import { describe, expect, it, vi } from 'vitest';

vi.mock('./custom-integration-step-form.svelte', () => ({ default: class {} }));

vi.mock('@/components/json-schema-form', () => ({
	createJsonSchemaForm: vi.fn(() => ({}))
}));

vi.mock('@sjsf/form', () => ({
	getValueSnapshot: vi.fn(() => ({ apiKey: 'abc' })),
	validate: vi.fn(() => ({ errors: [] }))
}));

import { validate } from '@sjsf/form';

import { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';

const integrationNoSchema = {
	id: 'ci1',
	name: 'Plain',
	input_json_schema: null
} as CustomChecksResponse;

const integrationWithSchema = {
	id: 'ci2',
	name: 'With Schema',
	input_json_schema: { type: 'object', properties: { apiKey: { type: 'string' } } },
	input_json_sample: { apiKey: '' }
} as CustomChecksResponse;

describe('CustomIntegrationStepForm', () => {
	it('selectIntegration auto-commits on add when no schema', () => {
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm({ intent: 'add' });
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationNoSchema);
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].integration).toBe(integrationNoSchema);
	});

	it('selectIntegration does not auto-commit on add when schema exists', () => {
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm({ intent: 'add' });
		form.onSubmit(onSubmit);
		form.selectIntegration(integrationWithSchema);
		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.state).toBe('configure');
	});

	it('canSave is false when schema invalid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [{ message: 'required' }] } as never);
		const form = new CustomIntegrationStepForm({
			intent: 'edit',
			initial: { integration: integrationWithSchema, config: {} }
		});
		expect(form.canSave()).toBe(false);
	});

	it('canSave is true when schema valid', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const form = new CustomIntegrationStepForm({
			intent: 'edit',
			initial: { integration: integrationWithSchema, config: { apiKey: 'abc' } }
		});
		expect(form.canSave()).toBe(true);
	});

	it('discardIntegration clears selection and schema form', () => {
		const form = new CustomIntegrationStepForm({
			intent: 'edit',
			initial: { integration: integrationWithSchema }
		});
		form.discardIntegration();
		expect(form.data.integration).toBeUndefined();
		expect(form.jsonSchemaForm).toBeUndefined();
		expect(form.state).toBe('select-integration');
	});

	it('submit commits valid schema config', () => {
		vi.mocked(validate).mockReturnValue({ errors: [] } as never);
		const onSubmit = vi.fn();
		const form = new CustomIntegrationStepForm({
			intent: 'add',
			initial: { integration: integrationWithSchema }
		});
		form.onSubmit(onSubmit);
		form.submit();
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0].config).toEqual({ apiKey: 'abc' });
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts --run`

Expected: FAIL — methods/state not implemented

- [ ] **Step 3: Implement full form class**

Replace stub in `custom-integration-step-form.svelte.ts`:

```typescript
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getValueSnapshot, validate } from '@sjsf/form';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { createJsonSchemaForm, type JsonSchemaForm } from '@/components/json-schema-form';

import { BaseForm, type InitFormOptions } from '../types';
import Component from './custom-integration-step-form.svelte';

export type CustomIntegrationStepFormData = {
	integration: CustomChecksResponse;
	config?: Record<string, unknown>;
};

type FormState = 'select-integration' | 'configure' | 'ready';

export class CustomIntegrationStepForm extends BaseForm<
	CustomIntegrationStepFormData,
	CustomIntegrationStepForm
> {
	readonly Component = Component;

	data = $state<Partial<CustomIntegrationStepFormData>>({});
	jsonSchemaForm = $state<JsonSchemaForm | undefined>(undefined);

	constructor(opts?: InitFormOptions<CustomIntegrationStepFormData>) {
		super(opts);
		if (opts?.initial) {
			this.data = { ...opts.initial };
			this.initJsonSchemaForm(opts.initial.integration, opts.initial.config);
		}
	}

	hasSchema = $derived(Boolean(this.data.integration?.input_json_schema));

	isSchemaValid = $derived.by(() => {
		if (!this.jsonSchemaForm) return true;
		return (validate(this.jsonSchemaForm).errors ?? []).length === 0;
	});

	state: FormState = $derived.by(() => {
		if (!this.data.integration) return 'select-integration';
		if (this.hasSchema && !this.isSchemaValid) return 'configure';
		return 'ready';
	});

	canSave() {
		return this.state === 'ready';
	}

	getSubmitData(): CustomIntegrationStepFormData | undefined {
		if (this.state !== 'ready' || !this.data.integration) return undefined;
		const config = this.jsonSchemaForm
			? (getValueSnapshot(this.jsonSchemaForm) as Record<string, unknown>)
			: undefined;
		return {
			integration: this.data.integration,
			config
		};
	}

	selectIntegration(integration: CustomChecksResponse) {
		this.data.integration = integration;
		this.data.config = undefined;
		this.initJsonSchemaForm(integration);
		if (this.intent === 'add' && !this.hasSchema) {
			const payload = this.getSubmitData();
			if (payload) this.commit(payload);
		}
	}

	discardIntegration() {
		this.data.integration = undefined;
		this.data.config = undefined;
		this.jsonSchemaForm = undefined;
	}

	submit() {
		this.commit();
	}

	private initJsonSchemaForm(integration: CustomChecksResponse, initialConfig?: Record<string, unknown>) {
		const schema = integration.input_json_schema;
		if (schema) {
			this.jsonSchemaForm = createJsonSchemaForm(schema as object, {
				hideTitle: true,
				initialValue: initialConfig ?? integration.input_json_sample ?? undefined
			});
		} else {
			this.jsonSchemaForm = undefined;
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/ --run`

Expected: PASS (all tests in folder)

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte.ts \
        webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts
git commit -m "feat(pipeline-form): add CustomIntegrationStepForm state and validation"
```

---

### Task 3: Svelte view

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte`

- [ ] **Step 1: Implement the view**

```svelte
<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { userOrganization } from '$lib/app-state';
	import { entities } from '$lib/global/entities';

	import type { CustomChecksResponse } from '@/pocketbase/types';

	import { JsonSchemaFormComponent } from '@/components/json-schema-form';
	import T from '@/components/ui-custom/t.svelte';
	import { Button } from '@/components/ui/button';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import type { CustomIntegrationStepForm } from './custom-integration-step-form.svelte.js';

	import ItemCard from '../_partials/item-card.svelte';
	import StepCollectionPicker from '../_partials/step-collection-picker.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	let { self: form }: SelfProp<CustomIntegrationStepForm> = $props();

	const orgId = $derived(userOrganization.current?.id ?? '');
	const pickerFilter = $derived(
		orgId ? `(owner.id = "${orgId}") || (published = true)` : 'published = true'
	);
</script>

{#if form.data.integration}
	<div class="space-y-6 border-b p-4">
		<WithLabel label={entities.custom_checks.labels.singular}>
			<ItemCard
				avatar={form.data.integration.logo
					? pb.files.getURL(form.data.integration, form.data.integration.logo)
					: undefined}
				title={form.data.integration.name}
				subtitle={form.data.integration.expand?.owner?.name}
				onDiscard={() => form.discardIntegration()}
			/>
		</WithLabel>

		{#if form.jsonSchemaForm}
			<div class="space-y-2">
				<h3 class="text-sm font-medium">{m.Fields()}</h3>
				<JsonSchemaFormComponent form={form.jsonSchemaForm} hideSubmitButton />
			</div>
		{/if}

		{#if form.intent === 'add' && form.hasSchema}
			<Button class="w-full" disabled={!form.canSave()} onclick={() => form.submit()}>
				<T>{m.Add_step()}</T>
			</Button>
		{/if}
	</div>
{/if}

{#if form.state === 'select-integration'}
	<StepCollectionPicker
		collection="custom_checks"
		label={entities.custom_checks.labels.singular}
		queryOptions={{
			filter: pickerFilter,
			searchFields: ['name'],
			expand: ['owner']
		}}
		onSelect={(record) => form.selectIntegration(record as CustomChecksResponse)}
	>
		{#snippet item({ record, onSelect })}
			{@const integration = record as CustomChecksResponse}
			<ItemCard
				avatar={integration.logo ? pb.files.getURL(integration, integration.logo) : undefined}
				title={integration.name}
				subtitle={integration.expand?.owner?.name}
				onClick={() => onSelect(record)}
			/>
		{/snippet}
	</StepCollectionPicker>
{/if}
```

- [ ] **Step 2: Run Svelte check**

Run: `cd webapp && bun run check`

Expected: no errors in `custom-integration-step-form.svelte`

- [ ] **Step 3: Run unit tests again**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/ --run`

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte
git commit -m "feat(pipeline-form): add custom integration step form UI"
```

---

### Task 4: Wire step registry and remove hub-item custom-check

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/index.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/index.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte.ts`

- [ ] **Step 1: Update `steps/index.ts`**

Replace hub import usage for custom-check:

```typescript
import { customCheckStepConfig } from './custom-integration';
// remove hubSteps.customCheckStepConfig from coreConfigs
export const coreConfigs: AnyConfig[] = [
	walletActionStepConfig,
	hubSteps.credentialsStepConfig,
	hubSteps.useCaseVerificationStepConfig,
	conformanceCheckStepConfig,
	customCheckStepConfig
];
```

- [ ] **Step 2: Remove `customCheckStepConfig` from `hub-item/index.ts`**

Delete lines 60–80 (the `customCheckStepConfig` export block).

- [ ] **Step 3: Narrow `HubStepCollection` in `hub-item-step-form.svelte.ts`**

```typescript
type HubStepCollection = Extract<
	HubItemType,
	'credentials' | 'use_cases_verifications'
>;
```

- [ ] **Step 4: Run full webapp unit tests for pipeline-form**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/ --run`

Expected: PASS

- [ ] **Step 5: Run lint**

Run: `cd webapp && bun run lint`

Expected: no new errors

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/index.ts \
        webapp/src/lib/pipeline-form/steps/hub-item/index.ts \
        webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte.ts
git commit -m "feat(pipeline-form): wire custom integration step and drop hub-item custom-check"
```

---

### Task 5: Manual verification

- [ ] **Step 1: Start dev stack**

Run: `make dev` (or `cd webapp && bun run dev` with backend running)

- [ ] **Step 2: Verify add flow — no schema**

1. Open pipeline composer → add custom integration step
2. Pick an integration without `input_json_schema`
3. Confirm step is added immediately (auto-commit)

- [ ] **Step 3: Verify add flow — with schema**

1. Add custom integration step
2. Pick an integration with `input_json_schema`
3. Confirm JSON schema form appears
4. Confirm **Add step** is disabled until required fields valid
5. Fill form → add step
6. Confirm YAML preview contains `check_id` and `config`

- [ ] **Step 4: Verify edit round-trip**

1. Edit the step with config
2. Confirm values restored in JSON schema form
3. Save pipeline → reload → confirm config persists in YAML

- [ ] **Step 5: Verify picker scope**

Confirm owned integrations and published integrations from other orgs both appear in picker.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Dedicated `CustomIntegrationStepForm` | Task 2, 3 |
| `custom_checks` picker owned + published | Task 3 |
| JSON schema form when schema exists | Task 3 |
| Required validation when schema exists | Task 2 |
| `config: Record<string, unknown>` serialize/deserialize | Task 1 |
| Remove hub-item custom-check config | Task 4 |
| Card display (name, logo, public URL) | Task 1 |
| Frontend only (no backend) | N/A — intentionally omitted |
| Unit tests serialize/deserialize + canSave | Task 1, 2 |

---

## Follow-up (separate work)

- Backend: marshal `with.config` → `CustomCheckWorkflow` `Config.env`
- Optional DRY with run page `CustomCheckConfigEditor`
