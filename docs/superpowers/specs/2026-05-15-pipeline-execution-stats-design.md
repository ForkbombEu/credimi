# Pipeline execution stats (card + detail page)

**Date:** 2026-05-15  
**Status:** Approved  
**Surfaces:** `pipeline-card.svelte`, `[...pipeline_path]/+page.svelte`

## Goal

Show lifetime pipeline execution statistics from `pipeline_scoreboard_cache` (same data as the scoreboard “Success Rate” column) on:

1. Pipeline cards in the list view
2. The pipeline detail / workflow runs page

Match the visual language of `total-executions-successes-percentage.svelte` without duplicating markup in three places.

## Data source

| Field | PB cache field |
|-------|----------------|
| Total runs | `total_runs` |
| Successes | `total_successes` |
| Success rate (%) | `success_rate` |
| Manual runs | `manually_executed_runs` |
| Scheduled runs | `scheduled_runs` |
| CI runs | `CI_runs` |

**Card:** Reuse existing client fetch — `Scoreboard.Records.loadForPipeline(pipeline.id)` → `scoreboardResults`.

**Detail page:** Load in `+page.ts` via the same API (same pattern as public hub `pipeline-page.svelte`). Return `scoreboard` on page data. Remove stats derived from the paginated workflow list (those reflected only the current page and were incorrect for lifetime totals).

## Shared component

**Path:** `webapp/src/lib/scoreboard/extras/pipeline-execution-stats.svelte`

**Helper:** `fromScoreboardRow(row: ScoreboardRow | undefined): ExecutionStats | undefined`

```ts
type ExecutionStats = {
  total: number;
  successes: number;
  percent: number;
  manual: number;
  scheduled: number;
  ci: number;
};
```

**Layouts:**

| Layout | Use |
|--------|-----|
| `inline` | Card header (beside “Recent workflows”); scoreboard table column |
| `stat-box-success` | Detail page box 1 — large highlight + label |
| `stat-box-modes` | Detail page box 2 — execution mode counts with icons/tooltips |

**Behavior (all layouts):**

- Primary line: `{successes}/{total} ({percent}%)`
- `text-emerald-600` on primary line when `percent >= 70` (match column)
- Secondary line (inline / modes box): manual / scheduled / CI with `HandIcon`, `ClockIcon`, `CogIcon` and existing tooltips (`m.Executed_manually()`, `m.Executed_via_scheduling()`, `m.Executed_via_ci()`)

**Refactor:** `total-executions-successes-percentage.svelte` column cell delegates to this component with `layout="inline"`; column `fn` unchanged for sorting.

## Pipeline card

### Visibility (rule B)

Show execution stats when `scoreboardResults` is defined (cache row exists for pipeline).

- **Do not** gate stats on `workflows.length > 0`.
- **Do not** show execution stats in `content` when cache is missing (keep `afterDescription` summary/placeholder as today).

### Placement

1. **Has recent workflows** (`workflows?.length > 0`): Header row layout:

   ```
   Recent workflows     [stats inline]     View all →
   ```

   Stats between title and “View all”, right-aligned, `layout="inline"`.

2. **No recent workflows, cache exists:** Render stats block in the area where the table would be (no “Recent workflows” title).

### Card body visibility

Today `content` only renders when `showContent` (`workflows.length > 0`). Extend so the card `content` snippet also renders when `scoreboardResults` exists (stats-only cards get a body).

### Unchanged

- `afterDescription`: `PipelineContentSummary` / “summary after first successful run” placeholder — entity summary, not execution stats.

## Pipeline detail page

### Stat boxes (two, not three)

Replace three paginated-workflow boxes with two scoreboard-backed boxes:

| Box | Layout | Label (i18n) |
|-----|--------|----------------|
| 1 | `stat-box-success` | `m.scoreboard_success_rate()` |
| 2 | `stat-box-modes` | `m.Execution_mode()` |

Drop the third box (`m.Total_runs()` / `m.Successful_runs()` as separate boxes).

### Empty / missing cache

When `scoreboard` is undefined: show zeros (`0/0 (0%)`, `0/0/0` mode counts) in both boxes. No extra placeholder copy unless we add it later.

### Loader change

`+page.ts`:

```ts
const scoreboard = await Scoreboard.Records.loadForPipeline(pipeline.id, { fetch });
return { pipeline, workflows, pagination, scoreboard };
```

## Files to touch

| File | Change |
|------|--------|
| `webapp/src/lib/scoreboard/extras/pipeline-execution-stats.svelte` | **New** — shared UI + `fromScoreboardRow` |
| `webapp/src/lib/scoreboard/extras/from-scoreboard-row.ts` (optional) | Helper if kept out of `.svelte` |
| `webapp/src/lib/scoreboard/columns/total-executions-successes-percentage.svelte` | Delegate to shared component |
| `webapp/src/routes/my/pipelines/_partials/pipeline-card.svelte` | Stats placement + `content` visibility |
| `webapp/src/routes/my/pipelines/[...pipeline_path]/+page.ts` | Load scoreboard |
| `webapp/src/routes/my/pipelines/[...pipeline_path]/+page.svelte` | Two stat boxes; remove workflow-derived stats |

No new i18n keys required.

## Testing

- **Card:** cache + workflows / cache only / no cache; stats align with scoreboard column for same pipeline
- **Detail page:** totals stable when changing status filter or pagination
- **Scoreboard column:** visual parity after refactor
- **Unit (optional):** `fromScoreboardRow` with partial/undefined row

## Out of scope

- Changing `PipelineContentSummary` content
- Public hub `pipeline-page.svelte` (already uses scoreboard; could adopt shared component in a follow-up)
- Backend / cache computation changes
