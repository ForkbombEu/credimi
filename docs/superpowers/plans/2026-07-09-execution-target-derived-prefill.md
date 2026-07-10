# Execution Target Derived Prefill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the mutable `ExecutionTarget` singleton with step-derived prefill, explicit lock rules passed via `initForm`, acyclic shared types, and preserved bulk wallet-version sync.

**Architecture:** Shared leaf types in `pipeline-form/shared/`; pure functions in `execution-target/` (`resolve`, `lock`, `sync-mobile-versions`); `StepsBuilder` derives target and wires `getExecutionTarget` / `isExecutionTargetLocked` into `initForm`; step forms consume `form-context` only.

**Tech Stack:** Svelte 5 runes, Vitest, TypeScript, existing `walletActionStepConfig.serialize`.

**Design spec:** `docs/superpowers/specs/2026-07-09-execution-target-derived-prefill-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/shared/mobile-target.ts` | `MobileTargetFields`, `GLOBAL_RUNNER`, `EXTERNAL_VERSION`, runner/version types |
| `webapp/src/lib/pipeline-form/shared/enriched-step.ts` | `EnrichedStep`, `Enrich404Error` (moved from steps-builder/types) |
| `webapp/src/lib/pipeline-form/shared/guards.ts` | `isMobileTargetFields` type guard |
| `webapp/src/lib/pipeline-form/execution-target/types.ts` | `ExecutionTargetConfig` alias |
| `webapp/src/lib/pipeline-form/execution-target/resolve.ts` | `resolveExecutionTarget(steps)` |
| `webapp/src/lib/pipeline-form/execution-target/lock.ts` | `isExecutionTargetLocked(ctx)` |
| `webapp/src/lib/pipeline-form/execution-target/sync-mobile-versions.ts` | `syncMobileStepVersionsIfSameWallet` |
| `webapp/src/lib/pipeline-form/execution-target/index.ts` | Re-exports |
| `webapp/src/lib/pipeline-form/steps/form-context.ts` | `ExecutionTargetFormContext`, extended `InitFormOptions` |
| `webapp/src/lib/pipeline-form/steps/wallet-action/types.ts` | `WalletActionStepData` extends `MobileTargetFields` |
| `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts` | Derived target, lock wiring, bulk sync call |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts` | Lock/prefill via initForm; locked add → select-action |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte` | `form.isExecutionTargetLocked` instead of global |
| `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts` | `getExecutionTarget` instead of singleton |
| `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts` | Remove `loadFromPipeline` / `clear` |

---

### Task 1: Shared leaf types

