# Execution Target Edge Cases Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `ExecutionTarget` lifecycle (clear on delete, sync from steps) and pipeline-level locking (2nd unchanged commit → locked; add 3rd+ prefilled+locked; multi-wallet forbids global runner).

**Architecture:** Extend `execution-target/state.svelte.ts` with `locked`, snapshot, and `syncFromSteps`. Add `getSharedExecutionTargetContext` beside bulk wallet helper. `StepsBuilder` orchestrates sync on mutations and passes `existingMobileCount` into `WalletActionStepForm`. Form discard gating reads `ExecutionTarget.state.locked` + add-3rd+ rule instead of `hasGlobalRunner()`.

**Tech Stack:** Svelte 5 runes (`$state`, `$derived`), Vitest, existing `StateManager` / `EnrichedStep` tuples.

**Design spec:** `docs/superpowers/specs/2026-06-25-execution-target-edge-cases-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.ts` | `getSharedExecutionTargetContext`, `countMobileSteps`, `hasDistinctMobileWallets` |
| `webapp/src/lib/pipeline-form/execution-target/state.svelte.ts` | `locked`, snapshot, `targetsEqual`, `syncFromSteps`, `beginSecondStepAdd`, `finishSecondStepAdd` |
| `webapp/src/lib/pipeline-form/execution-target/state.test.ts` | Unit tests for sync/lock/clear |
| `webapp/src/lib/pipeline-form/steps/types.ts` | `existingMobileCount` on `InitFormOptions` |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts` | Hook sync + snapshot + pass count to `initForm` |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts` | `isTargetLocked`, multi-wallet auto-global guard, `canSave` validation |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte` | Discard gating + hide “Choose later” when multi-wallet |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts` | Lock + multi-wallet tests |
| `webapp/src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.test.ts` | Shared context helper tests |
| `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts` | Align `loadFromPipeline` with `syncFromSteps` |

---

### Task 1: Shared execution target helper

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.ts`
- Create: `webapp/src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
// shared-execution-target-context.test.ts
import { describe, expect, it } from 'vitest';
import { GLOBAL_RUNNER, EXTERNAL_VERSION } from '../../steps/wallet-action/wallet-action-step-form.svelte.js';
import {
	countMobileSteps,
	getSharedExecutionTargetContext,
	hasDistinctMobileWallets
} from './shared-execution-target-context.js';

const walletA = { id: 'w-a', name: 'A' } as never;
const walletB = { id: 'w-b', name: 'B' } as never;
const version = { id: 'v1', tag: '1.0' } as never;
const runner = { path: 'org/runner', name: 'R' } as never;

function mobileStep(data: object) {
	return [{ use: 'mobile-automation', id: 's1', continue_on_error: false, with: {} }, data] as const;
}

describe('getSharedExecutionTargetContext', () => {
	it('returns null for zero mobile steps', () => {
		expect(getSharedExecutionTargetContext([])).toBeNull();
	});

	it('returns context when two steps share wallet, version, runner', () => {
		const data = { wallet: walletA, version, runner, action: {} };
		const steps = [mobileStep(data), mobileStep(data)];
		const ctx = getSharedExecutionTargetContext(steps);
		expect(ctx?.wallet.id).toBe('w-a');
		expect(ctx?.mobileIndices).toEqual([0, 1]);
	});

	it('returns null when wallets differ', () => {
		const steps = [
			mobileStep({ wallet: walletA, version, runner: GLOBAL_RUNNER, action: {} }),
			mobileStep({ wallet: walletB, version, runner: GLOBAL_RUNNER, action: {} })
		];
		expect(getSharedExecutionTargetContext(steps)).toBeNull();
	});
});

describe('hasDistinctMobileWallets', () => {
	it('is false for single wallet', () => {
		const steps = [mobileStep({ wallet: walletA, version, runner: GLOBAL_RUNNER, action: {} })];
		expect(hasDistinctMobileWallets(steps)).toBe(false);
	});

	it('is true for two wallets', () => {
		const steps = [
			mobileStep({ wallet: walletA, version, runner: GLOBAL_RUNNER, action: {} }),
			mobileStep({ wallet: walletB, version, runner: GLOBAL_RUNNER, action: {} })
		];
		expect(hasDistinctMobileWallets(steps)).toBe(true);
	});
});

