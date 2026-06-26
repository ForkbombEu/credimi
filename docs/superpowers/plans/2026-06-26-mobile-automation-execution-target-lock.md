# Mobile-Automation Execution Target Lock Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix wallet-action composer locking so a single mobile-automation step is fully editable, while wallet/version/runner lock on second+ add and on edit when 2+ mobile steps exist.

**Architecture:** Lock rules live in `execution-target/rules.ts` (`shouldLockFormFields`). `StepsBuilder` computes lock at form open and passes `lockExecutionTarget` explicitly through `InitFormOptions`. Sync helpers (`syncAfterStepsChange`, `establishFromStep`, etc.) live in `execution-target/sync.ts`. Raw `state` and `hasGlobalRunner` are no longer exported.

**Tech Stack:** Svelte 5 runes, Vitest (`cd webapp && bun run test:unit -- --run`), Paraglide, existing `EnrichedStep` tuples.

**Design spec:** `docs/superpowers/specs/2026-06-26-mobile-automation-execution-target-lock-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/execution-target/rules.ts` | `countMobileSteps`, `shouldLockFormFields` |
| `webapp/src/lib/pipeline-form/execution-target/sync.ts` | Prefill, establish, sync, getters |
| `webapp/src/lib/pipeline-form/execution-target/state.svelte.ts` | Internal `$state` only |
| `webapp/src/lib/pipeline-form/execution-target/index.ts` | Public barrel |
| `webapp/src/lib/pipeline-form/execution-target/rules.test.ts` | Lock matrix unit tests |
| `webapp/src/lib/pipeline-form/execution-target/sync.test.ts` | Sync/prefill unit tests |
| `webapp/src/lib/pipeline-form/steps/types.ts` | `lockExecutionTarget` on `InitFormOptions` |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts` | Pass lock flag; call sync on delete/edit |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts` | Store lock flag; use public API |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte` | Discard/picker gating |
| `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts` | `getCurrentWallet()` |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts` | Lock flag + API tests |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts` | Builder integration tests |

---

### Task 1: Lock rules (`execution-target/rules.ts`)

**Files:**
- Create: `webapp/src/lib/pipeline-form/execution-target/rules.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/rules.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
// rules.test.ts
import { describe, expect, it } from 'vitest';
import type { EnrichedStep } from '../steps-builder/types';
import { countMobileSteps, shouldLockFormFields } from './rules';

function mobileStep(): EnrichedStep {
	return [{ use: 'mobile-automation', id: 's1', continue_on_error: false, with: {} }, {}];
}

function debugStep(): EnrichedStep {
	return [{ use: 'debug' }, {}];
}

describe('countMobileSteps', () => {
	it('counts only mobile-automation steps', () => {
		expect(countMobileSteps([debugStep(), mobileStep(), mobileStep()])).toBe(2);
	});
});

