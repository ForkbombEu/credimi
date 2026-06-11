# Pipeline Custom Integration Config Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop pre-filling pipeline custom-integration config from `input_json_sample`; use JSON Schema defaults on first add, persist last-committed config per integration in localStorage, and restore it when the user re-selects the same integration.

**Architecture:** New `config-storage.ts` module wraps `createStorageHandlers` with get/set by canonified `checkId`. `CustomIntegrationStepForm` resolves initial values via explicit edit config → localStorage → undefined (schema defaults), and overrides `commit()` to persist after successful save.

**Tech Stack:** Svelte 5 runes, `@sjsf/form`, `createStorageHandlers` from `@/utils/storage`, Vitest.

**Design spec:** `docs/superpowers/specs/2026-06-06-pipeline-custom-integration-config-persistence-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps/custom-integration/config-storage.ts` | **Create** — localStorage get/set + `resolveInitialConfig` |
| `webapp/src/lib/pipeline-form/steps/custom-integration/config-storage.test.ts` | **Create** — storage unit tests |
| `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte.ts` | **Modify** — use resolver, override `commit()` |
| `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts` | **Modify** — resolver + persist tests |

**Unchanged:** Svelte view, `index.ts`, standalone run page `CustomCheckConfigEditor`.

---

### Task 1: localStorage config storage module

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps/custom-integration/config-storage.test.ts`
- Create: `webapp/src/lib/pipeline-form/steps/custom-integration/config-storage.ts`

- [ ] **Step 1: Write failing storage tests**

```typescript
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { CustomChecksResponse } from '@/pocketbase/types';

vi.mock('$app/environment', () => ({ browser: true }));

vi.mock('$lib/utils', () => ({
	getPath: vi.fn((record: { canonified_name?: string }, trim?: boolean) =>
		trim ? (record.canonified_name ?? '') : (record.canonified_name ?? '')
	)
}));

import { getStoredConfig, resolveInitialConfig, setStoredConfig } from './config-storage.js';

const STORAGE_KEY = 'pipeline_custom_integration_configs';

const integration = {
	id: 'ci1',
	canonified_name: 'org/my-integration',
	input_json_sample: { apiKey: 'sample-should-not-be-used' }
} as CustomChecksResponse;

