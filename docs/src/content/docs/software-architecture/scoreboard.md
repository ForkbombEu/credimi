---
title: "Scoreboard Feature"
description: "Scoreboard feature API endpoints, data structures, and integration"
---

# Scoreboard Feature

## Overview

The Scoreboard shows aggregated pipeline execution results across the platform. The public UI reads pre-aggregated rows from the PocketBase `pipeline_scoreboard_cache` collection. A Temporal workflow periodically refreshes that cache from per-namespace pipeline statistics.

## API Endpoints

### `GET /api/pipeline/scoreboard/{namespace}`

Returns aggregated pipeline statistics for a single Temporal namespace (organization).

**Authentication:** Internal / trusted callers (used by the aggregation workflow)

### `POST /api/pipeline/scoreboard/save-results`

Accepts merged aggregation output and refreshes `pipeline_scoreboard_cache`.

**Authentication:** Internal admin API key

### `POST /api/pipeline/scoreboard/aggregate/start`

Starts the `AggregateScoreboardWorkflow`. Optional `?schedule=<seconds>` creates a recurring schedule.

### `DELETE /api/pipeline/scoreboard/aggregate/schedule/{schedule_id}`

Cancels a scheduled aggregation workflow.

### Legacy (removed from active routes)

The older `/api/my/results` and `/api/all-results` endpoints described an OpenTelemetry tabbed scoreboard. That implementation is commented out in `scoreboard_handler.go` and is no longer used by the frontend.

## Data Model

### `pipeline_scoreboard_cache`

Primary read model for the scoreboard UI. One row per pipeline, with fields such as:

- `total_runs`, `total_successes`, `success_rate`
- `pipeline` relation (used for published filtering and sorts)
- `expanded_data` JSON snapshot for display: wallets, issuers, verifiers, credentials, devices, custom integrations, and latest execution artifacts
- Relation ID fields remain for PocketBase filter/sort; they are not expanded in the public UI

### Frontend type

```typescript
// webapp/src/lib/scoreboard/types.ts
export type ScoreboardRow = PipelineScoreboardCacheResponse<...>;
```

## Frontend

### Public Scoreboard (`/scoreboard`)

- Loads rows via `Scoreboard.loadData()` (`$lib/scoreboard/functions.ts`)
- Renders `Scoreboard.Component` with a `Scoreboard.Instance` table controller
- Columns include pipeline name, evidence, wallets, issuance (issuers with nested credentials), presentations (verifier logos), conformance checks, devices, and last run
- Supports server-side pagination and sorting on relation/scalar fields
- Row details also show custom integrations and execution modes

### Homepage Section

`webapp/src/routes/(public)/_partials/scoreboard-section.svelte` shows a small random sample of scoreboard rows on the public landing page.

### Library exports

```typescript
import { Scoreboard } from '$lib';

Scoreboard.loadData();
Scoreboard.Component;
Scoreboard.Instance;
Scoreboard.EntityDisplay;
```

## Aggregation Workflow

`AggregateScoreboardWorkflow` (`pkg/workflowengine/workflows/scoreboard.go`):

1. Enumerate organization namespaces
2. Fetch per-namespace stats from `/api/pipeline/scoreboard/{namespace}`
3. Merge results across tenants
4. Persist via `/api/pipeline/scoreboard/save-results`

Handlers live in `pkg/internal/apis/handlers/scoreboard.go`.

## Usage Examples

### Load scoreboard data in a SvelteKit route

```typescript
import { Scoreboard } from '$lib';

export const load = async ({ fetch }) => {
	const data = await Scoreboard.loadData({ fetch });
	return { scoreboardData: data.items };
};
```

### Start aggregation manually

```bash
curl -X POST "https://your-domain/api/pipeline/scoreboard/aggregate/start" \
  -H "X-Api-Key: $CREDIMI_INTERNAL_ADMIN_KEY"
```

## Future Enhancements

- Filtering and column presets on the public table
- Historical trending from `pipeline_results`
- Per-organization public views
- Scheduled aggregation monitoring in the admin UI
