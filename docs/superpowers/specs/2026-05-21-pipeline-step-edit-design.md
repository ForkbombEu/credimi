# Pipeline Step Edit (Composer) — Design Spec

**Date:** 2026-05-21  
**Status:** Approved (design interview)  
**Scope:** Re-open the same `initForm` used when adding a pipeline step, in edit mode, from each step card in the steps composer (`webapp/src/lib/pipeline-form/`).

---

## Summary

Each editable step card gets a pencil that opens the **left “Add step” column** with the step’s `initForm`, prefilled from `EnrichedStep` data. Submit **updates** the existing step in place (preserving `id` and `continue_on_error`). **Back** discards draft changes and returns to idle.

The existing **Sheet + `EditComponent`** flow (hub/wallet PocketBase record editors) is **removed** — one unified edit path through `initForm`.

---

## Problem

Today the composer only supports **add** in the left column (`initAddStep` → `mode: form` → push on submit). Step cards expose edit only for hub and wallet steps via a **separate** `EditComponent` in a Sheet that edits underlying PB records (YAML/code), not the step-selection wizard. Utils (email, HTTP) and conformance steps have **no** edit affordance.

Users expect “edit this step” to mean “change what I picked when I added it,” using the same UI.

---

## Decisions

| Topic | Decision |
|-------|----------|
| Edit UI surface | **Left column** — same pane as add; title **“Edit step”** when editing |
| Replace Sheet `EditComponent` | **Yes** — remove `EditComponent` from configs and delete `edit-component.svelte` files |
| Implementation approach | **Approach 2** — extend `Config.initForm(opts)` + `BaseForm` contract |
| Wizard edit entry | **Ready/summary view** — all selections visible; change via discard on each field |
| Cancel / Back | **Discard** in-form changes → `idle`; step list unchanged |
| Debug steps | **No** edit button |
| Enrich failure (`Error` / `Enrich404Error`) | **No** edit button |
| Submit on edit | **Explicit Save** — wizards must not auto-commit on last pick when `intent === 'edit'` |
| Submit on add | **Unchanged** — auto-commit on final selection where applicable today |
| Step identity on save | Preserve `step.id` and `continue_on_error`; update `with` + enriched tuple `[1]` |
| Switching form while editing | **Discard** current form (same as Back) before opening add/edit |
| Visual feedback | **Highlight** the step card being edited (border/ring) |

---

## Architecture

### Mode shape (`StepsBuilder`)

```ts
type BuilderMode =
	| { id: 'idle' }
	| { id: 'form'; intent: 'add' | 'edit'; stepIndex?: number; form: pipelinestep.Form };
```

| Method | Behavior |
|--------|----------|
| `initAddStep(type)` | `config.initForm({ intent: 'add' })`; on submit → `push` step; → `idle` |
| `initEditStep(index)` | Validate step is editable; `config.initForm({ intent: 'edit', initial: stepData })`; `stepIndex = index`; → `form` |
| `exitFormState()` | Tear down `$effect.root` cleanup; → `idle` (discard) |

Submit handler branches on `intent`:

- **add:** create new `PipelineStep`, append to `steps`.
- **edit:** `steps[stepIndex][0].with = config.serialize(data)`; `steps[stepIndex][1] = data`; keep `id` / `continue_on_error`; → `idle`.

### Form contract (`steps/types.ts`)

```ts
export type FormIntent = 'add' | 'edit';

export type InitFormOptions<Deserialized> = {
	intent: FormIntent;
	initial?: Deserialized;
};

export interface Config<...> {
	initForm: (opts?: InitFormOptions<Deserialized>) => Form<Deserialized>;
	// EditComponent removed
}

export abstract class BaseForm<Deserialized, T> {
	intent: FormIntent;
	commit(data: Deserialized): void; // calls handleSubmit when valid
	// selection helpers call commit() only when intent === 'add' or via explicit Save
}
```

`initForm` defaults: `{ intent: 'add' }` when `opts` omitted (backward compatible at call sites).

### Composer UI (`steps-builder.svelte`)

- Column title: **“Add step”** (idle/add) vs **“Edit step”** (edit) — new i18n key `Edit_step`.
- **Back** (existing): calls `exitFormState()` — discard.
- **Save** footer when `intent === 'edit'`: enabled when form is valid / `ready`; calls `form.commit(currentData)` (forms expose `canSave` / `getData()` or equivalent).
- Pass `editingIndex` to `StepCard` for highlight when `mode.id === 'form' && mode.intent === 'edit'`.

