# Remove Manual Pipeline Routes — Design Spec

**Date:** 2026-06-17  
**Status:** Approved (design interview)  
**Scope:** Remove `/new/manual` and `/edit/manual` routes; consolidate all manual YAML editing into the inline composer introduced in `2026-06-17-inline-manual-pipeline-edit-design.md`.

**Depends on:** Inline manual pipeline edit (implemented).

---

## Summary

After inline manual editing shipped, the separate full-page manual editor routes are redundant. This change:

1. **Deletes** `/my/pipelines/new/manual` and `/my/pipelines/.../edit/manual`.
2. **Creates** manual pipelines via `/new` → YAML column ⋮ → **Edit manually**.
3. **Edits** `manual: true` pipelines on `/edit` with the builder auto-opening in locked manual mode (`{ id: 'manual', editor }`), seeded from `record.yaml`.
4. **Locks** manual mode for persisted manual pipelines and enrichment-failure fallbacks — no **Back to steps**.

No redirects for legacy `/manual` URLs (bookmarks 404).

---

## Problem

Two parallel manual-editing surfaces exist:

- Inline composer (YAML column dropdown, `InlineManualEditor`)
- Full-page `PipelineFormManual` at `/manual` routes

The inline path is the intended UX. The routes add navigation complexity, duplicate validation/save logic, and force redirects (`edit/+page.ts`, `pipeline-card.svelte`) that split users away from the blocks composer.

---

## Decisions

| Topic | Decision |
|-------|----------|
| Create manual pipeline | `/new` only; user clicks **Edit manually** in YAML preview column |
| Edit `manual: true` pipeline | `/edit` only; auto-enter **locked** manual mode with `record.yaml` |
| **Back to steps** for `manual: true` | **Hidden** — locked in manual (Option A) |
| **Back to steps** for user-initiated inline manual | Shown; confirm discard if dirty (unchanged) |
| Legacy `/manual` URLs | **Remove entirely** — no redirects (Option B) |
| Top-bar **Manual mode** link | **Remove** |
| `pipeline-form-manual.svelte` | **Delete** |
| `getManualEditHref()` | **Delete** |
| Enrichment failure on edit | Auto-enter **locked** manual mode with `record.yaml` (replaces redirect to `/edit/manual`) |
| Save semantics | Unchanged — manual mode save sets `manual: true` |
| Architecture | **Approach 1** — `PipelineForm` constructor owns auto-init; `StepsBuilder.manualLocked` flag |

---

## Architecture

### Routing & cleanup

**Delete:**

- `webapp/src/routes/my/pipelines/(group)/new/manual/`
- `webapp/src/routes/my/pipelines/(group)/[...path]/edit/manual/`
- `webapp/src/lib/pipeline-form/pipeline-form-manual.svelte`

**Remove from existing files:**

- `getManualEditHref()` in `webapp/src/lib/pipeline/utils.ts`
- `manualEditHref` derived in `pipeline-form.svelte.ts`
- Top-bar **Manual mode** button in `pipeline-form.svelte`
- `manual` branch in `pipeline-card.svelte` edit href

**Update links:**

- `pipeline-card.svelte` → always `/my/pipelines/(group)/[...path]/edit`

### Edit page load (`edit/+page.ts`)

| Condition | Behavior |
|-----------|----------|
| `pipeline.manual === true` | Skip `getEnrichedPipeline`. Return minimal `{ pipeline: { record, steps: [], runtime: undefined }, startLockedManual: true }`. |
| `pipeline.manual === false` | Try `getEnrichedPipeline` as today. |
| Enrichment succeeds | Return enriched pipeline (normal blocks edit). |
| Enrichment fails | Return minimal pipeline + `startLockedManual: true`. |

Manual pipelines do not need enriched steps — side columns are grayed and non-interactive in manual mode.

### `PipelineForm` auto-init

Extend `Props`:

```ts
type Props = {
	mode: 'create' | 'edit';
	pipeline?: EnrichedPipeline;
	startLockedManual?: boolean;
};
```

In constructor, after `StepsBuilder` creation:

```ts
if (props.pipeline?.record.manual || props.startLockedManual) {
	this.stepsBuilder.enterManualMode(props.pipeline.record.yaml, { locked: true });
}
```

Edit `+page.svelte` passes `startLockedManual` from load data when present.

### `StepsBuilder` locking

