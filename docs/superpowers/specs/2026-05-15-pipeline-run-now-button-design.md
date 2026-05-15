# Pipeline Run Now Button — Design Spec

**Date:** 2026-05-15  
**Status:** Approved (brainstorming)  
**Scope:** Refactor run controls out of `pipeline-card.svelte`; disable run when execution runner is offline.

## Summary

Extract the pipeline card “Run now” control into `Pipeline.Runner.RunNowButton`. When the pipeline’s execution runner (user-selected for global pipelines, YAML-fixed for specific pipelines) is known to be offline, disable the run button, show a hover tooltip, and prefix the runner subtitle with `[Offline]`.

## Decisions (from brainstorming)

| Topic | Decision |
|-------|----------|
| Approach | **B** — `lib/pipeline/runner/` module + `binding.ts` helper |
| Offline scope | Global **and** specific pipelines |
| Status `undefined` (checking) | Run stays **enabled**; disable only when `isOnline === false` |
| No runner selected (global) | Unchanged — Run enabled, click opens selection modal |
| Subtitle when offline | Prefix runner label with `[Offline]` |

## Architecture

### New files

- `webapp/src/lib/pipeline/runner/run-now-button.svelte` — UI, run flow, modal, offline UX
- `webapp/src/lib/pipeline/runner/binding.test.ts` — unit tests for `getExecutionRunnerPath`

### Modified files

- `webapp/src/lib/pipeline/runner/binding.ts` — add `getExecutionRunnerPath`
- `webapp/src/lib/pipeline/runner/index.ts` — export `RunNowButton`
- `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte` — replace inline run block with `<Pipeline.Runner.RunNowButton />`
- `webapp/messages/en.json` — add `Runner_offline_run_disabled`

### Component boundary (`RunNowButton`)

**Props:**

- `pipeline: PipelinesResponse`
- `onRun?: () => void` — called after successful run (same as today)

**Owns:**

- `handleRunNow` branching (`not-needed` / `specific` / `global`)
- `ButtonGroup` (Run + Configure cog when runner required)
- `Pipeline.Runner.SelectModal` and `runPipelineAfterRunnerSelect` state
- Offline disable, tooltip, `[Offline]` subtitle

**Does not own:** scoreboard, schedules, workflow table (remain in `pipeline-card.svelte`).

## Runner path resolution

Add to `binding.ts`:

```ts
export function getExecutionRunnerPath(pipeline: PipelinesResponse): string | undefined
```

| `getType(pipeline)` | Return value |
|---------------------|--------------|
| `not-needed` | `undefined` |
| `global` | `get(pipeline.id)` — may be `undefined` |
| `specific` | `runner_id` from the first `mobile-automation` step (canonified path, same format as `getPath(runner)`) |

Assumption: specific pipelines use a consistent runner across mobile-automation steps (enforced today by `getType` throwing on mixed global/specific; multiple distinct `runner_id` values are out of scope).

## Offline logic

```ts
const executionPath = getExecutionRunnerPath(pipeline);
const isRunnerOffline =
  executionPath !== undefined && Runners.status.isOnline(executionPath) === false;
```

- **Disable** primary Run button when `isRunnerOffline`.
- **Guard** `handleRunNow` — return early if offline.
- **No disable** when `executionPath` is `undefined` or status is `undefined` (still probing).

### Status probing

- Pipelines layout already polls org runners via `Pipeline.Runners.status.startPolling`.
- When `executionPath` is set and `isOnline(executionPath) === undefined`, trigger a targeted probe:
  - Prefer runner from `Runners.store.read()` by path match.
  - Else resolve via `getRecordByCanonifiedPath` (same as `runner-select-modal.svelte`).
  - Call `Runners.status.probe([runner], { reason: 'visible' })`.

## UI & i18n

### Run button

- `disabled={isRunnerOffline}` on primary Run `Button` only.
- Configure cog: unchanged (`disabled` when `specific`, existing tooltips).

### Tooltip (offline only)

- Wrap `ButtonGroup` in `@/components/ui-custom/tooltip.svelte` when `isRunnerOffline`.
- Use an `inline-flex` wrapper around the group so hover works on disabled controls.
- Message key: `Runner_offline_run_disabled`

```json
"Runner_offline_run_disabled": "The selected runner is currently offline. To run this pipeline, please select a different runner."
```

No tooltip while checking or when enabled.

### Subtitle under “Run now”

When runner is required and a path exists, show truncated path segment (last path segment), as today.

When `isRunnerOffline`, prefix: `[Offline] {name}` (literal bracket prefix, not i18n — status marker).

## `pipeline-card.svelte` after refactor

Remove:

- Run-related state (`runnerSelectionDialogOpen`, `runPipelineAfterRunnerSelect`, `handleRunNow`)
- `ButtonGroup` run UI and `SelectModal` at bottom

Keep in `actions` snippet:

```svelte
<Pipeline.Runner.RunNowButton {pipeline} {onRun} />
```

## Testing

### Unit (`binding.test.ts`)

Table-driven tests for `getExecutionRunnerPath`:

- Pipeline with no mobile-automation steps
- Global pipeline with no stored runner
- Global pipeline with stored runner in localStorage mock
- Specific pipeline with `runner_id` in YAML

No network; mock YAML strings and `lsSync` as needed.

### Manual UAT

On `/my/pipelines`:

1. Global pipeline, select offline runner → Run disabled, tooltip, `[Offline]` subtitle.
2. Global pipeline, no runner → Run enabled, opens modal.
3. Specific pipeline with offline fixed runner → same disable UX.
4. Runner comes back online (poll) → Run re-enables, subtitle loses `[Offline]`.

### Commands

```bash
cd webapp && bun run test:unit -- binding
cd webapp && bun run check
cd webapp && bun run lint
```

## Out of scope

- E2E tests for this change
- Disabling run while status is unknown (`undefined`)
- Requiring runner selection before first run (global)
- Pipelines with multiple distinct execution runners per step
