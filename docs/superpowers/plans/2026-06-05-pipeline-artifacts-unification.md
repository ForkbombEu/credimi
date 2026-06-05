# Pipeline Artifacts Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify pipeline execution artifacts (video, screenshot, log, report) across `list-executions` API and scoreboard expand, with one shared Svelte preview component supporting `preview` and `compact` variants.

**Architecture:** Extend Go helpers (`buildPipelineExecutionArtifacts`) as single source of truth for grouping. Wire into `list-executions` handlers and `OnRecordEnrich("pipeline_results")`. Frontend normalizes via `execution-artifacts.ts` adapters into `ExecutionArtifactsPreview.svelte`.

**Tech Stack:** Go (PocketBase hooks, handlers), Svelte 5, Vitest, existing MediaPreview / IconButton / Sheet / RenderMD.

**Spec:** `docs/superpowers/specs/2026-06-05-pipeline-artifacts-unification-design.md`

**Pre-flight:** Run `gitnexus_impact({ target: "computePipelineResultsFromRecord", direction: "upstream" })` and note blast radius before Go edits.

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/shared.go` | Add `Report` to `WorkflowExecutionSummary`; add `PipelineExecutionArtifacts` struct |
| `pkg/internal/apis/handlers/pipeline_results_handler.go` | `computePipelineReportURL`, `BuildPipelineExecutionArtifacts`, `attachPipelineArtifactsToSummary` |
| `pkg/internal/apis/handlers/pipeline_handler.go` | Batch-fetch + attach artifacts on grouped `list-executions` |
| `pkg/internal/pipeline_results/pipeline_results.go` | `OnRecordEnrich` hook |
| `pkg/internal/pipeline_results/pipeline_results_test.go` | Enrich hook tests |
| `pkg/routes/routes.go` | Register `RegisterPipelineResultsHooks` |
| `webapp/src/lib/pipeline/execution-artifacts.ts` | Canonical type + adapters |
| `webapp/src/lib/pipeline/execution-artifacts.test.ts` | Adapter unit tests |
| `webapp/src/lib/pipeline/results/execution-artifacts-preview.svelte` | Shared dual-variant UI |
| `webapp/src/lib/scoreboard/columns/video-screenshot.svelte` | Delegate to shared component |
| `webapp/src/lib/pipeline/workflows-table.svelte` | Delegate to shared component |
| `webapp/src/lib/pipeline/workflows-table-small.svelte` | Delegate to shared component |
| `webapp/src/lib/pipeline/workflows.ts` | Add `report?`; remove mock early-return |

---

### Task 1: Go artifact types and report URL helper

**Files:**
- Modify: `pkg/internal/apis/handlers/shared.go`
- Modify: `pkg/internal/apis/handlers/pipeline_results_handler.go`
- Test: `pkg/internal/apis/handlers/workflow_helpers_test.go`

- [ ] **Step 1: Write the failing test**

Add to `workflow_helpers_test.go`:

```go
func TestComputePipelineReportURL(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://app.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Id = "rec123"
	record.Set("report", "run_report.md")

	got := computePipelineReportURL(app, record)
	require.Equal(t, "https://app.test/api/files/pipeline_results/rec123/report/run_report.md", got)

	record.Set("report", "")
	require.Equal(t, "", computePipelineReportURL(app, record))
	require.Equal(t, "", computePipelineReportURL(nil, record))
	require.Equal(t, "", computePipelineReportURL(app, nil))
}