describe('shouldLockFormFields', () => {
	it('add with 0 mobile steps → unlocked', () => {
		expect(shouldLockFormFields({ intent: 'add', steps: [] })).toBe(false);
	});

	it('add with 1+ mobile steps → locked', () => {
		expect(shouldLockFormFields({ intent: 'add', steps: [mobileStep()] })).toBe(true);
	});

	it('edit with 1 mobile step → unlocked', () => {
		expect(shouldLockFormFields({ intent: 'edit', steps: [mobileStep()] })).toBe(false);
	});

	it('edit with 2+ mobile steps → locked', () => {
		expect(
			shouldLockFormFields({ intent: 'edit', steps: [mobileStep(), mobileStep()] })
		).toBe(true);
	});
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/execution-target/rules.test.ts`  
Expected: FAIL — module not found

- [ ] **Step 3: Implement rules**

```ts
// rules.ts
import type { FormIntent } from '../steps/types';
import type { EnrichedStep } from '../steps-builder/types';

export function countMobileSteps(steps: EnrichedStep[]): number {
	return steps.filter(([raw]) => raw.use === 'mobile-automation').length;
}

export function shouldLockFormFields(opts: {
	intent: FormIntent;
	steps: EnrichedStep[];
}): boolean {
	const count = countMobileSteps(opts.steps);
	return opts.intent === 'add' ? count >= 1 : count >= 2;
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/execution-target/rules.test.ts`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/execution-target/rules.ts webapp/src/lib/pipeline-form/execution-target/rules.test.ts
git commit -m "feat(pipeline-form): add execution target lock rules"
```

---

### Task 2: Sync helpers (`execution-target/sync.ts`)

**Files:**
- Create: `webapp/src/lib/pipeline-form/execution-target/sync.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/sync.test.ts`
- Modify: `webapp/src/lib/pipeline-form/execution-target/state.svelte.ts`

- [ ] **Step 1: Make state internal in `state.svelte.ts`**

Remove exported `hasGlobalRunner` and `hasUndefinedRunner`. Keep `loadFromPipeline`, `clear`, and export a new internal getter used by sync:

```ts
export function getCurrentConfig(): Config | undefined {
	return state.current;
}

export function setCurrentConfig(config: Config | undefined) {
	state.current = config;
}
```

`loadFromPipeline` and `clear` continue to mutate `state.current` as today.

- [ ] **Step 2: Write failing sync tests**

```ts
// sync.test.ts
import { describe, expect, it, beforeEach } from 'vitest';
import { GLOBAL_RUNNER, EXTERNAL_VERSION } from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import type { EnrichedStep } from '../steps-builder/types';
import type { WalletActionStepData } from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import * as ExecutionTarget from './index';

const wallet = { id: 'w1', name: 'Wallet' } as WalletActionStepData['wallet'];
const version = EXTERNAL_VERSION;
const runner = GLOBAL_RUNNER;
const action = { id: 'a1', name: 'Act' } as WalletActionStepData['action'];

function mobileTuple(data: WalletActionStepData): EnrichedStep {
	return [
		{ use: 'mobile-automation', id: 's1', continue_on_error: false, with: {} },
		data as never
	];
}

describe('ExecutionTarget sync', () => {
	beforeEach(() => ExecutionTarget.clear());

	it('getAddFormPrefill returns undefined when no target', () => {
		expect(ExecutionTarget.getAddFormPrefill()).toBeUndefined();
	});

	it('establishFromStep sets prefill', () => {
		ExecutionTarget.establishFromStep({ wallet, version, runner, action });
		expect(ExecutionTarget.getAddFormPrefill()).toEqual({ wallet, version, runner });
	});

	it('syncAfterStepsChange clears when no mobile steps', () => {
		ExecutionTarget.establishFromStep({ wallet, version, runner, action });
		ExecutionTarget.syncAfterStepsChange([]);
		expect(ExecutionTarget.getAddFormPrefill()).toBeUndefined();
	});

	it('syncAfterStepsChange updates from sole mobile step', () => {
		const data: WalletActionStepData = {
			wallet,
			version,
			runner: { name: 'R', path: 'org/r', isOwned: true, isPublished: true, isOnline: true },
			action
		};
		ExecutionTarget.syncAfterStepsChange([mobileTuple(data)]);
		expect(ExecutionTarget.getAddFormPrefill()?.runner).toEqual(data.runner);
	});

	it('getCurrentWallet returns wallet from target', () => {
		ExecutionTarget.establishFromStep({ wallet, version, runner, action });
		expect(ExecutionTarget.getCurrentWallet()).toEqual(wallet);
	});

	it('shouldDefaultRunnerToGlobal when runner is global or undefined', () => {
		ExecutionTarget.establishFromStep({ wallet, version, runner: GLOBAL_RUNNER, action });
		expect(ExecutionTarget.shouldDefaultRunnerToGlobal()).toBe(true);
	});

	it('shouldOfferChooseRunnerLater false when locked', () => {
		expect(ExecutionTarget.shouldOfferChooseRunnerLater(true)).toBe(false);
	});
});
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/execution-target/sync.test.ts`  
Expected: FAIL

- [ ] **Step 4: Implement `sync.ts`**

```ts
import { isError } from 'effect/Predicate';
import type { HubItem } from '$lib/hub';
import type { EnrichedStep } from '../steps-builder/types';
import type {
	SelectedRunner,
	SelectedVersion,
	WalletActionStepData
} from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import { GLOBAL_RUNNER } from '../steps/wallet-action/wallet-action-step-form.svelte.js';
import { countMobileSteps } from './rules';
import { clear, getCurrentConfig, setCurrentConfig, type Config } from './state.svelte';

export function getAddFormPrefill(): Pick<WalletActionStepData, 'wallet' | 'version' | 'runner'> | undefined {
	const current = getCurrentConfig();
	if (!current) return undefined;
	return { wallet: current.wallet, version: current.version, runner: current.runner };
}

export function getCurrentWallet(): HubItem | undefined {
	return getCurrentConfig()?.wallet;
}

export function establishFromStep(data: WalletActionStepData) {
	setCurrentConfig({ wallet: data.wallet, version: data.version, runner: data.runner });
}

export function shouldDefaultRunnerToGlobal(): boolean {
	const runner = getCurrentConfig()?.runner;
	return runner === GLOBAL_RUNNER || runner === undefined;
}

export function shouldOfferChooseRunnerLater(lockExecutionTarget: boolean): boolean {
	if (lockExecutionTarget) return false;
	return getCurrentConfig()?.runner === undefined;
}

export function syncVersionIfSameWallet(walletId: string, version: SelectedVersion) {
	const current = getCurrentConfig();
	if (current?.wallet.id === walletId) {
		setCurrentConfig({ ...current, version });
	}
}

function walletActionDataFromStep(tuple: EnrichedStep): WalletActionStepData | undefined {
	const [raw, data] = tuple;
	if (raw.use !== 'mobile-automation') return undefined;
	if (isError(data)) return undefined;
	return data as unknown as WalletActionStepData;
}

export function syncAfterStepsChange(steps: EnrichedStep[]) {
	const count = countMobileSteps(steps);
	if (count === 0) {
		clear();
		return;
	}
	if (count === 1) {
		const tuple = steps.find(([raw]) => raw.use === 'mobile-automation');
		if (!tuple) return;
		const data = walletActionDataFromStep(tuple);
		if (!data) return;
		setCurrentConfig({ wallet: data.wallet, version: data.version, runner: data.runner });
	}
}
```

- [ ] **Step 5: Update `index.ts` public barrel**

```ts
export { loadFromPipeline, clear } from './state.svelte';
export { countMobileSteps, shouldLockFormFields } from './rules';
export {
	getAddFormPrefill,
	getCurrentWallet,
	establishFromStep,
	shouldDefaultRunnerToGlobal,
	shouldOfferChooseRunnerLater,
	syncAfterStepsChange,
	syncVersionIfSameWallet
} from './sync';
```

Use `export * as ExecutionTarget from './index'` pattern — update `index.ts` to re-export named functions (not `export * as` from state).

Current `index.ts` is `export * as ExecutionTarget from './state.svelte'`. Change to:

```ts
import * as state from './state.svelte';
import * as rules from './rules';
import * as sync from './sync';

export const ExecutionTarget = {
	loadFromPipeline: state.loadFromPipeline,
	clear: state.clear,
	countMobileSteps: rules.countMobileSteps,
	shouldLockFormFields: rules.shouldLockFormFields,
	getAddFormPrefill: sync.getAddFormPrefill,
	getCurrentWallet: sync.getCurrentWallet,
	establishFromStep: sync.establishFromStep,
	shouldDefaultRunnerToGlobal: sync.shouldDefaultRunnerToGlobal,
	shouldOfferChooseRunnerLater: sync.shouldOfferChooseRunnerLater,
	syncAfterStepsChange: sync.syncAfterStepsChange,
	syncVersionIfSameWallet: sync.syncVersionIfSameWallet
};
```

- [ ] **Step 6: Run sync tests**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/execution-target/`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add webapp/src/lib/pipeline-form/execution-target/
git commit -m "feat(pipeline-form): add execution target sync API"
```

---

### Task 3: `InitFormOptions.lockExecutionTarget`

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/types.ts`

- [ ] **Step 1: Add optional field**

```ts
export type InitFormOptions<Deserialized = unknown> = {
	intent?: FormIntent;
	initial?: Deserialized;
	lockExecutionTarget?: boolean;
};
```

- [ ] **Step 2: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/types.ts
git commit -m "refactor(pipeline-form): add lockExecutionTarget to InitFormOptions"
```

---

### Task 4: `StepsBuilder` integration

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts`

- [ ] **Step 1: Write failing builder test**

Add to `steps-builder.test.ts` (mock execution-target):

```ts
vi.mock('../execution-target/index.js', () => ({
	ExecutionTarget: {
		shouldLockFormFields: vi.fn(() => false),
		syncAfterStepsChange: vi.fn()
	}
}));
```

Test that `initAddStep` / `initEditStep` passes lock flag — spy on wallet config `initForm` or inspect via internal state after open. Simpler approach: test `openForm` path by mocking a config:

```ts
it('passes lockExecutionTarget from shouldLockFormFields to initForm', () => {
	const initForm = vi.fn(() => ({ onSubmit: vi.fn() }));
	vi.mocked(ExecutionTarget.shouldLockFormFields).mockReturnValue(true);
	// inject mock config into pipelinestep.configs temporarily or call openForm via internal API
});
```

Pragmatic v1: rely on `rules.test.ts` for matrix + manual verification in wallet form test. Add builder test only for `deleteStep` calling `syncAfterStepsChange`:

```ts
it('deleteStep calls syncAfterStepsChange', () => {
	const builder = createBuilder();
	(builder as BuilderInternal).state.steps = [
		[{ use: 'mobile-automation', id: '', continue_on_error: false, with: {} }, {}]
	];
	builder.deleteStep(0);
	expect(ExecutionTarget.syncAfterStepsChange).toHaveBeenCalledOnce();
});
```

- [ ] **Step 2: Update `openForm` in `steps-builder.svelte.ts`**

```ts
const lockExecutionTarget =
	config.use === 'mobile-automation'
		? ExecutionTarget.shouldLockFormFields({ intent, steps: state.steps })
		: undefined;

form = config.initForm({
	intent,
	initial: opts.initial as never,
	lockExecutionTarget
});
```

In submit handler after edit save:

```ts
if (config.use === 'mobile-automation') {
	ExecutionTarget.syncAfterStepsChange(inner.steps);
}
```

In `deleteStep`:

```ts
deleteStep(index: number) {
	this.stateManager.run((state) => {
		state.steps.splice(index, 1);
		ExecutionTarget.syncAfterStepsChange(state.steps);
	});
}
```

Replace `ExecutionTarget.syncVersionIfSameWallet` import usage — still valid on namespace.

- [ ] **Step 3: Run tests**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/steps-builder/steps-builder.test.ts`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts webapp/src/lib/pipeline-form/steps-builder/steps-builder.test.ts
git commit -m "feat(pipeline-form): wire execution target lock and sync in StepsBuilder"
```

---

### Task 5: `WalletActionStepForm` class

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts`

- [ ] **Step 1: Write failing test for lock flag**

```ts
it('stores lockExecutionTarget from opts', () => {
	const form = new WalletActionStepForm({ lockExecutionTarget: true });
	expect(form.lockExecutionTarget).toBe(true);
});

it('prefills add from getAddFormPrefill', () => {
	vi.spyOn(ExecutionTarget, 'getAddFormPrefill').mockReturnValue({
		wallet: { id: 'w1', name: 'W' } as never,
		version: EXTERNAL_VERSION,
		runner: GLOBAL_RUNNER
	});
	const form = new WalletActionStepForm({ intent: 'add' });
	expect(form.data.wallet?.id).toBe('w1');
	expect(form.data.action).toBeUndefined();
});
```

- [ ] **Step 2: Implement class changes**

```ts
readonly lockExecutionTarget: boolean;

constructor(opts?: InitFormOptions<WalletActionStepData>) {
	super(opts);
	this.lockExecutionTarget = opts?.lockExecutionTarget ?? false;
	if (opts?.initial) {
		this.data = { ...opts.initial };
	} else {
		const prefill = ExecutionTarget.getAddFormPrefill();
		if (prefill) {
			this.data = { ...prefill, action: undefined };
		}
	}
}

// selectWallet/selectVersion/selectExternalVersion:
if (ExecutionTarget.shouldDefaultRunnerToGlobal()) {
	this.data.runner = 'global';
}

// selectAction:
ExecutionTarget.establishFromStep({
	wallet: this.data.wallet!,
	version: this.data.version!,
	runner: this.data.runner!,
	action
} as WalletActionStepData);
```

- [ ] **Step 3: Run tests**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts
git commit -m "feat(pipeline-form): wallet form uses execution target API and lock flag"
```

---

### Task 6: Wallet form UI (`wallet-action-step-form.svelte`)

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte`

- [ ] **Step 1: Replace discard gating**

Remove `isRunnerGlobal` derived. Use `form.lockExecutionTarget`:

```svelte
onDiscard={form.lockExecutionTarget ? undefined : () => form.removeWallet()}
```

Same for version and runner.

- [ ] **Step 2: Gate picker states when locked**

Wrap picker branches so when `form.lockExecutionTarget` and wallet/version/runner are set, do not render `select-wallet`, `select-version`, `select-runner` sections. The `state` derived will naturally be `select-action` or `ready` when pre-filled.

- [ ] **Step 3: Update chooseRunnerLater**

```svelte
{#if ExecutionTarget.shouldOfferChooseRunnerLater(form.lockExecutionTarget)}
```

- [ ] **Step 4: Run svelte autofixer / lint**

Run: `cd webapp && bun run check` (or lint scoped files)  
Expected: no new errors

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte
git commit -m "fix(pipeline-form): lock wallet target fields from explicit lock flag"
```

---

### Task 7: Conformance-check migration

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts`

- [ ] **Step 1: Replace `ExecutionTarget.state.current?.wallet` with `getCurrentWallet()`**

```ts
() => ExecutionTarget.getCurrentWallet()?.id,
// ...
const wallet = ExecutionTarget.getCurrentWallet();
```

- [ ] **Step 2: Run conformance tests**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts`  
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts
git commit -m "refactor(pipeline-form): conformance check uses getCurrentWallet"
```

---

### Task 8: Fix remaining callers and full suite

**Files:**
- Modify: any file still referencing `ExecutionTarget.state`, `hasGlobalRunner`, `hasUndefinedRunner`
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.test.ts` mock if needed

- [ ] **Step 1: Grep and fix**

Run: `rg "ExecutionTarget\.(state|hasGlobalRunner|hasUndefinedRunner)" webapp/src/lib/pipeline-form`  
Expected: no matches

- [ ] **Step 2: Run full pipeline-form unit tests**

Run: `cd webapp && bun run test:unit -- --run src/lib/pipeline-form/`  
Expected: PASS

- [ ] **Step 3: Commit any fixes**

```bash
git add -A webapp/src/lib/pipeline-form/
git commit -m "chore(pipeline-form): remove deprecated execution target exports"
```

---

## Spec self-review

| Spec requirement | Task |
|------------------|------|
| Lock matrix in `shouldLockFormFields` | Task 1 |
| Explicit `lockExecutionTarget` via `InitFormOptions` | Tasks 3–6 |
| Public API only from execution-target | Task 2 |
| `syncAfterStepsChange` on delete/edit | Task 4 |
| Single-step edit unlocked | Task 1 + 6 |
| Conformance `getCurrentWallet` | Task 7 |
| No raw `state` export | Task 2 + 8 |