Extend `enterManualMode`:

```ts
enterManualMode(initialYaml: string, options?: { locked?: boolean })
```

| State | `locked` | **Back to steps** | `exitManualMode()` |
|-------|----------|-------------------|---------------------|
| User clicks **Edit manually** | `false` | shown | works (confirm if dirty) |
| `pipeline.manual === true` on load | `true` | hidden | no-op, returns `true` |
| Enrichment failure on edit | `true` | hidden | no-op, returns `true` |

Add `isManualLocked` getter on `StepsBuilder`. Store `manualLocked` in builder state (set in `enterManualMode`, cleared only when exiting unlocked manual mode).

`steps-builder.svelte` renders **Back to steps** only when `builder.mode.id === 'manual' && !builder.isManualLocked`.

---

## UI

| Element | Change |
|---------|--------|
| Top bar **Manual mode** link | Removed |
| YAML column ⋮ → **Edit manually** | Unchanged; hidden while in manual mode |
| **Back to steps** | Shown only when `!builder.isManualLocked` |
| Side columns in manual mode | Unchanged (grayed, scrollable) |
| Top-bar Undo/Redo/Info/Parameters | Unchanged (disabled in manual mode) |
| Save | Unchanged |

### User flows

**Create manual pipeline**

1. Navigate to `/my/pipelines/new`.
2. YAML column ⋮ → **Edit manually** (unlocked).
3. Edit YAML, save → persists with `manual: true`.
4. Re-open → `/edit` opens in locked manual mode.

**Edit `manual: true` pipeline**

1. Navigate to `/edit` (from pipeline card or direct URL).
2. Builder opens in locked manual mode with `record.yaml`.
3. No **Back to steps**; edit and save YAML only.

**Edit blocks pipeline (optional inline manual)**

1. Navigate to `/edit`.
2. YAML column ⋮ → **Edit manually** (unlocked).
3. **Back to steps** available; confirm if dirty.

---

## File map

| Action | File |
|--------|------|
| Delete | `routes/.../new/manual/`, `routes/.../edit/manual/`, `pipeline-form-manual.svelte` |
| Modify | `edit/+page.ts`, `edit/+page.svelte`, `pipeline-form.svelte.ts`, `pipeline-form.svelte`, `steps-builder.svelte.ts`, `steps-builder.svelte`, `pipeline-card.svelte`, `pipeline/utils.ts` |
| Test | `steps-builder.test.ts`; optional `pipeline-form` constructor test |

---

## Error handling & edge cases

| Scenario | Behavior |
|----------|----------|
| Invalid YAML on save | Block save; sticky error bar (unchanged) |
| Navigate away with dirty YAML | `validateExit()` confirm (unchanged) |
| `exitManualMode()` when locked | No-op; returns `true` |
| Create + **Edit manually** on empty pipeline | Seed from `createPipelineYaml`; allowed |
| YAML `name` vs metadata `name` | No extra sync (same as full manual route) |
| Old `/manual` bookmarks | 404 |
| `blocks_mode` i18n | Unused after delete; optional cleanup |

---

## Testing

### Unit — `StepsBuilder`

- `enterManualMode(yaml, { locked: true })` sets `isManualLocked`
- `exitManualMode()` no-op when locked
- Unlocked: `exitManualMode()` works; dirty confirm unchanged

### Unit — `PipelineForm`

- Constructor with `record.manual === true` → locked manual mode, YAML = `record.yaml`
- Constructor with `startLockedManual: true` → same

### Smoke (manual)

1. Create → **Edit manually** → save → re-open → locked, no **Back to steps**
2. Edit existing `manual: true` → locked manual on load
3. Edit blocks pipeline → **Edit manually** → **Back to steps** works
4. No top-bar **Manual mode** link
5. `/new/manual`, `/edit/manual` → 404

E2E out of scope.

---

## Out of scope

- Redirects for legacy `/manual` URLs
- Parsing manual YAML back into `EnrichedStep[]`
- Changing `manual: true` persistence semantics
- Removing `manual` field from PocketBase schema
- E2E tests

---

## Supersedes (from inline manual spec)

The following items in `2026-06-17-inline-manual-pipeline-edit-design.md` are superseded by this spec:

- "Top-bar Manual mode link — **Keep**"
- "After save, future edits redirect to `/edit/manual`"
- "Out of scope: Removing `/edit/manual` full-page route"