describe('countMobileSteps', () => {
	it('counts only mobile-automation steps', () => {
		const steps = [
			mobileStep({ wallet: walletA, version, runner, action: {} }),
			[{ use: 'debug' }, {}]
		];
		expect(countMobileSteps(steps)).toBe(1);
	});
});
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.test.ts --run
```

- [ ] **Step 3: Implement helper**

```ts
// shared-execution-target-context.ts
import type { HubItem } from '$lib/hub';

import type { WalletActionStepData } from '../../steps/wallet-action/wallet-action-step-form.svelte.js';
import { EXTERNAL_VERSION, GLOBAL_RUNNER } from '../../steps/wallet-action/wallet-action-step-form.svelte.js';

import { Enrich404Error, type EnrichedStep } from '../types';

export type SharedExecutionTargetContext = {
	wallet: HubItem;
	version: WalletActionStepData['version'];
	runner: WalletActionStepData['runner'];
	mobileIndices: number[];
};

function isWalletActionData(value: unknown): value is WalletActionStepData {
	if (!value || typeof value !== 'object') return false;
	const o = value as Record<string, unknown>;
	return typeof o.wallet === 'object' && o.wallet !== null && 'version' in o && 'runner' in o;
}

export function versionKey(version: WalletActionStepData['version']): string {
	return version === EXTERNAL_VERSION ? EXTERNAL_VERSION : version.id;
}

export function runnerKey(runner: WalletActionStepData['runner']): string {
	return runner === GLOBAL_RUNNER ? GLOBAL_RUNNER : runner.path;
}

export function targetsEqual(
	a: Pick<WalletActionStepData, 'wallet' | 'version' | 'runner'>,
	b: Pick<WalletActionStepData, 'wallet' | 'version' | 'runner'>
): boolean {
	return (
		a.wallet.id === b.wallet.id &&
		versionKey(a.version) === versionKey(b.version) &&
		runnerKey(a.runner) === runnerKey(b.runner)
	);
}

export function countMobileSteps(steps: EnrichedStep[]): number {
	return steps.filter(([raw]) => raw.use === 'mobile-automation').length;
}

export function mobileWalletIds(steps: EnrichedStep[]): Set<string> {
	const ids = new Set<string>();
	for (const [raw, data] of steps) {
		if (raw.use !== 'mobile-automation') continue;
		if (data instanceof Enrich404Error || data instanceof Error) continue;
		if (!isWalletActionData(data)) continue;
		ids.add(data.wallet.id);
	}
	return ids;
}

export function hasDistinctMobileWallets(steps: EnrichedStep[]): boolean {
	return mobileWalletIds(steps).size > 1;
}