### Step card (`step-card.svelte`)

- Pencil visible when step has valid `stepData` and config (not `debug`, not enrich error).
- `onclick` → `builder.initEditStep(index)`.
- Remove `Sheet`, `EditComponent` import/render.
- `editing={builder.mode.id === 'form' && builder.mode.intent === 'edit' && builder.mode.stepIndex === index}` for highlight class.

---

## Per step type

### Wallet action (`mobile-automation`)

- `initForm({ intent, initial })` copies `initial` into `data` → `state === 'ready'`.
- Summary block (wallet / version / runner / action) already shown when `data` populated.
- **Add:** `selectAction` → `commit`.
- **Edit:** `selectAction` updates `data.action` only; **Save** commits.
- Add **ready** UI for action when `state === 'ready'` and action set: show selected action `ItemCard` with discard to re-enter `select-action` (small gap today — only summary header, no action row in ready).

### Conformance check

- Prefill → `ready`; summary chips for standard / version / suite / test.
- **Add:** `selectTest` → `commit`.
- **Edit:** `selectTest` sets test; **Save** commits.
- When `ready`, show selected test in summary (test chip may need adding if missing).

### Hub steps (credential, use-case, custom-check)

- **Add:** search → `selectItem` → `commit`.
- **Edit:** show selected `ItemCard` at top (new — mirror conformance summary); search below; `selectItem` updates selection; **Save** commits.
- `initForm` receives `initial: HubItem` for edit.

### Email / HTTP utils

- Prefill `data` from `initial`.
- Existing submit button: label **Save** when `intent === 'edit'`, keep add label for add.
- `submit()` → `commit`.

### Debug

- No `initForm` edit path; no pencil.

---

## Files to change

| File | Change |
|------|--------|
| `steps/types.ts` | `InitFormOptions`, `FormIntent`; remove `EditComponent` types |
| `steps-builder/steps-builder.svelte.ts` | Mode type, `initEditStep`, submit branches, effect cleanup |
| `steps-builder/steps-builder.svelte` | Edit title, Save footer, `editingIndex` |
| `steps-builder/_partials/step-card.svelte` | Edit trigger, highlight, remove Sheet |
| `steps/wallet-action/*` | `initForm` opts, intent branching, ready action UI |
| `steps/conformance-check/*` | intent branching, ready test summary if needed |
| `steps/hub-item/*` | summary UI, intent branching; remove `EditComponent` |
| `steps/utils-steps/*` | `initForm` opts, Save label |
| `messages/en.json` (+ locales if required) | `Edit_step`, `Save` reuse or `Save_step` |
| **Delete** | `steps/wallet-action/edit-component.svelte`, `steps/hub-item/edit-component.svelte` |

---

## Error handling & edge cases

- Starting add/edit while already in `form` mode: call `exitFormState()` first (discard).
- `linkProcedure` (if configured on config): invoke when replacing serialized `with` on edit save, same as add pipeline generation if applicable.
- Undo/redo: all mutations via existing `StateManager.run`.
- Wallet global runner: preserve existing `onDiscard` rules when `ExecutionTarget.hasGlobalRunner()`.

---

## Testing

| Test | Assert |
|------|--------|
| `StepsBuilder` edit save | Step at index updated; `id` / `continue_on_error` unchanged |
| `StepsBuilder` edit Back | Steps array unchanged vs snapshot before edit |
| Wallet (or conformance) form | `intent: 'edit'` + `selectTest`/`selectAction` does not call `handleSubmit` until `commit()` |
| Hub form edit | Initial item shown in summary; Save replaces tuple |

Prefer unit tests on `.svelte.ts` classes; no E2E required for v1.

---

## Out of scope

- Editing PocketBase record YAML/code outside the step picker (former Sheet behavior).
- Inline edit in YAML preview column.
- Edit for debug steps.

---

## Open questions (resolved in interview)

| Question | Answer |
|----------|--------|
| Sheet vs initForm | Replace Sheet entirely (A) |
| Wizard edit entry | Ready/summary view (A) |
| Back behavior | Discard (A) |
| Approach | Config.initForm + BaseForm (2) |
