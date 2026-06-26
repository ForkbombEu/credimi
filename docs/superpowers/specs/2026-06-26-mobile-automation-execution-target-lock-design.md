# Mobile-Automation Execution Target Lock — Design Spec

**Date:** 2026-06-26  
**Status:** Approved (design interview)  
**Scope:** Fix wallet-action step form locking in the pipeline composer so a single mobile-automation step remains fully editable, while wallet/version/runner are locked when adding a second+ step or editing any step in a pipeline with 2+ mobile-automation steps.

---

## Summary

The pipeline composer shares an **execution target** (wallet, version, runner) across all `mobile-automation` steps via `ExecutionTarget.state`. Today, locking of wallet/version/runner fields in the wallet-action form is tied to `ExecutionTarget.hasGlobalRunner()` (runner === `'global'`). That incorrectly locks fields when editing the **only** mobile step with a global runner, and inconsistently unlocks them when the runner was explicitly chosen.

This spec replaces that heuristic with locking rules owned by the **`execution-target` module**, exposed through a small public API. Consumers call meaningful functions; raw reactive state and low-level helpers are not exported.

---

## Problem

| Scenario | Current behavior | Expected behavior |
|----------|------------------|-------------------|
| Edit the only mobile-automation step (runner = global) | Wallet/version/runner appear locked (no discard) | Fully editable |
| Add second mobile-automation step | Pre-fill works; locking depends on runner type | Wallet/version/runner locked; only action selectable |
| Edit any step when 2+ mobile steps exist | Locking depends on runner type | Wallet/version/runner locked; action editable |

The root cause is `wallet-action-step-form.svelte` using `isRunnerGlobal` (derived from `ExecutionTarget.hasGlobalRunner()`) to hide discard buttons and block changes, regardless of intent or step count.

---

## Decisions

| Topic | Decision |
|-------|----------|
| Locking rule (add) | Lock wallet/version/runner when `mobileStepCount >= 1` at form open |
| Locking rule (edit) | Lock wallet/version/runner when `mobileStepCount >= 2` at form open |
| Single-step edit | Fully unlocked — all fields editable including wallet, version, runner, action |
| Action field | Never locked by execution-target rules; always changeable via discard + re-pick or Edit action sheet |
| Module ownership | All lock/count/sync rules live in `execution-target/`; export only the public API below |
| Approach | `StepsBuilder` calls `ExecutionTarget.shouldLockFormFields(...)` and passes result through `InitFormOptions` |
| Replace `isRunnerGlobal` for locking | Yes — use `form.lockExecutionTarget` in the Svelte template |
| Stop exporting raw `state` | Yes — replace direct `ExecutionTarget.state` reads with getters (`getCurrentWallet()`, etc.) |
| Bulk wallet version UI | Unchanged — already assumes shared target across mobile steps |
| Step-list sync | `ExecutionTarget.syncAfterStepsChange(steps)` on delete and single-step edit save |

---

## Locking matrix

| Context | Mobile steps in pipeline | Wallet / version / runner | Action |
|---------|--------------------------|---------------------------|--------|
| First add | 0 (becomes 1 on save) | Editable | Editable |
| Second+ add | ≥ 1 | Locked (pre-filled from execution target) | Editable |
| Edit | 1 | Editable | Editable |
| Edit | ≥ 2 | Locked | Editable |

**Locked** means: show summary `ItemCard`s only; no discard handlers; do not render wallet/version/runner picker sections.

---

## Architecture

### `execution-target` module layout

```
execution-target/
  index.ts            # public barrel — only meaningful exports
  state.svelte.ts     # internal reactive state (not exported)
  rules.ts            # pure lock/count rules (internal)
  sync.ts             # step-list ↔ state sync (internal or thin exports)
```

`index.ts` re-exports **only** the public API. `state.svelte.ts` holds `$state`; other modules import it internally.

### Public API (`execution-target/index.ts`)

| Function | Purpose |
|----------|---------|
| `loadFromPipeline(pipeline)` | Initialize from pipeline on form mount (existing) |
| `clear()` | Reset when no mobile steps remain (existing) |
| `countMobileSteps(steps)` | Count `mobile-automation` steps in the builder list |
| `shouldLockFormFields({ intent, steps })` | **Single source of truth** for the locking matrix |
| `getAddFormPrefill()` | Wallet/version/runner for second+ add pre-fill, or `undefined` |
| `shouldDefaultRunnerToGlobal()` | Whether wallet/version selection should auto-set runner to global |
| `shouldOfferChooseRunnerLater(lockExecutionTarget)` | Whether the “Choose later” runner card is shown |
| `establishFromStep(data)` | Set execution target when first action is committed on add |
| `syncAfterStepsChange(steps)` | After delete or edit save: clear if 0 mobile steps; update from sole step if count === 1 |
| `syncVersionIfSameWallet(walletId, version)` | Bulk version change (existing) |
| `getCurrentWallet()` | Read-only accessor for conformance-check and other consumers |

**Not exported:** `state`, `hasGlobalRunner()`, `hasUndefinedRunner()`. Existing callers migrate to the functions above.

### Lock rule implementation (`rules.ts`)

```ts
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

### `StepsBuilder` integration

When `openForm` is called:

```ts
const lockExecutionTarget = ExecutionTarget.shouldLockFormFields({
	intent,
	steps: state.steps
});