func TestBuildPipelineExecutionArtifacts(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	app.Settings().Meta.AppURL = "https://app.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Id = "rec123"
	record.Set("video_results", []string{"abc_result_video_main.mp4"})
	record.Set("screenshots", []string{"abc_screenshot_main.png"})
	record.Set("logcats", []string{"abc_logfile_main.zip"})
	record.Set("report", "run_report.md")

	got := BuildPipelineExecutionArtifacts(app, record)
	require.Len(t, got.Results, 1)
	require.Contains(t, got.Results[0].Log, "abc_logfile_main.zip")
	require.Equal(t, "https://app.test/api/files/pipeline_results/rec123/report/run_report.md", got.Report)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/... -run 'TestComputePipelineReportURL|TestBuildPipelineExecutionArtifacts' -v`

Expected: FAIL — `computePipelineReportURL` / `BuildPipelineExecutionArtifacts` not defined

- [ ] **Step 3: Implement types and helpers**

In `shared.go`, after `PipelineResults`:

```go
type PipelineExecutionArtifacts struct {
	Results []PipelineResults `json:"results"`
	Report  string            `json:"report,omitempty"`
}
```

Add `Report string \`json:"report,omitempty"\`` to `WorkflowExecutionSummary`.

In `pipeline_results_handler.go`:

```go
func computePipelineReportURL(app core.App, record *core.Record) string {
	if app == nil || record == nil {
		return ""
	}
	reportName := record.GetString("report")
	if reportName == "" {
		return ""
	}
	return utils.JoinURL(
		app.Settings().Meta.AppURL,
		"api", "files", "pipeline_results",
		record.Id,
		record.GetString("report"),
		reportName,
	)
}

func BuildPipelineExecutionArtifacts(app core.App, record *core.Record) PipelineExecutionArtifacts {
	results := computePipelineResultsFromRecord(app, record)
	if results == nil {
		results = []PipelineResults{}
	}
	return PipelineExecutionArtifacts{
		Results: results,
		Report:  computePipelineReportURL(app, record),
	}
}

func attachPipelineArtifactsToSummary(
	summary *WorkflowExecutionSummary,
	app core.App,
	record *core.Record,
) {
	if summary == nil {
		return
	}
	artifacts := BuildPipelineExecutionArtifacts(app, record)
	summary.Results = artifacts.Results
	summary.Report = artifacts.Report
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/... -run 'TestComputePipelineReportURL|TestBuildPipelineExecutionArtifacts' -v`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/shared.go pkg/internal/apis/handlers/pipeline_results_handler.go pkg/internal/apis/handlers/workflow_helpers_test.go
git commit -m "feat: add pipeline execution artifacts builder and report URL helper"
```

---

### Task 2: Wire artifacts into per-pipeline list-executions

**Files:**
- Modify: `pkg/internal/apis/handlers/pipeline_results_handler.go` (`buildPipelineExecutionHierarchyFromResult`)
- Test: `pkg/internal/apis/handlers/pipeline_results_handler_test.go`

- [ ] **Step 1: Write the failing test**

Extend an existing hierarchy test (or add) asserting `Report` is set when record has `report` field. Pattern from `TestComputePipelineResultsFromRecord` — create record with video/screenshot/log/report, call `buildPipelineExecutionHierarchyFromResult`, assert `rootSummary.Report` non-empty.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/... -run 'buildPipelineExecutionHierarchyFromResult|PipelineExecutionHierarchy' -v`

Expected: FAIL — `Report` empty

- [ ] **Step 3: Update hierarchy builder**

In `buildPipelineExecutionHierarchyFromResult`, replace the block that only sets `rootSummary.Results`:

```go
if rootSummary.Type.Name == w.Name() {
	if resultRecord != nil {
		attachPipelineArtifactsToSummary(rootSummary, app, resultRecord)
	}
	if len(rootSummary.Results) == 0 {
		rootSummary.Results = computePipelineResults(
			app,
			namespace,
			rootExecution.Execution.WorkflowID,
			rootExecution.Execution.RunID,
		)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/pipeline_results_handler.go pkg/internal/apis/handlers/pipeline_results_handler_test.go
git commit -m "feat: attach report URL to per-pipeline execution summaries"
```

---

### Task 3: Fix grouped list-executions artifacts

**Files:**
- Modify: `pkg/internal/apis/handlers/pipeline_handler.go`
- Modify: `pkg/internal/apis/handlers/pipeline_results_handler.go` (add `attachPipelineArtifactsToSummaries` helper)
- Test: `pkg/internal/apis/handlers/pipeline_handler_test.go`

- [ ] **Step 1: Write the failing test**

Extend `TestHandleGetPipelineDetailsReturnsResults` — set file fields on `resultRecord`:

```go
resultRecord.Set("video_results", []string{"abc_result_video_main.mp4"})
resultRecord.Set("screenshots", []string{"abc_screenshot_main.png"})
resultRecord.Set("logcats", []string{"abc_logfile_main.zip"})
resultRecord.Set("report", "run_report.md")
```

After unmarshaling response, assert:

```go
summary := response[pipelineRecord.Id][0]
require.Len(t, summary.Results, 1)
require.NotEmpty(t, summary.Results[0].Video)
require.NotEmpty(t, summary.Results[0].Log)
require.Contains(t, summary.Report, "run_report.md")
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/... -run TestHandleGetPipelineDetailsReturnsResults -v`

Expected: FAIL — `Results` empty or `Report` empty

- [ ] **Step 3: Add batch attach helper**

In `pipeline_results_handler.go`:

```go
func attachPipelineArtifactsToSummaries(
	app core.App,
	ownerID string,
	summaries []*WorkflowExecutionSummary,
) error {
	if app == nil || len(summaries) == 0 {
		return nil
	}

	refs := make([]workflowExecutionRef, 0, len(summaries))
	for _, summary := range summaries {
		if summary == nil || summary.Execution == nil {
			continue
		}
		refs = append(refs, workflowExecutionRef{
			WorkflowID: summary.Execution.WorkflowID,
			RunID:      summary.Execution.RunID,
		})
	}

	resultRecords, err := fetchPipelineResultRecords(app, ownerID, refs)
	if err != nil {
		return err
	}

	for _, summary := range summaries {
		if summary == nil || summary.Execution == nil {
			continue
		}
		ref := workflowExecutionRef{
			WorkflowID: summary.Execution.WorkflowID,
			RunID:      summary.Execution.RunID,
		}
		if record, ok := resultRecords[ref]; ok {
			attachPipelineArtifactsToSummary(summary, app, record)
		}
	}

	return nil
}
```

- [ ] **Step 4: Wire into HandleGetPipelineDetails**

After `selectTopExecutionsByPipeline`, before building response, flatten all selected summaries and call `attachPipelineArtifactsToSummaries(e.App, orgID, flatSummaries)`. Handle error with 500 response.

- [ ] **Step 5: Run test to verify it passes**

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/internal/apis/handlers/pipeline_handler.go pkg/internal/apis/handlers/pipeline_results_handler.go pkg/internal/apis/handlers/pipeline_handler_test.go
git commit -m "feat: attach artifacts to grouped list-executions via batch lookup"
```

---

### Task 4: OnRecordEnrich hook for pipeline_results

**Files:**
- Create: `pkg/internal/pipeline_results/pipeline_results.go`
- Create: `pkg/internal/pipeline_results/pipeline_results_test.go`
- Modify: `pkg/routes/routes.go`

- [ ] **Step 1: Write the failing test**

```go
// pkg/internal/pipeline_results/pipeline_results_test.go
package pipelineresults

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../../test_pb_data/"

func TestHandlePipelineResultsEnrichSetsArtifacts(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	RegisterPipelineResultsHooks(app)
	app.Settings().Meta.AppURL = "https://app.test"

	coll, err := app.FindCollectionByNameOrId("pipeline_results")
	require.NoError(t, err)

	record := core.NewRecord(coll)
	record.Set("video_results", []string{"abc_result_video_main.mp4"})
	record.Set("screenshots", []string{"abc_screenshot_main.png"})
	record.Set("logcats", []string{"abc_logfile_main.zip"})
	record.Set("report", "run_report.md")
	require.NoError(t, app.Save(record))

	enriched, err := app.ExpandRecord(record, []string{}, nil)
	require.NoError(t, err)

	artifacts, ok := enriched.Get("artifacts").(map[string]any)
	require.True(t, ok, "artifacts field missing")
	results, ok := artifacts["results"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, results)
	require.Contains(t, artifacts["report"], "run_report.md")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/pipeline_results/... -run TestHandlePipelineResultsEnrichSetsArtifacts -v`

Expected: FAIL — package / hook not found

- [ ] **Step 3: Implement hook**

```go
// pkg/internal/pipeline_results/pipeline_results.go
package pipelineresults

import (
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterPipelineResultsHooks(app core.App) {
	app.OnRecordEnrich("pipeline_results").BindFunc(HandlePipelineResultsEnrich)
}

func HandlePipelineResultsEnrich(e *core.RecordEnrichEvent) error {
	artifacts := handlers.BuildPipelineExecutionArtifacts(e.App, e.Record)
	e.Record.WithCustomData(true)
	e.Record.Set("artifacts", artifacts)
	return e.Next()
}
```

In `pkg/routes/routes.go`, add import and call:

```go
pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
// ...
pipelineresults.RegisterPipelineResultsHooks(app)
```

- [ ] **Step 4: Run test to verify it passes**

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/pipeline_results/ pkg/routes/routes.go
git commit -m "feat: enrich pipeline_results with artifacts on record view"
```

---

### Task 5: Frontend adapters

**Files:**
- Create: `webapp/src/lib/pipeline/execution-artifacts.ts`
- Create: `webapp/src/lib/pipeline/execution-artifacts.test.ts`
- Modify: `webapp/src/lib/pipeline/workflows.ts`

- [ ] **Step 1: Write the failing test**

```ts
// webapp/src/lib/pipeline/execution-artifacts.test.ts
import { describe, expect, it } from 'vitest';

import { fromApiSummary, fromEnrichedRecord } from './execution-artifacts';

describe('fromApiSummary', () => {
	it('returns undefined when no results and no report', () => {
		expect(fromApiSummary({})).toBeUndefined();
	});

	it('maps results and report', () => {
		expect(
			fromApiSummary({
				results: [{ video: 'v', screenshot: 's', log: 'l' }],
				report: 'https://app/r.md'
			})
		).toEqual({
			results: [{ video: 'v', screenshot: 's', log: 'l' }],
			report: 'https://app/r.md'
		});
	});
});

describe('fromEnrichedRecord', () => {
	it('returns artifacts when present', () => {
		expect(
			fromEnrichedRecord({
				artifacts: { results: [{ video: 'v', screenshot: 's', log: 'l' }] }
			})
		).toEqual({
			results: [{ video: 'v', screenshot: 's', log: 'l' }]
		});
	});

	it('returns undefined when artifacts missing', () => {
		expect(fromEnrichedRecord({})).toBeUndefined();
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/execution-artifacts.test.ts`

Expected: FAIL — module not found

- [ ] **Step 3: Implement adapters**

```ts
// webapp/src/lib/pipeline/execution-artifacts.ts
export type PipelineExecutionArtifacts = {
	results: Array<{ video: string; screenshot: string; log: string }>;
	report?: string;
};

export function fromApiSummary(summary: {
	results?: PipelineExecutionArtifacts['results'];
	report?: string;
}): PipelineExecutionArtifacts | undefined {
	const hasResults = (summary.results?.length ?? 0) > 0;
	const hasReport = Boolean(summary.report);
	if (!hasResults && !hasReport) return undefined;
	return {
		results: summary.results ?? [],
		report: summary.report
	};
}

export function fromEnrichedRecord(record: {
	artifacts?: PipelineExecutionArtifacts;
}): PipelineExecutionArtifacts | undefined {
	if (!record.artifacts) return undefined;
	const { results, report } = record.artifacts;
	const hasResults = (results?.length ?? 0) > 0;
	const hasReport = Boolean(report);
	if (!hasResults && !hasReport) return undefined;
	return { results: results ?? [], report };
}
```

Add `report?: string` to `ExecutionSummary` in `workflows.ts`.

- [ ] **Step 4: Run test to verify it passes**

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline/execution-artifacts.ts webapp/src/lib/pipeline/execution-artifacts.test.ts webapp/src/lib/pipeline/workflows.ts
git commit -m "feat: add pipeline execution artifacts adapters"
```

---

### Task 6: Shared ExecutionArtifactsPreview component

**Files:**
- Create: `webapp/src/lib/pipeline/results/execution-artifacts-preview.svelte`
- Delete: `webapp/src/lib/pipeline/results/artifacts-buttons.svelte` (WIP stub)

- [ ] **Step 1: Create component**

Port logic from `video-screenshot.svelte` (report Sheet + MediaPreview) and `workflows-table-small.svelte` (IconButtons). Props:

```svelte
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { FileCogIcon, ImageIcon, VideoIcon } from '@lucide/svelte';

	import MediaPreview from '$lib/components/media-preview.svelte';
	import type { PipelineExecutionArtifacts } from '$lib/pipeline/execution-artifacts';

	import RenderMD from '@/components/ui-custom/renderMD.svelte';
	import Sheet from '@/components/ui-custom/sheet.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';

	type Props = {
		artifacts: PipelineExecutionArtifacts;
		variant?: 'preview' | 'compact';
		previewClass?: string;
		emptyState?: Snippet<[]>;
	};

	let { artifacts, variant = 'preview', previewClass, emptyState }: Props = $props();

	const hasContent = $derived(
		artifacts.results.length > 0 || Boolean(artifacts.report)
	);

	const reportPromise = $derived.by(() => {
		if (!artifacts.report) return undefined;
		return fetch(artifacts.report).then((res) => res.text());
	});
</script>
```

Render loop: for each result group, branch on `variant === 'preview'` (MediaPreview trio) vs `compact` (IconButton trio). Append report Sheet when `reportPromise` resolves (same markup as `video-screenshot.svelte`). If `!hasContent` and `emptyState` provided, render `emptyState`.

Run Svelte autofixer / `bun run check` after writing.

- [ ] **Step 2: Delete WIP stub**

Remove `artifacts-buttons.svelte`; grep for imports and update any references.

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline/results/
git commit -m "feat: add shared execution artifacts preview component"
```

---

### Task 7: Wire consumers

**Files:**
- Modify: `webapp/src/lib/scoreboard/columns/video-screenshot.svelte`
- Modify: `webapp/src/lib/pipeline/workflows-table.svelte`
- Modify: `webapp/src/lib/pipeline/workflows-table-small.svelte`
- Modify: `webapp/src/lib/pipeline/workflows.ts`

- [ ] **Step 1: Refactor scoreboard column**

Replace `groupExecutionArtifacts` and inline render with:

```svelte
import ExecutionArtifactsPreview from '$lib/pipeline/results/execution-artifacts-preview.svelte';
import { fromEnrichedRecord } from '$lib/pipeline/execution-artifacts';

export const column = Column.define({
	fn: (row) => fromEnrichedRecord(row.expand.latest_successful_execution ?? {}),
	// ...
});
```

Cell: `<ExecutionArtifactsPreview artifacts={value} variant="preview" previewClass="size-8!" />` or show `<EntityDisplay.Na />` when undefined.

- [ ] **Step 2: Refactor workflow tables**

`workflows-table.svelte`: replace inline MediaPreview loop with:

```svelte
import ExecutionArtifactsPreview from './results/execution-artifacts-preview.svelte';
import { fromApiSummary } from './execution-artifacts';

{@const artifacts = fromApiSummary(workflow)}
{#if artifacts}
	<ExecutionArtifactsPreview {artifacts} variant="preview" />
{:else}
	<span class="text-muted-foreground opacity-50">N/A</span>
{/if}
```

`workflows-table-small.svelte`: same with `variant="compact"` and existing `na()` snippet.

- [ ] **Step 3: Remove mock in workflows.ts**

Delete lines:

```ts
const test = await import('./workflows.mock.json');
return test.default;
```

- [ ] **Step 4: Run checks**

Run: `cd webapp && bun run check`
Run: `cd webapp && bun run test:unit -- src/lib/pipeline/execution-artifacts.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/columns/video-screenshot.svelte webapp/src/lib/pipeline/workflows-table.svelte webapp/src/lib/pipeline/workflows-table-small.svelte webapp/src/lib/pipeline/workflows.ts
git commit -m "feat: wire shared artifacts preview across scoreboard and workflow tables"
```

---

### Task 8: Final verification

- [ ] **Step 1: Run Go unit tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/... ./pkg/internal/pipeline_results/... -count=1`

Expected: PASS

- [ ] **Step 2: Run webapp check**

Run: `cd webapp && bun run check`

Expected: PASS

- [ ] **Step 3: GitNexus detect changes**

Run: `gitnexus_detect_changes()` per AGENTS.md — confirm scope matches expected symbols only.

- [ ] **Step 4: Manual smoke (optional)**

- Scoreboard row shows log icon alongside video/screenshot
- Report sheet opens from both variants
- Grouped `list-executions` returns populated `results` + `report`