**Files:**
- Create: `webapp/src/lib/pipeline-form/shared/mobile-target.ts`
- Create: `webapp/src/lib/pipeline-form/shared/enriched-step.ts`
- Create: `webapp/src/lib/pipeline-form/shared/guards.ts`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/types.ts`

- [ ] **Step 1: Create `mobile-target.ts`**

Move from `wallet-action-step-form.svelte.ts`:

```ts
export const GLOBAL_RUNNER = 'global' as const;
export const EXTERNAL_VERSION = 'installed_from_external_source' as const;
export type SelectedRunner = Record | typeof GLOBAL_RUNNER;
export type SelectedVersion = WalletVersionsResponse | typeof EXTERNAL_VERSION;
export type MobileTargetFields = { wallet: HubItem; version: SelectedVersion; runner: SelectedRunner };
```

- [ ] **Step 2: Create `enriched-step.ts`**

Move `EnrichedStep` and `Enrich404Error` from `steps-builder/types.ts` into `shared/enriched-step.ts`.

- [ ] **Step 3: Create `guards.ts`**

```ts
export function isMobileTargetFields(value: unknown): value is MobileTargetFields { ... }
```

- [ ] **Step 4: Update `steps-builder/types.ts`**

Re-export from shared:

```ts
export type { EnrichedStep } from '../shared/enriched-step.js';
export { Enrich404Error } from '../shared/enriched-step.js';
```

Update imports across codebase that referenced `steps-builder/types` for `EnrichedStep` — paths can stay if re-exported.

- [ ] **Step 5: Run typecheck**

Run: `cd webapp && bun run check`

Expected: PASS (or only errors from not-yet-updated wallet-action imports — fix in Task 2)

---

### Task 2: `wallet-action/types.ts` extraction

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps/wallet-action/types.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/index.ts`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/_partials/bulk-wallet-version-context.ts`
- Modify: `webapp/src/lib/pipeline-form/steps-builder/_partials/bulk-wallet-version-change.svelte`

- [ ] **Step 1: Create `wallet-action/types.ts`**

```ts
import type { MobileTargetFields } from '../../shared/mobile-target.js';
export type WalletActionStepData = MobileTargetFields & { action: WalletActionsResponse };
export type { SelectedRunner, SelectedVersion } from '../../shared/mobile-target.js';
export { GLOBAL_RUNNER, EXTERNAL_VERSION } from '../../shared/mobile-target.js';
```

- [ ] **Step 2: Update `wallet-action-step-form.svelte.ts`**

Import types from `./types.js`; remove local type definitions and `ExecutionTarget` import (wired in Task 6).

- [ ] **Step 3: Update dependent imports**

Point `bulk-wallet-version-context.ts`, `bulk-wallet-version-change.svelte`, `index.ts`, `card-details.svelte` at `wallet-action/types.ts` where they only need types/constants.

- [ ] **Step 4: Run typecheck**

Run: `cd webapp && bun run check`

---

### Task 3: `execution-target` pure modules + tests

**Files:**
- Create: `webapp/src/lib/pipeline-form/execution-target/types.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/resolve.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/resolve.test.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/lock.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/lock.test.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/sync-mobile-versions.ts`
- Create: `webapp/src/lib/pipeline-form/execution-target/sync-mobile-versions.test.ts`
- Modify: `webapp/src/lib/pipeline-form/execution-target/index.ts`
- Delete: `webapp/src/lib/pipeline-form/execution-target/state.svelte.ts`

- [ ] **Step 1: Write failing `resolve.test.ts`**

Cases: empty steps → `undefined`; one valid mobile step → config; two steps → last wins; error-enriched → `undefined`.

- [ ] **Step 2: Implement `resolve.ts`**

```ts
export function resolveExecutionTarget(steps: EnrichedStep[]): ExecutionTargetConfig | undefined {
  const mobile = steps.filter(([raw]) => raw.use === 'mobile-automation');
  const last = mobile.at(-1);
  if (!last) return undefined;
  const [, data] = last;
  if (isError(data) || !isMobileTargetFields(data)) return undefined;
  const { wallet, version, runner } = data;
  return { wallet, version, runner };
}
```

- [ ] **Step 3: Run resolve tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/execution-target/resolve.test.ts --run`

- [ ] **Step 4: Write failing `lock.test.ts`**

| intent | mobileStepCount | runner | expected |
|--------|-----------------|--------|----------|
| edit | 1 | global | false |
| add | 1 | global | true |
| add | 1 | specific | false |
| edit | 2 | global (latest) | true |
| edit | 2 | specific (latest) | false |
| add | 0 | — | false |

- [ ] **Step 5: Implement `lock.ts`**

- [ ] **Step 6: Write failing `sync-mobile-versions.test.ts`**

Two mobile steps same wallet → both versions updated and `with` re-serialized; different wallet step untouched.

- [ ] **Step 7: Implement `sync-mobile-versions.ts`**

Extract logic from current `applyBulkWalletVersion` loop + serialization; return new steps array (immutable) or mutate in place matching existing `StateManager` pattern.

- [ ] **Step 8: Update `index.ts`, delete `state.svelte.ts`**

- [ ] **Step 9: Run all execution-target tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/execution-target --run`

---

### Task 4: `form-context` + `steps/types.ts`

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps/form-context.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/types.ts`

- [ ] **Step 1: Create `form-context.ts`**

```ts
import type { MobileTargetFields } from '../shared/mobile-target.js';
import type { FormIntent } from './types.js';

export type ExecutionTargetFormContext = {
  getExecutionTarget: () => MobileTargetFields | undefined;
  isExecutionTargetLocked: () => boolean;
};

export type InitFormOptions<T> = {
  intent?: FormIntent;
  initial?: T;
} & Partial<ExecutionTargetFormContext>;
```

- [ ] **Step 2: Update `steps/types.ts`**

Remove duplicate `InitFormOptions` definition; re-export from `form-context.ts`. Keep `Form` interface `initForm` signature compatible.

- [ ] **Step 3: Run typecheck**

Run: `cd webapp && bun run check`

---