config.initForm({ intent, initial, lockExecutionTarget });
```

On `deleteStep` and on mobile-automation edit save: `ExecutionTarget.syncAfterStepsChange(state.steps)`.

`syncAfterStepsChange` implementation:
- `count === 0` → `clear()`
- `count === 1` → read wallet/version/runner from the sole mobile step’s enriched data and update internal state
- `count >= 2` → no-op on save (target already established); delete may still reduce count

### Form contract (`steps/types.ts`)

```ts
export type InitFormOptions<Deserialized = unknown> = {
	intent?: FormIntent;
	initial?: Deserialized;
	lockExecutionTarget?: boolean;
};
```

### `WalletActionStepForm` (`wallet-action-step-form.svelte.ts`)

- Store `readonly lockExecutionTarget: boolean` (default `false`).
- On add without `initial`: pre-fill from `ExecutionTarget.getAddFormPrefill()` (not raw `state.current`).
- `selectWallet` / `selectVersion` / `selectExternalVersion`: use `ExecutionTarget.shouldDefaultRunnerToGlobal()` instead of `hasGlobalRunner() || hasUndefinedRunner()`.
- `selectAction` on add: call `ExecutionTarget.establishFromStep(data)` instead of assigning `state.current`.

### UI (`wallet-action-step-form.svelte`)

Replace `isRunnerGlobal` discard gating with `form.lockExecutionTarget`.

When locked and wallet/version/runner are populated, skip `select-wallet`, `select-version`, and `select-runner` states.

`chooseRunnerLater` snippet: `ExecutionTarget.shouldOfferChooseRunnerLater(form.lockExecutionTarget)`.

### Conformance-check migration

Replace `ExecutionTarget.state.current?.wallet` with `ExecutionTarget.getCurrentWallet()`.

---

## Files to change

| File | Change |
|------|--------|
| `execution-target/rules.ts` (new) | `countMobileSteps`, `shouldLockFormFields` |
| `execution-target/sync.ts` (new) | `syncAfterStepsChange`, `establishFromStep`, `getAddFormPrefill`, `getCurrentWallet`, `shouldDefaultRunnerToGlobal`, `shouldOfferChooseRunnerLater` |
| `execution-target/state.svelte.ts` | Keep internal state only; remove exported low-level helpers |
| `execution-target/index.ts` | Export public API only (no `state`, no `hasGlobalRunner`) |
| `execution-target/*.test.ts` (new) | Unit tests for rules and sync |
| `steps/types.ts` | Add `lockExecutionTarget?: boolean` to `InitFormOptions` |
| `steps-builder/steps-builder.svelte.ts` | Call `shouldLockFormFields` + `syncAfterStepsChange` |
| `steps/wallet-action/wallet-action-step-form.svelte.ts` | Use public `ExecutionTarget` API |
| `steps/wallet-action/wallet-action-step-form.svelte` | Use `lockExecutionTarget` for discard and picker gating |
| `steps/conformance-check/conformance-check-step-form.svelte.ts` | `getCurrentWallet()` instead of `state.current` |
| `steps/wallet-action/wallet-action-step-form.test.ts` | Lock flag + API usage |
| `steps-builder/steps-builder.test.ts` | Integration with `shouldLockFormFields` / `syncAfterStepsChange` |

---

## Edge cases

| Case | Behavior |
|------|----------|
| Delete down to 1 mobile step | Next edit opens unlocked |
| Delete all mobile steps | `syncAfterStepsChange` → `clear()`; next add is fresh |
| Single-step edit with runner = global | Wallet/version/runner editable (fixes current bug) |
| Edit with 2+ steps, change action only | Save updates action; execution target fields unchanged |
| Undo/redo after delete | Existing `StateManager` handles step list; consider whether `ExecutionTarget` needs re-sync from steps on undo — **v1: re-sync on form open only** (lock computed from current steps at open time) |

---

## Testing

| Test | Assert |
|------|--------|
| `shouldLockFormFields` on first add | `false` when `mobileStepCount === 0` |
| `shouldLockFormFields` on second add | `true` when `mobileStepCount >= 1` |
| `shouldLockFormFields` on edit, 1 step | `false` |
| `shouldLockFormFields` on edit, 2+ steps | `true` |
| `syncAfterStepsChange` with 0 mobile steps | calls internal clear |
| `syncAfterStepsChange` with 1 mobile step | updates target from that step’s data |
| `getAddFormPrefill` | returns prefill when target set; `undefined` when not |
| Edit intent + `selectAction` | Still requires explicit `commit()` (existing test) |

Prefer unit tests in `execution-target/*.test.ts` for rules/sync; no E2E required for v1.

---

## Out of scope

- Changing bulk wallet version change UI or semantics.
- Locking action selection across steps.
- Re-syncing `ExecutionTarget` on every undo/redo (deferred to v1 form-open computation).
- Backend / YAML validation of shared execution target invariants.

---

## Open questions (resolved in interview)

| Question | Answer |
|----------|--------|
| Edit behavior with 2+ mobile steps | Locked on edit (same as second add) — **A** |
| Recommended approach | Pass `lockExecutionTarget` from `StepsBuilder` — **Approach 1** |
| API surface | Group all rules in `execution-target/`; export only meaningful functions (no raw `state`) |
| Lock delivery to form | **Explicit** — `StepsBuilder` passes `lockExecutionTarget` via `InitFormOptions` (not session state on the module) |
