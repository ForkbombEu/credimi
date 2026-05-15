---
title: "Scoreboard Feature - Implementation Summary"
description: "Summary of the Scoreboard feature implementation"
---

## Overview

The Scoreboard feature provides a public dashboard of pipeline execution results. Data is aggregated in the backend, stored in `pipeline_scoreboard_cache`, and rendered in the webapp as a paginated table with rich entity display.

## Features Delivered

### Backend

- **`GET /api/pipeline/scoreboard/{namespace}`** — per-tenant pipeline stats
- **`POST /api/pipeline/scoreboard/save-results`** — refresh the PocketBase cache
- **`POST /api/pipeline/scoreboard/aggregate/start`** — start or schedule aggregation
- **`DELETE /api/pipeline/scoreboard/aggregate/schedule/{schedule_id}`** — cancel a schedule
- **`AggregateScoreboardWorkflow`** — Temporal workflow that merges namespace stats
- **Unit/integration tests** in `pkg/internal/apis/handlers/scoreboard_test.go`

### Frontend

#### Public Scoreboard (`/scoreboard`)

- Paginated TanStack table backed by `pipeline_scoreboard_cache`
- Sortable columns and entity display components
- Links to marketplace pipeline pages

#### Homepage Section

- Random sample of scoreboard rows on the public landing page
- Pipeline success summary and related entities

#### Shared module (`webapp/src/lib/scoreboard`)

- `loadData()` PocketBase query helper
- `ScoreboardTable` controller class
- Column definitions and `EntityDisplay` helpers

### Documentation

- `SCOREBOARD.md` — API, data model, and frontend usage
- `ARCHITECTURE.md` — system diagram and data flows
- `SUMMARY.md` — this file

## Removed

- `/my/scoreboard` authenticated route
- `/my/scoreboard/[type]/[id]` detail route
- Legacy tabbed scoreboard components (`ScoreboardTableTabbed`, `OTelDetails`)
- Frontend dependency on `/api/my/results` and `/api/all-results`

## File Inventory

### Backend

```
pkg/internal/apis/handlers/
├── scoreboard.go
├── scoreboard_test.go
└── scoreboard_handler.go          (legacy OTel code, commented out)

pkg/workflowengine/workflows/
├── scoreboard.go
└── scoreboard_test.go
```

### Frontend

```
webapp/src/lib/scoreboard/
├── index.ts
├── functions.ts
├── types.ts
├── table.svelte
├── table.svelte.ts
├── columns/
├── entity-display/
└── extras/pipeline-content-summary.svelte

webapp/src/routes/
├── (public)/scoreboard/
│   ├── +page.ts
│   └── +page.svelte
└── (public)/_partials/scoreboard-section.svelte
```

## Integration Guide

### Quick Start

1. Start the backend and webapp (`make dev`)
2. Open the public scoreboard: `https://your-domain/scoreboard`
3. Refresh cache (internal): `POST /api/pipeline/scoreboard/aggregate/start`

### Testing

```bash
# Backend
go test -tags=unit ./pkg/internal/apis/handlers/ -run Scoreboard

# Frontend
cd webapp && bun run check
```

## Conclusion

The scoreboard is a PocketBase-backed public read model with Temporal-driven aggregation. The legacy authenticated `/my/scoreboard` experience and OpenTelemetry tabbed UI have been removed in favor of the unified `$lib/scoreboard` module used by `/scoreboard` and the homepage section.
