# Inline Manual Pipeline Edit (Composer) — Design Spec

**Date:** 2026-06-17  
**Status:** Approved (design interview)  
**Scope:** Add inline manual YAML editing inside the blocks pipeline composer (`webapp/src/lib/pipeline-form/`), without removing the existing full-page manual editor route.

---

## Summary

The YAML preview column gains a header dropdown (ellipsis menu) with **Edit manually**. Entering manual mode:

1. Seeds a `CodeEditor` with the current derived pipeline YAML.
2. Grays out (but keeps scrollable) the **Add step** and **Steps sequence** columns.
3. Hides the YAML preview column and shows a **Manual edit** column with an editable YAML editor.
4. Disables top-bar controls that do not apply in manual mode (Undo, Redo, Info, Parameters), each with a tooltip explaining why.
5. Shows validation errors in a **sticky bottom bar** inside the manual column (errors only).

**Back to steps** exits manual mode; if YAML changed, confirm discard. Steps builder state is **not** updated from manual edits.

**Save** in manual mode validates YAML (AJV + parse), persists `editor.yaml`, and sets `manual: true` (same semantics as `pipeline-form-manual.svelte`). After save, future edits redirect to `/edit/manual` per existing `+page.ts` behavior.

The top-bar **Manual mode** link (navigation to the separate full-page editor) **remains**.

---

## Problem

Power users editing pipelines in blocks mode can preview YAML but cannot edit it inline. The only manual path today is leaving the composer via **Manual mode** (separate route + `manual: true` on save). A faster inline path is needed while keeping the visual steps as context.

---

## Decisions

