# Scoreboard Feature Architecture

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend (Svelte/TS)                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────┐      ┌──────────────────────┐            │
│  │ /my/scoreboard       │      │ /scoreboard          │            │
│  │ (Authenticated)      │      │ (Public)             │            │
│  │                      │      │                      │            │
│  │ - 4 Tabs             │      │ - 4 Tabs             │            │
│  │ - Success Rates      │      │ - Success Rates      │            │
│  │ - OTel Data          │      │ - OTel Data          │            │
│  └──────────────────────┘      └──────────────────────┘            │
│           │                             │                           │
│           │                             │                           │
│  ┌────────▼───────────────────────┐    │                           │
│  │ /my/scoreboard/[type]/[id]     │    │                           │
│  │                                │    │                           │
│  │ - Summary Cards                │    │                           │
│  │ - Test History Chart           │    │                           │
│  │ - OTel Span Table              │    │                           │
│  └────────────────────────────────┘    │                           │
│                                         │                           │
└─────────────────────────────────────────┼───────────────────────────┘
                                          │
                    ┌─────────────────────┴───────────────────┐
                    │                                         │
                    ▼                                         ▼
            ┌───────────────┐                      ┌──────────────────┐
            │ GET           │                      │ GET              │
            │ /api/my/      │                      │ /api/all-results │
            │ results       │                      │                  │
            └───────────────┘                      └──────────────────┘
                    │                                         │
┌───────────────────┴─────────────────────────────────────────┴────────┐
│                       Backend API (Go)                               │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │ ScoreboardHandler                                          │    │
│  │                                                            │    │
│  │  - HandleMyResults()      - HandleAllResults()            │    │
│  │  - buildScoreboardResponse()                              │    │
│  │  - buildOTelData()                                        │    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │ Aggregation Functions (Placeholder)                        │    │
│  │                                                            │    │
│  │  - aggregateWalletResults()                               │    │
│  │  - aggregateIssuerResults()                               │    │
│  │  - aggregateVerifierResults()                             │    │
│  │  - aggregatePipelineResults()                             │    │
│  └────────────────────────────────────────────────────────────┘    │
│                            │                                        │
└────────────────────────────┼────────────────────────────────────────┘
                             │
                             ▼
                ┌────────────────────────┐
                │ PocketBase Collections │
                ├────────────────────────┤
                │ - wallets              │
                │ - wallet_actions       │
                │ - credential_issuers   │
                │ - verifiers            │
                │ - use_cases_verif...   │
                │ - pipelines            │
                │ - pipeline_results     │
                └────────────────────────┘
```

## Data Flow

### User-Specific Results Flow

1. User navigates to `/my/scoreboard`
2. Frontend calls `GET /api/my/results` with auth token
3. Backend validates authentication
4. Backend gets user's organization ID
5. Backend aggregates data for user's entities
6. Backend formats response with:
   - Summary tables (wallets, issuers, verifiers, pipelines)
   - OpenTelemetry traces/spans
7. Frontend displays tabbed interface with results

### Public Results Flow

1. User navigates to `/scoreboard` (public)
2. Frontend calls `GET /api/all-results` (no auth required)
3. Backend aggregates data for all entities
4. Backend formats response in same structure
5. Frontend displays public scoreboard

### Detail Page Flow

1. User clicks "View Details" on an entry
2. Frontend navigates to `/my/scoreboard/{type}/{id}`
3. Frontend calls `GET /api/my/results`
4. Frontend filters results to specific entity
5. Frontend displays:
   - Summary metrics
   - Test run history
   - OpenTelemetry span details

## OpenTelemetry Data Structure

```
OTelTracesData
└── ResourceSpans[]
    ├── Resource
    │   └── Attributes[]
    │       ├── service.name: "credimi"
    │       └── service.version: "1.0.0"
    └── ScopeSpans[]
        ├── Scope
        │   ├── name: "credimi.scoreboard"
        │   └── version: "1.0.0"
        └── Spans[]
            ├── traceId
            ├── spanId
            ├── name
            ├── kind: "SPAN_KIND_INTERNAL"
            ├── startTimeUnixNano
            ├── endTimeUnixNano
            ├── attributes[]
            │   ├── entity.id
            │   ├── entity.name
            │   ├── entity.type
            │   ├── test.total_runs
            │   ├── test.success_count
            │   ├── test.failure_count
            │   ├── test.success_rate
            │   └── test.last_run
            └── status
                ├── code: "OK" | "ERROR"
                └── message
```

## Integration Points

### Current State
- ✅ API routes registered
- ✅ Frontend pages created
- ✅ Type definitions complete
- ✅ OpenTelemetry format implemented
- ⚠️ Using placeholder data

### Next Steps
1. Implement real database queries in aggregation functions
2. Add filtering/sorting to API endpoints
3. Implement chart visualization in detail pages
4. Add export functionality
5. Integrate with external OTel collectors (optional)

## File Structure

```
credimi/
├── pkg/internal/apis/
│   ├── RoutesRegistry.go (modified - added routes)
│   └── handlers/
│       ├── scoreboard_handler.go (new - 356 lines)
│       └── scoreboard_handler_test.go (new - 143 lines)
├── webapp/src/routes/
│   ├── (public)/scoreboard/
│   │   └── +page.svelte (new - public view)
│   └── my/scoreboard/
│       ├── +page.svelte (new - 183 lines)
│       ├── types.ts (new - type definitions)
│       └── [type]/[id]/
│           └── +page.svelte (new - detail view)
└── docs/
    ├── SCOREBOARD.md (new - documentation)
    └── ARCHITECTURE.md (this file)
```
