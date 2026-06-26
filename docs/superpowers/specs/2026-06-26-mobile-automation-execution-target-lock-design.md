# Mobile-Automation Execution Target Lock — Design Spec

**Date:** 2026-06-26  
**Status:** Approved (design interview)  
**Scope:** Fix wallet-action step form locking in the pipeline composer so a single mobile-automation step remains fully editable, while wallet/version/runner are locked when adding a second+ step or editing any step in a pipeline with 2+ mobile-automation steps.

---

## Summary

The pipeline composer shares an **execution target** (wallet, version, runner) across all `mobile-automation` steps via `ExecutionTarget.state`. Today, locking of wallet/version/runner fields in the wallet-action form is tied to `ExecutionTarget.hasGlobalRunner()` (runner === `'global'`). That incorrectly locks fields when editing the **only** mobile step with a global runner, and inconsistently unlocks them when the runner was explicitly chosen.

This spec replaces that heuristic with an explicit **`lockExecutionTarget`** flag computed from form intent and mobile-step count.

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
| Approach | **Approach 1** — `StepsBuilder` passes `lockExecutionTarget` through `InitFormOptions` |
| Replace `isRunnerGlobal` for locking | Yes — use `form.lockExecutionTarget` in the Svelte template |
| Bulk wallet version UI | Unchanged — already assumes shared target across mobile steps |
| `ExecutionTarget` on delete | `clear()` when last mobile-automation step is removed |
| `ExecutionTarget` on single-step edit save | Update `state.current` from committed step data |

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

### Lock computation (`StepsBuilder`)

Add a helper (alongside `getBulkWalletVersionContext`):

```ts
function countMobileSteps(steps: EnrichedStep[]): number
```

When `openForm` is called:

```ts
const mobileCount = countMobileSteps(state.steps);

const lockExecutionTarget =
	intent === 'add'
		? mobileCount >= 1
		: mobileCount >= 2;
```

Pass to `config.initForm({ intent, initial, lockExecutionTarget })`.

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
- Expose for the Svelte component.

Constructor behavior unchanged for pre-fill from `ExecutionTarget.state.current` on add when execution target exists.

### UI (`wallet-action-step-form.svelte`)

Replace:

```svelte
const isRunnerGlobal = $derived(ExecutionTarget.hasGlobalRunner());
onDiscard={isRunnerGlobal ? undefined : () => form.removeWallet()}
```

With:

```svelte
onDiscard={form.lockExecutionTarget ? undefined : () => form.removeWallet()}
```

(Same pattern for version and runner.)

When `lockExecutionTarget` is true and wallet/version/runner are populated, skip `select-wallet`, `select-version`, and `select-runner` states — the form should remain at `select-action` or `ready` depending on whether action is set.

`chooseRunnerLater` snippet: only show when not locked and `ExecutionTarget.hasUndefinedRunner()`.

### `ExecutionTarget` sync (`steps-builder.svelte.ts`)

**On `deleteStep`:** after splice, if `countMobileSteps(steps) === 0`, call `ExecutionTarget.clear()`.

**On edit submit** (in `openForm` submit handler): when saved step is `mobile-automation` and `countMobileSteps === 1`, set `ExecutionTarget.state.current` from committed `WalletActionStepData` (wallet, version, runner).

---

## Files to change

| File | Change |
|------|--------|
| `steps/types.ts` | Add `lockExecutionTarget?: boolean` to `InitFormOptions` |
| `steps-builder/_partials/mobile-step-count.ts` (new) | `countMobileSteps(steps)` helper |
| `steps-builder/steps-builder.svelte.ts` | Compute lock flag; pass to `initForm`; sync `ExecutionTarget` on delete/edit |
| `steps/wallet-action/wallet-action-step-form.svelte.ts` | Accept and store `lockExecutionTarget` |
| `steps/wallet-action/wallet-action-step-form.svelte` | Use `lockExecutionTarget` instead of `isRunnerGlobal` for discard and picker gating |
| `steps/wallet-action/wallet-action-step-form.test.ts` | Tests for lock flag behavior |
| `steps-builder/steps-builder.test.ts` | Tests for lock flag passed on add/edit; `ExecutionTarget.clear` on delete |

---

## Edge cases

| Case | Behavior |
|------|----------|
| Delete down to 1 mobile step | Next edit opens unlocked |
| Delete all mobile steps | `ExecutionTarget.clear()`; next add is fresh |
| Single-step edit with runner = global | Wallet/version/runner editable (fixes current bug) |
| Edit with 2+ steps, change action only | Save updates action; execution target fields unchanged |
| Undo/redo after delete | Existing `StateManager` handles step list; consider whether `ExecutionTarget` needs re-sync from steps on undo — **v1: re-sync on form open only** (lock computed from current steps at open time) |

---

## Testing

| Test | Assert |
|------|--------|
| `lockExecutionTarget` on first add | `false` when `mobileStepCount === 0` |
| `lockExecutionTarget` on second add | `true` when `mobileStepCount >= 1` |
| `lockExecutionTarget` on edit, 1 step | `false` |
| `lockExecutionTarget` on edit, 2+ steps | `true` |
| Edit intent + `selectAction` | Still requires explicit `commit()` (existing test) |
| Delete last mobile step | `ExecutionTarget.clear()` called |
| Single-step edit save | `ExecutionTarget.state.current` updated |

Prefer unit tests on `.svelte.ts` classes; no E2E required for v1.

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