describe('config-storage', () => {
	beforeEach(() => {
		const store = new Map<string, string>();
		vi.stubGlobal('localStorage', {
			clear: () => store.clear(),
			getItem: (key: string) => store.get(key) ?? null,
			removeItem: (key: string) => {
				store.delete(key);
			},
			setItem: (key: string, value: string) => {
				store.set(key, value);
			}
		});
	});

	it('getStoredConfig returns undefined when empty', () => {
		expect(getStoredConfig('org/my-integration')).toBeUndefined();
	});

	it('setStoredConfig and getStoredConfig round-trip', () => {
		setStoredConfig('org/my-integration', { apiKey: 'saved' });
		expect(getStoredConfig('org/my-integration')).toEqual({ apiKey: 'saved' });
	});

	it('stores multiple integrations independently', () => {
		setStoredConfig('org/a', { x: 1 });
		setStoredConfig('org/b', { y: 2 });
		expect(getStoredConfig('org/a')).toEqual({ x: 1 });
		expect(getStoredConfig('org/b')).toEqual({ y: 2 });
	});

	it('setStoredConfig preserves other integrations', () => {
		setStoredConfig('org/a', { x: 1 });
		setStoredConfig('org/b', { y: 2 });
		expect(getStoredConfig('org/a')).toEqual({ x: 1 });
		const raw = localStorage.getItem(STORAGE_KEY);
		expect(raw).toBeTruthy();
		expect(JSON.parse(raw!)).toEqual({
			'org/a': { x: 1 },
			'org/b': { y: 2 }
		});
	});

	it('resolveInitialConfig prefers explicit config over localStorage', () => {
		setStoredConfig('org/my-integration', { apiKey: 'from-storage' });
		expect(resolveInitialConfig(integration, { apiKey: 'from-yaml' })).toEqual({
			apiKey: 'from-yaml'
		});
	});

	it('resolveInitialConfig uses localStorage when no explicit config', () => {
		setStoredConfig('org/my-integration', { apiKey: 'from-storage' });
		expect(resolveInitialConfig(integration)).toEqual({ apiKey: 'from-storage' });
	});

	it('resolveInitialConfig returns undefined when no explicit config and no localStorage', () => {
		expect(resolveInitialConfig(integration)).toBeUndefined();
	});

	it('getStoredConfig returns undefined when localStorage throws', () => {
		vi.stubGlobal('localStorage', {
			getItem: () => {
				throw new Error('quota exceeded');
			},
			setItem: vi.fn(),
			removeItem: vi.fn(),
			clear: vi.fn()
		});
		const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
		expect(getStoredConfig('org/my-integration')).toBeUndefined();
		errorSpy.mockRestore();
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/config-storage.test.ts --run`

Expected: FAIL — module `./config-storage.js` not found

- [ ] **Step 3: Implement storage module**

```typescript
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { getPath } from '$lib/utils';

import type { CustomChecksResponse } from '@/pocketbase/types';

import { createStorageHandlers } from '@/utils/storage';

type CustomIntegrationConfigStore = Record<string, Record<string, unknown>>;

const STORAGE_KEY = 'pipeline_custom_integration_configs';
const storage = createStorageHandlers<CustomIntegrationConfigStore>(STORAGE_KEY, localStorage);

export function getStoredConfig(checkId: string): Record<string, unknown> | undefined {
	try {
		return storage.get()?.[checkId];
	} catch (error) {
		console.error('Failed to get custom integration config:', error);
		return undefined;
	}
}

export function setStoredConfig(checkId: string, config: Record<string, unknown>): void {
	try {
		const current = storage.get() ?? {};
		storage.set({ ...current, [checkId]: config });
	} catch (error) {
		console.error('Failed to set custom integration config:', error);
	}
}

export function resolveInitialConfig(
	integration: CustomChecksResponse,
	explicitConfig?: Record<string, unknown>
): Record<string, unknown> | undefined {
	if (explicitConfig !== undefined) return explicitConfig;
	return getStoredConfig(getPath(integration, true)) ?? undefined;
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/config-storage.test.ts --run`

Expected: PASS (8 tests)

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/custom-integration/config-storage.ts \
        webapp/src/lib/pipeline-form/steps/custom-integration/config-storage.test.ts
git commit -m "$(cat <<'EOF'
feat: add localStorage helpers for custom integration pipeline config

Persist last-committed integration config per checkId for reuse when
adding pipeline steps.
EOF
)"
```

---

### Task 2: Wire storage into CustomIntegrationStepForm

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts`

- [ ] **Step 1: Write failing form tests**

Add to `custom-integration-step-form.test.ts` (keep existing mocks; add these):

```typescript
vi.mock('$lib/utils', () => ({
	getPath: vi.fn((record: { canonified_name?: string }, trim?: boolean) =>
		trim ? (record.canonified_name ?? '') : (record.canonified_name ?? '')
	)
}));

vi.mock('./config-storage.js', () => ({
	getStoredConfig: vi.fn(),
	resolveInitialConfig: vi.fn((integration, explicitConfig) => {
		if (explicitConfig !== undefined) return explicitConfig;
		return undefined;
	}),
	setStoredConfig: vi.fn()
}));

import { createJsonSchemaForm } from '@/components/json-schema-form';

import { resolveInitialConfig, setStoredConfig } from './config-storage.js';
```

Update `integrationWithSchema` fixture:

```typescript
const integrationWithSchema = {
	id: 'ci2',
	name: 'With Schema',
	canonified_name: 'org/with-schema',
	input_json_schema: { type: 'object', properties: { apiKey: { type: 'string' } } },
	input_json_sample: { apiKey: 'sample-value' }
} as CustomChecksResponse;
```

Add new tests inside `describe('CustomIntegrationStepForm')`:

```typescript
beforeEach(() => {
	vi.mocked(validate).mockReset();
	vi.mocked(validate).mockReturnValue({ errors: [] } as never);
	vi.mocked(createJsonSchemaForm).mockClear();
	vi.mocked(resolveInitialConfig).mockReset();
	vi.mocked(resolveInitialConfig).mockImplementation((_integration, explicitConfig) => {
		if (explicitConfig !== undefined) return explicitConfig;
		return undefined;
	});
	vi.mocked(setStoredConfig).mockReset();
});

it('selectIntegration does not pass input_json_sample to createJsonSchemaForm', () => {
	const form = new CustomIntegrationStepForm({ intent: 'add' });
	form.selectIntegration(integrationWithSchema);
	expect(createJsonSchemaForm).toHaveBeenCalledWith(integrationWithSchema.input_json_schema, {
		hideTitle: true,
		initialValue: undefined
	});
});

it('selectIntegration uses resolveInitialConfig without explicit config', () => {
	const form = new CustomIntegrationStepForm({ intent: 'add' });
	form.selectIntegration(integrationWithSchema);
	expect(resolveInitialConfig).toHaveBeenCalledWith(integrationWithSchema, undefined);
});

it('constructor passes explicit config to resolveInitialConfig in edit mode', () => {
	vi.mocked(resolveInitialConfig).mockReturnValue({ apiKey: 'from-yaml' });
	new CustomIntegrationStepForm({
		intent: 'edit',
		initial: {
			integration: integrationWithSchema,
			config: { apiKey: 'from-yaml' }
		}
	});
	expect(resolveInitialConfig).toHaveBeenCalledWith(integrationWithSchema, {
		apiKey: 'from-yaml'
	});
});

it('commit persists config to localStorage', () => {
	vi.mocked(validate).mockReturnValue({ errors: [] } as never);
	const onSubmit = vi.fn();
	const form = new CustomIntegrationStepForm({
		intent: 'add',
		initial: { integration: integrationWithSchema }
	});
	form.onSubmit(onSubmit);
	form.commit();
	expect(onSubmit).toHaveBeenCalledOnce();
	expect(setStoredConfig).toHaveBeenCalledWith('org/with-schema', { apiKey: 'abc' });
});

it('commit does not persist when payload is invalid', () => {
	vi.mocked(validate).mockReturnValue({ errors: [{ message: 'required' }] } as never);
	const form = new CustomIntegrationStepForm({
		intent: 'add',
		initial: { integration: integrationWithSchema }
	});
	form.commit();
	expect(setStoredConfig).not.toHaveBeenCalled();
});
```

Note: move the shared `beforeEach` reset block so it covers both old and new tests (merge with existing `beforeEach`).

- [ ] **Step 2: Run tests to verify new tests fail**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts --run`

Expected: FAIL on `selectIntegration does not pass input_json_sample` (still passes `input_json_sample`)

- [ ] **Step 3: Update form class**

Replace full contents of `custom-integration-step-form.svelte.ts` relevant sections:

**Add imports:**

```typescript
import { getPath } from '$lib/utils';

import { resolveInitialConfig, setStoredConfig } from './config-storage.js';
```

**Add `commit` override** (after `getSubmitData`, before `selectIntegration`):

```typescript
commit(data?: CustomIntegrationStepFormData) {
	const payload = data ?? this.getSubmitData();
	if (payload === undefined) return;
	super.commit(payload);
	if (payload.config && payload.integration) {
		setStoredConfig(getPath(payload.integration, true), payload.config);
	}
}
```

**Update `initJsonSchemaForm`:**

```typescript
private initJsonSchemaForm(
	integration: CustomChecksResponse,
	initialConfig?: Record<string, unknown>
) {
	const schema = integration.input_json_schema;
	if (schema) {
		this.jsonSchemaForm = createJsonSchemaForm(schema as object, {
			hideTitle: true,
			initialValue: resolveInitialConfig(integration, initialConfig)
		});
	} else {
		this.jsonSchemaForm = undefined;
	}
}
```

Remove `integration.input_json_sample` fallback entirely.

- [ ] **Step 4: Run all custom-integration tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/custom-integration/ --run`

Expected: PASS (all tests in module)

- [ ] **Step 5: Run typecheck**

Run: `cd webapp && bun run check`

Expected: 0 errors

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.svelte.ts \
        webapp/src/lib/pipeline-form/steps/custom-integration/custom-integration-step-form.test.ts
git commit -m "$(cat <<'EOF'
feat: persist and restore custom integration config in pipeline form

Pre-fill from schema defaults or localStorage instead of input_json_sample;
save last committed config per integration on add and edit.
EOF
)"
```

---

### Task 3: Manual verification

**Files:** None (browser QA only)

- [ ] **Step 1: Start dev stack** (if not running)

Run: `make dev`

- [ ] **Step 2: Add mode — schema defaults only**

1. Open pipeline composer (`/my/pipelines/new`)
2. Add a custom-check step for an integration that has `input_json_sample` **and** schema `default` values
3. Confirm fields show schema defaults, **not** sample data
4. Confirm Add step is disabled until required fields are valid

- [ ] **Step 3: Add mode — localStorage restore**

1. Fill config, click Add step
2. Add another custom-check step, select the **same** integration
3. Confirm previously committed values are pre-filled

- [ ] **Step 4: Edit mode — YAML wins**

1. Edit an existing custom-check step that has saved `with.config` in YAML
2. Confirm YAML config is shown (not a different localStorage value)

- [ ] **Step 5: Edit mode — updates localStorage**

1. Change a field, click Save
2. Add a new step with the same integration
3. Confirm updated values are pre-filled

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| No `input_json_sample` in pipeline form | Task 2 |
| Schema defaults when no localStorage | Task 2 (`undefined` initialValue) |
| localStorage restore on re-select | Task 1 + Task 2 |
| Edit mode uses YAML config | Task 2 (`explicitConfig` in constructor) |
| Persist on add + edit commit | Task 2 (`commit` override) |
| Per-integration browser-wide key | Task 1 (`checkId` via `getPath`) |
| Stale config loads with validation errors | Existing `isValid` — no new code |
| Graceful localStorage errors | Task 1 |
| Out of scope: run page | — |

---

## Manual test integrations (dev)

Use integrations visible in local dev (from prior QA):

- **No schema:** LoTL check — auto-add, no config form
- **With schema + sample:** Multistep Issuer Test — verify sample is **not** pre-filled
- Re-select after commit — values restored from localStorage

Login: local account used in prior QA session.