export function getSharedExecutionTargetContext(
	steps: EnrichedStep[]
): SharedExecutionTargetContext | null {
	const mobileIndices: number[] = [];
	for (let i = 0; i < steps.length; i++) {
		if (steps[i]![0].use === 'mobile-automation') mobileIndices.push(i);
	}
	if (mobileIndices.length === 0) return null;

	let wallet: HubItem | undefined;
	let version: WalletActionStepData['version'] | undefined;
	let runner: WalletActionStepData['runner'] | undefined;

	for (const i of mobileIndices) {
		const [, data] = steps[i]!;
		if (data instanceof Enrich404Error || data instanceof Error) return null;
		if (!isWalletActionData(data)) return null;

		if (!wallet) {
			wallet = data.wallet;
			version = data.version;
			runner = data.runner;
		} else if (!targetsEqual({ wallet, version: version!, runner: runner! }, data)) {
			return null;
		}
	}

	if (!wallet || version === undefined || runner === undefined) return null;
	return { wallet, version, runner, mobileIndices };
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.test.ts --run
```

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.ts \
  webapp/src/lib/pipeline-form/steps-builder/_partials/shared-execution-target-context.test.ts
git commit -m "feat(pipeline-form): add shared execution target context helper"
```

---

### Task 2: ExecutionTarget state — sync, lock, snapshot

**Files:**
- Modify: `webapp/src/lib/pipeline-form/execution-target/state.svelte.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/state.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
// state.test.ts
import { afterEach, describe, expect, it } from 'vitest';

import * as ExecutionTarget from './state.svelte.js';
import { GLOBAL_RUNNER } from '../steps/wallet-action/wallet-action-step-form.svelte.js';

const wallet = { id: 'w1', name: 'W' } as never;
const version = { id: 'v1', tag: '1' } as never;
const action = { id: 'a1', name: 'A' } as never;

function mobileTuple(overrides?: Partial<{ wallet: typeof wallet }>) {
	const data = { wallet, version, runner: GLOBAL_RUNNER, action, ...overrides };
	return [{ use: 'mobile-automation', id: 's', continue_on_error: false, with: {} }, data] as const;
}

afterEach(() => {
	ExecutionTarget.clear();
});

describe('ExecutionTarget.syncFromSteps', () => {
	it('clears when no mobile steps', () => {
		ExecutionTarget.state.current = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.syncFromSteps([[{ use: 'debug' }, {}]]);
		expect(ExecutionTarget.state.current).toBeUndefined();
		expect(ExecutionTarget.state.locked).toBe(false);
	});

	it('sets current and unlocked for one mobile step', () => {
		ExecutionTarget.syncFromSteps([mobileTuple()]);
		expect(ExecutionTarget.state.current?.wallet.id).toBe('w1');
		expect(ExecutionTarget.state.locked).toBe(false);
	});

	it('locks when two mobile steps share target', () => {
		ExecutionTarget.syncFromSteps([mobileTuple(), mobileTuple()]);
		expect(ExecutionTarget.state.locked).toBe(true);
	});
});

describe('ExecutionTarget.finishSecondStepAdd', () => {
	it('locks when submitted target matches snapshot', () => {
		const config = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.state.current = config;
		ExecutionTarget.beginSecondStepAdd();
		ExecutionTarget.finishSecondStepAdd(config);
		expect(ExecutionTarget.state.locked).toBe(true);
		expect(ExecutionTarget.state.secondStepPrefillSnapshot).toBeUndefined();
	});

	it('stays unlocked when submitted target differs', () => {
		const config = { wallet, version, runner: GLOBAL_RUNNER };
		ExecutionTarget.state.current = config;
		ExecutionTarget.beginSecondStepAdd();
		const otherWallet = { id: 'w2', name: 'W2' } as never;
		ExecutionTarget.finishSecondStepAdd({ wallet: otherWallet, version, runner: GLOBAL_RUNNER });
		expect(ExecutionTarget.state.locked).toBe(false);
	});
});
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/execution-target/state.test.ts --run
```

- [ ] **Step 3: Implement state changes**

Update `state.svelte.ts`:

```ts
import type { EnrichedStep } from '../steps-builder/types.js';
import {
	getSharedExecutionTargetContext,
	targetsEqual
} from '../steps-builder/_partials/shared-execution-target-context.js';
import { isError } from 'effect/Predicate';

export const state = $state({
	current: undefined as Config | undefined,
	locked: false,
	secondStepPrefillSnapshot: undefined as Config | undefined
});

export function clear() {
	state.current = undefined;
	state.locked = false;
	state.secondStepPrefillSnapshot = undefined;
}

export function beginSecondStepAdd() {
	if (!state.current) return;
	state.secondStepPrefillSnapshot = { ...state.current };
}

export function finishSecondStepAdd(submitted: Config) {
	if (state.secondStepPrefillSnapshot && targetsEqual(submitted, state.secondStepPrefillSnapshot)) {
		state.locked = true;
	} else {
		state.locked = false;
	}
	state.secondStepPrefillSnapshot = undefined;
}

function configFromStepData(data: WalletActionStepData): Config {
	const { wallet, version, runner } = data;
	return { wallet, version, runner };
}

export function syncFromSteps(steps: EnrichedStep[]) {
	const shared = getSharedExecutionTargetContext(steps);
	const mobileSteps = steps.filter(([raw, data]) => raw.use === 'mobile-automation' && !isError(data));

	if (mobileSteps.length === 0) {
		clear();
		return;
	}

	const lastData = mobileSteps.at(-1)![1];
	if (isError(lastData)) {
		state.current = undefined;
		state.locked = false;
		return;
	}

	state.current = configFromStepData(lastData as WalletActionStepData);
	state.locked = mobileSteps.length >= 2 && shared !== null;
}

export function loadFromPipeline(pipeline: EnrichedPipeline) {
	syncFromSteps(pipeline.steps);
}
```

Keep existing `hasGlobalRunner`, `hasUndefinedRunner`, `syncVersionIfSameWallet` unchanged.

Re-export `targetsEqual` from index if needed.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/execution-target/state.test.ts --run
```

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/execution-target/state.svelte.ts \
  webapp/src/lib/pipeline-form/execution-target/state.test.ts
git commit -m "feat(pipeline-form): sync ExecutionTarget from steps with lock state"
```

---

### Task 3: StepsBuilder hooks

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/types.ts`

- [ ] **Step 1: Extend `InitFormOptions`**

```ts
export type InitFormOptions<Deserialized = unknown> = {
	intent?: FormIntent;
	initial?: Deserialized;
	existingMobileCount?: number;
};
```

- [ ] **Step 2: Add private sync helper and wire mutations**

In `steps-builder.svelte.ts`:

```ts
import { countMobileSteps } from './_partials/shared-execution-target-context.js';

private syncExecutionTarget(steps = this.state.steps) {
	ExecutionTarget.syncFromSteps(steps);
}

private finishMobileStepSubmit(
	intent: pipelinestep.FormIntent,
	formData: GenericRecord,
	config: pipelinestep.AnyConfig
) {
	if (config.use !== 'mobile-automation') return;
	const data = formData as WalletActionStepData;
	if (intent === 'add' && ExecutionTarget.state.secondStepPrefillSnapshot) {
		ExecutionTarget.finishSecondStepAdd({
			wallet: data.wallet,
			version: data.version,
			runner: data.runner
		});
	}
	this.syncExecutionTarget();
}
```

In `openForm`, before `config.initForm`:

```ts
const existingMobileCount = countMobileSteps(state.steps);
if (config.use === 'mobile-automation' && intent === 'add' && existingMobileCount === 1) {
	ExecutionTarget.beginSecondStepAdd();
}
const form = config.initForm({
	intent,
	initial: opts.initial as never,
	existingMobileCount
});
```

In form `onSubmit` handler after mutating steps, replace bare sync with:

```ts
this.finishMobileStepSubmit(inner.mode.intent, formData, config);
```

In `deleteStep`:

```ts
deleteStep(index: number) {
	this.stateManager.run((state) => {
		state.steps.splice(index, 1);
	});
	this.syncExecutionTarget();
}
```

In `cloneStep` after splice:

```ts
this.syncExecutionTarget();
```

In `applyBulkWalletVersion` — after existing loop, call `this.syncExecutionTarget()` instead of only `syncVersionIfSameWallet` (keep version sync call too).

- [ ] **Step 3: Run existing builder tests**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/steps-builder/steps-builder.test.ts --run
```

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/types.ts \
  webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts
git commit -m "feat(pipeline-form): sync ExecutionTarget on steps builder mutations"
```

---

### Task 4: WalletActionStepForm — lock + multi-wallet

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts`

- [ ] **Step 1: Write failing tests**

Add to `wallet-action-step-form.test.ts`:

```ts
import * as ExecutionTarget from '$lib/pipeline-form/execution-target';

describe('WalletActionStepForm target lock', () => {
	afterEach(() => ExecutionTarget.clear());

	it('isTargetLocked is false for single step edit with global runner', () => {
		ExecutionTarget.state.locked = false;
		const form = new WalletActionStepForm({
			intent: 'edit',
			existingMobileCount: 1,
			initial: { wallet: { id: 'w1', name: 'W' } as never, version: EXTERNAL_VERSION, runner: GLOBAL_RUNNER, action: { id: 'a1' } as never }
		});
		expect(form.isTargetLocked).toBe(false);
	});

	it('isTargetLocked is true when ExecutionTarget locked', () => {
		ExecutionTarget.state.locked = true;
		const form = new WalletActionStepForm({
			intent: 'edit',
			existingMobileCount: 2,
			initial: { wallet: { id: 'w1', name: 'W' } as never, version: EXTERNAL_VERSION, runner: GLOBAL_RUNNER, action: { id: 'a1' } as never }
		});
		expect(form.isTargetLocked).toBe(true);
	});

	it('isTargetLocked is true when adding 3rd step', () => {
		ExecutionTarget.state.locked = false;
		const form = new WalletActionStepForm({ intent: 'add', existingMobileCount: 2 });
		expect(form.isTargetLocked).toBe(true);
	});
});

describe('WalletActionStepForm multi-wallet global runner', () => {
	it('canSave is false when distinct wallets and runner is global', () => {
		const form = new WalletActionStepForm({
			intent: 'add',
			existingMobileCount: 1,
			otherMobileWalletIds: ['w-other']
		});
		form.data = {
			wallet: { id: 'w-new', name: 'N' } as never,
			version: EXTERNAL_VERSION,
			runner: GLOBAL_RUNNER,
			action: { id: 'a1', name: 'A' } as never
		};
		expect(form.canSave()).toBe(false);
	});
});
```

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts --run
```

- [ ] **Step 3: Implement form class**

Extend `InitFormOptions` usage in constructor — store `existingMobileCount` and `otherMobileWalletIds` (new optional field on opts, populated by StepsBuilder from `mobileWalletIds` excluding current form wallet when editing).

```ts
export class WalletActionStepForm extends BaseForm<...> {
	private existingMobileCount: number;
	private otherMobileWalletIds: string[];

	constructor(opts?: InitFormOptions<WalletActionStepData> & {
		existingMobileCount?: number;
		otherMobileWalletIds?: string[];
	}) {
		super(opts);
		this.existingMobileCount = opts?.existingMobileCount ?? 0;
		this.otherMobileWalletIds = opts?.otherMobileWalletIds ?? [];
		// ... existing initial / ExecutionTarget prefill
	}

	get isTargetLocked() {
		return ExecutionTarget.state.locked || (this.intent === 'add' && this.existingMobileCount >= 2);
	}

	private shouldAllowGlobalRunner(): boolean {
		const walletId = this.data.wallet?.id;
		if (!walletId) return true;
		return !this.otherMobileWalletIds.some((id) => id !== walletId);
	}

	selectWallet(wallet: HubItem) {
		this.data.wallet = wallet;
		if (this.shouldAllowGlobalRunner() && (ExecutionTarget.hasGlobalRunner() || ExecutionTarget.hasUndefinedRunner())) {
			this.data.runner = 'global';
		}
	}

	// same guard in selectVersion / selectExternalVersion

	canSave() {
		if (this.state !== 'ready') return false;
		if (this.data.runner === GLOBAL_RUNNER && this.otherMobileWalletIds.length > 0) {
			const walletId = this.data.wallet?.id;
			if (walletId && this.otherMobileWalletIds.some((id) => id !== walletId)) return false;
		}
		return true;
	}
}
```

Pass `otherMobileWalletIds` from `openForm`:

```ts
import { mobileWalletIds } from './_partials/shared-execution-target-context.js';

const walletIds = [...mobileWalletIds(state.steps)];
const otherMobileWalletIds =
	config.use === 'mobile-automation'
		? walletIds.filter((id) => id !== (opts.initial as WalletActionStepData | undefined)?.wallet?.id)
		: undefined;
```

- [ ] **Step 4: Update svelte view**

```svelte
const isTargetLocked = $derived(form.isTargetLocked);
const showChooseRunnerLater = $derived(
	ExecutionTarget.hasUndefinedRunner() && !form.hasOtherMobileWallets
);
```

Replace `isRunnerGlobal` discard checks with `isTargetLocked`.

Expose `hasOtherMobileWallets` getter on form (`otherMobileWalletIds.length > 0`).

In `{#snippet chooseRunnerLater()}`: wrap with `{#if showChooseRunnerLater}`.

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts --run
```

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts \
  webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte \
  webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts
git commit -m "feat(pipeline-form): wallet step target lock and multi-wallet global guard"
```

---

### Task 5: Pipeline form load + full test pass

**Files:**
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts` (verify `loadFromPipeline` delegates to `syncFromSteps` — no extra change if Task 2 updated it)
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.test.ts` (extend mock with `syncFromSteps` if needed)

- [ ] **Step 1: Run pipeline-form unit tests**

```bash
cd webapp && bun run test:unit -- src/lib/pipeline-form/ --run
```

- [ ] **Step 2: Run lint/check on touched files**

```bash
cd webapp && bun run check
```

- [ ] **Step 3: Commit any mock/fixture fixes**

```bash
git add -A webapp/src/lib/pipeline-form/
git commit -m "test(pipeline-form): update ExecutionTarget mocks after sync API"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Clear on delete last mobile step | Task 2 `syncFromSteps`, Task 3 `deleteStep` |
| `locked` on 2nd unchanged commit | Task 2 `finishSecondStepAdd`, Task 3 submit hook |
| Add 3rd+ prefilled + locked | Task 4 `isTargetLocked` |
| Edit when locked — action only | Task 4 svelte discard gating |
| Multi-wallet no global runner | Task 4 `shouldAllowGlobalRunner`, `canSave`, choose-later hide |
| Keep auto-global (single wallet) | Task 4 guarded auto-global |
| `loadFromPipeline` aligned | Task 2 |
| Bulk version still works | Task 3 `applyBulkWalletVersion` + `syncVersionIfSameWallet` |

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-06-25-execution-target-edge-cases.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — implement tasks in this session with checkpoints

Which approach?