### Task 5: `StepsBuilder` wiring

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps-builder/steps-builder.svelte.ts`

- [ ] **Step 1: Add derived execution target**

```ts
executionTarget = $derived(resolveExecutionTarget(this.state.steps));
```

- [ ] **Step 2: Add `countMobileAutomationSteps(steps)` helper** (local or in execution-target)

- [ ] **Step 3: Update `openForm`**

```ts
const mobileStepCount = countMobileAutomationSteps(this.state.steps);
config.initForm({
  intent,
  initial: opts.initial as never,
  getExecutionTarget: () => this.executionTarget,
  isExecutionTargetLocked: () =>
    isExecutionTargetLocked({
      intent,
      mobileStepCount,
      target: this.executionTarget
    })
});
```

For **edit**, `mobileStepCount` includes the step being edited (full committed count).

- [ ] **Step 4: Refactor `applyBulkWalletVersion`**

Use `syncMobileStepVersionsIfSameWallet`; remove `ExecutionTarget.syncVersionIfSameWallet` call.

- [ ] **Step 5: Remove `ExecutionTarget` import**

---

### Task 6: `WalletActionStepForm` lock + prefill

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts`

- [ ] **Step 1: Write failing tests**

- Locked add with target → initial `state === 'select-action'`, data prefilled without action.
- Unlocked add with target → can still reach `select-wallet` when data cleared.
- Edit sole step with global runner → `isExecutionTargetLocked === false`.
- `selectAction` does not mutate any global store (existing edit test stays).

- [ ] **Step 2: Constructor changes**

```ts
private getExecutionTarget: () => MobileTargetFields | undefined;
isExecutionTargetLocked = false;

constructor(opts?: InitFormOptions<WalletActionStepData>) {
  super(opts);
  this.getExecutionTarget = opts?.getExecutionTarget ?? (() => undefined);
  this.isExecutionTargetLocked = opts?.isExecutionTargetLocked?.() ?? false;

  if (opts?.initial) {
    this.data = { ...opts.initial };
  } else {
    const target = this.getExecutionTarget();
    if (target) {
      this.data = { ...target, action: undefined };
    }
  }
}
```

- [ ] **Step 3: Adjust `state` derived for locked add**

When `intent === 'add' && isExecutionTargetLocked && data.wallet && data.version && data.runner && !data.action` → `'select-action'` (skip wallet/version/runner pickers).

- [ ] **Step 4: Remove `selectAction` ExecutionTarget write**

- [ ] **Step 5: Replace `defaultRunnerIfNeeded`**

Use `getExecutionTarget()` for runner defaulting when unlocked; when locked, runner already prefilled.

- [ ] **Step 6: Update `.svelte` template**

Replace `ExecutionTarget.hasGlobalRunner()` with `form.isExecutionTargetLocked` for `onDiscard` on wallet/version/runner cards.

- [ ] **Step 7: Run wallet-action tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts --run`

---

### Task 7: `ConformanceCheckStepForm`

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts`

- [ ] **Step 1: Replace singleton reads**

```ts
private getExecutionTarget: () => MobileTargetFields | undefined;

constructor(opts?: InitFormOptions<FormData>) {
  super(opts);
  this.getExecutionTarget = opts?.getExecutionTarget ?? (() => undefined);
  ...
}

walletActions = resource(
  () => this.getExecutionTarget()?.wallet?.id,
  ...
);
```

Update `testPickerNotice`, `testOptions` to use `this.getExecutionTarget()`.

- [ ] **Step 2: Remove `ExecutionTarget` import**

- [ ] **Step 3: Run conformance tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts --run`

---

### Task 8: `PipelineForm` cleanup

**Files:**
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/pipeline-form.test.ts`

- [ ] **Step 1: Remove `ExecutionTarget.loadFromPipeline` / `clear` from constructor**

- [ ] **Step 2: Update test mock**

Remove or simplify `execution-target` mock in `pipeline-form.test.ts`.

- [ ] **Step 3: Run pipeline-form tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/pipeline-form.test.ts --run`

---

### Task 9: Final validation

- [ ] **Step 1: Format webapp**

Run: `cd webapp && bun run format`

- [ ] **Step 2: Lint**

Run: `cd webapp && bun run lint`

- [ ] **Step 3: Typecheck**

Run: `cd webapp && bun run check`

- [ ] **Step 4: Unit tests (pipeline-form scope)**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form --run`

- [ ] **Step 5: Manual smoke** (see design spec)

---

## Residual risks

- **Open form during bulk version change:** rare; `getExecutionTarget()` reads live steps — acceptable.
- **Import churn:** `Enrich404Error` move requires grep for direct `steps-builder/types` imports of the class.
- **HubItemStepForm / other steps:** only wallet-action and conformance-check consume execution context; no changes expected elsewhere.