| Topic | Decision |
|-------|----------|
| Entry point | YAML preview column header dropdown → **Edit manually** |
| Architecture | **Approach 1** — extend `StepsBuilder` mode; **Approach 3 (partial)** — `InlineManualEditor` class for YAML field + validation |
| Manual mode shape | `{ id: 'manual'; editor: InlineManualEditor }` |
| Save | Editor YAML + `manual: true` (same as full manual route) |
| Exit manual mode | **Back to steps**; confirm if dirty; discard YAML edits; steps unchanged |
| Top-bar Manual mode link | **Keep** — still navigates to `/edit/manual` |
| Undo / Redo in manual mode | **Disabled** + tooltip |
| Info / Parameters in manual mode | **Disabled** + tooltip |
| All disabled top-bar buttons | **Tooltip on each** explaining unavailability in manual mode |
| Step form open on enter | **Auto-close** step form (discard unsaved step edits), then enter manual mode |
| Debouncing | Runed [`Debounced`](https://runed.dev/docs/utilities/debounced) — debounced value **is** validation result |
| Validation UI | Sticky bottom bar; **errors only** (no pending/valid/idle states) |
| Shared validation | `Pipeline.validateYaml()` on `$lib/pipeline`; reuse in inline editor + full manual route + save gate |

---

## Architecture

### Builder mode (`StepsBuilder`)

```ts
type BuilderMode =
	| { id: 'idle' }
	| { id: 'form'; intent: FormIntent; stepIndex?: number; config: AnyConfig; form: Form }
	| { id: 'manual'; editor: InlineManualEditor };
```

| Method | Behavior |
|--------|----------|
| `enterManualMode(initialYaml)` | If `form` mode → `exitFormState()`; create `InlineManualEditor(initialYaml)`; set `mode = { id: 'manual', editor }`; optionally `await editor.validateNow()` on enter |
| `exitManualMode()` | If `editor.isDirty` → confirm discard; `editor.dispose()`; `mode = { id: 'idle' }` |
| `isManualMode` | `mode.id === 'manual'` |

`initialYaml` comes from `PipelineForm.yamlString` (derived from metadata name, steps, runtime via `createPipelineYaml`).

### `InlineManualEditor`

```ts
class InlineManualEditor {
	yaml = $state('');
	readonly baselineYaml: string;

	private debouncedValidation = new Debounced(
		() => Pipeline.validateYaml(this.yaml),
		400
	);

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

- **No separate `validation` `$state`** — `debouncedValidation.current` is the validation result.
- **Save / pre-save gate:** `await editor.validateNow()` then check `ok`.
- **Dispose:** `cancel()` pending debounce on exit.

### `Pipeline.validateYaml(yaml: string)`

Lives in `webapp/src/lib/pipeline/validate-yaml.ts`, exported from `webapp/src/lib/pipeline/index.ts` as `validateYaml` (callable as `Pipeline.validateYaml` via `$lib`).

```ts
type PipelineYamlValidation =
	| { ok: true; value: string }
	| { ok: false; message: string };
```

- Parse YAML; on failure return parse error message.
- Validate against `schemas/pipeline/pipeline_schema.json` via AJV (same options as `pipeline-form-manual.svelte`).
- On schema failure return `ajv.errorsText(...)`.
- On success return `{ ok: true, value: yaml }` (the validated input string).
- Extract from `pipeline-form-manual.svelte` `refineAsPipelineYaml` logic; manual route calls it from Zod `superRefine`.

### `PipelineForm` integration

| Concern | Blocks mode | Manual mode (`builder.mode.id === 'manual'`) |
|---------|-------------|-----------------------------------------------|
| `save()` | `createPipelineYaml` → `yaml`, `manual` omitted/false | `await editor.validateNow()` → `yaml: result.value`, `manual: true` |
| `hasChanges` | steps / runtime / metadata diff | `editor.isDirty` OR metadata/runtime diff vs initial |
| `canSave` | `hasChanges && steps.length > 0` | `hasChanges && editor.isValid` (re-validate on save regardless) |
| `validateExit()` | existing confirm | also treat `editor.isDirty` as unsaved changes |

Metadata and runtime forms are **not** editable in manual mode (top-bar buttons disabled). Their values at enter-time are already reflected in seeded YAML; save still sends `metadataForm.value` alongside `editor.yaml`.

---

## UI

### Normal mode — YAML preview column

- Header `titleRight`: ellipsis `DropdownMenu` (mirror Steps sequence pattern in `bulk-wallet-version-change.svelte`).
- Single item: **Edit manually** → `builder.enterManualMode(form.yamlString)`.
- Hidden when already in manual mode.

### Manual mode — column layout

| Column | Behavior |
|--------|----------|
| Add step | `opacity-60`, click-blocking overlay, **scrollable** |
| Steps sequence | same |
| YAML preview | **hidden** |
| Manual edit (3rd pane) | `CodeEditor` (`lang: yaml`, flex grow, `min-width: 0` on flex parent) |

**Disabled-but-scrollable pattern:** semi-transparent overlay (`pointer-events: auto`) on interactive children; scroll container keeps `overflow-y-scroll` without `pointer-events-none` on the scroll root.

### Manual edit column header

- Title: **Manual edit** (new i18n key).
- Right: **Back to steps** button → `exitManualMode()`.

### Validation feedback

- Sticky bar at **bottom** of manual edit column.
- Render **only when** `editor.validation.ok === false`.
- Display `editor.validation.message`.
- No pending, valid, or idle indicators.

### Top bar (`pipeline-form.svelte`)

| Control | Manual mode |
|---------|-------------|
| Undo | disabled + tooltip |
| Redo | disabled + tooltip |
| Info (metadata) | disabled + tooltip |
| Parameters (runtime) | disabled + tooltip |
| Manual mode (href) | unchanged |
| Save | enabled per `canSave`; runs `validateNow()` before persist |

Use existing `Tooltip` / `IconButton` `tooltip` patterns. One i18n string (or per-control strings if clearer) explaining controls are unavailable during manual edit.

---

## File layout

```
webapp/src/lib/pipeline/
  validate-yaml.ts                       # Pipeline.validateYaml
  validate-yaml.test.ts
webapp/src/lib/pipeline-form/
  pipeline-form.svelte                   # top-bar disabled states + tooltips
  pipeline-form.svelte.ts                # save / hasChanges / canSave branches
  pipeline-form-manual.svelte            # use validate-pipeline-yaml in Zod refine
  steps-builder/
    steps-builder.svelte                 # layout modes, overlays, column swap
    steps-builder.svelte.ts              # enterManualMode / exitManualMode
    inline-manual-editor.svelte.ts       # InlineManualEditor class
    inline-manual-editor.test.ts
    _partials/
      manual-editor-column.svelte        # CodeEditor + sticky error + back button
      yaml-preview-menu.svelte           # dropdown Edit manually
```

---

## Error handling & edge cases

| Scenario | Behavior |
|----------|----------|
| Invalid YAML on Save | Block save; sticky bar shows error from `validateNow()` |
| Save API error | `showPipelineFormError` (existing) |
| Exit with dirty YAML | Confirm discard |
| Exit with clean YAML | Immediate |
| `beforeNavigate` with dirty manual YAML | Existing `validateExit` + dirty check |
| Enter while step form open | `exitFormState()` then enter |
| Empty pipeline (0 steps) | Allowed; seed includes `steps: []` |
| Post-save `manual: true` | Next edit loads `/edit/manual` (existing redirect) |
| YAML `name` vs metadata `name` | Same as full manual route — no extra sync |

---

## Testing

### Unit — `Pipeline.validateYaml`

- Valid pipeline document → `{ ok: true, value: yaml }`
- Malformed YAML → `{ ok: false, message }`
- Schema violation → `{ ok: false, message }` with AJV text

### Unit — `InlineManualEditor`

- `isDirty` when `yaml` differs from `baselineYaml`
- `validateNow()` flushes debounce via `updateImmediately()`
- `dispose()` calls `cancel()` without throwing

### Unit — `StepsBuilder`

- `enterManualMode` closes `form` mode
- `exitManualMode` returns to `idle` and disposes editor

E2E out of scope for this feature.

---

## i18n (new keys)

| Key | EN (draft) |
|-----|------------|
| `edit_manually` | Edit manually |
| `manual_edit` | Manual edit |
| `back_to_steps` | Back to steps |
| `discard_manual_yaml_changes` | Discard manual YAML changes? |
| `unavailable_in_manual_edit` | Unavailable while editing YAML manually |

(Adjust copy in implementation; per-control tooltips may reuse one string or split for Undo vs Info.)

---

## Out of scope

- Parsing manual YAML back into `EnrichedStep[]` on exit or save
- Removing `/edit/manual` full-page route
- Changing post-save redirect for `manual: true` pipelines
- E2E tests
