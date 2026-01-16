# Scoreboard Feature

## Overview

The Scoreboard feature provides a comprehensive view of test results for Wallets, Issuers, Verifiers, and Pipelines in the Credimi platform. Results are available in OpenTelemetry-compatible format for standardized telemetry integration.

## API Endpoints

### 1. `/api/my/results` (Authenticated)

Returns scoreboard results for the current authenticated user's Wallet/Issuer/Verifier/Pipelines.

**Method:** GET  
**Authentication:** Required  
**Response Format:**

```json
{
  "summary": {
    "wallets": [...],
    "issuers": [...],
    "verifiers": [...],
    "pipelines": [...]
  },
  "otelData": {
    "resourceSpans": [...]
  }
}
```

### 2. `/api/all-results` (Public)

Returns scoreboard results for all Wallet/Issuer/Verifier/Pipelines across the system.

**Method:** GET  
**Authentication:** Not required  
**Response Format:** Same as `/api/my/results`

## Data Structures

### ScoreboardEntry

Each entity (wallet, issuer, verifier, pipeline) is represented with:

```typescript
{
  id: string;
  name: string;
  type: 'wallet' | 'issuer' | 'verifier' | 'pipeline';
  totalRuns: number;
  successCount: number;
  failureCount: number;
  successRate: number;  // Percentage (0-100)
  lastRun: string;       // ISO 8601 timestamp
}
```

### OpenTelemetry Format

Results are also provided in OpenTelemetry format with:
- Resource spans containing service metadata
- Scope spans with trace/span information
- Attributes including test metrics (success rate, counts, etc.)
- Status information (OK/ERROR)

## Frontend Pages

### 1. User Scoreboard (`/my/scoreboard`)

Authenticated users can view their own scoreboard with:
- Tabbed interface for Wallets, Issuers, Verifiers, and Pipelines
- Summary table showing success rates and test statistics
- Links to detailed pages for each entity
- Expandable OpenTelemetry data section

### 2. Public Scoreboard (`/scoreboard`)

Public view showing aggregate results for all entities across the system.

### 3. Detail Pages (`/my/scoreboard/[type]/[id]`)

Individual entity pages showing:
- Summary statistics cards
- Test run history visualization (placeholder)
- Recent test runs table with OpenTelemetry span data
- Raw OpenTelemetry data in expandable section

## Implementation Notes

### Current Status

The implementation includes:
- ✅ Full API structure with OpenTelemetry format
- ✅ Frontend components with responsive design
- ✅ Type definitions for TypeScript
- ⚠️ **Placeholder data aggregation** - Currently returns example data

### TODO: Real Data Integration

The following functions need to be implemented with actual database queries:

1. `aggregateWalletResults()` - Query `wallets` and `wallet_actions` collections
2. `aggregateIssuerResults()` - Query `credential_issuers` collection
3. `aggregateVerifierResults()` - Query `verifiers` and `use_cases_verifications` collections
4. `aggregatePipelineResults()` - Query `pipelines` and `pipeline_results` collections

Each function should:
- Filter by `ownerID` when `userSpecific` is true
- Aggregate test run data from related collections
- Calculate success/failure counts and rates
- Return properly formatted `ScoreboardEntry` objects

### Database Collections

The following PocketBase collections are relevant:
- `wallets` - Wallet definitions
- `wallet_actions` - Wallet action test results
- `credential_issuers` - Credential issuer definitions
- `verifiers` - Verifier definitions
- `use_cases_verifications` - Use case verification results
- `pipelines` - Pipeline definitions
- `pipeline_results` - Pipeline execution results (with `owner`, `workflow_id`, `run_id`)

### OpenTelemetry Integration

The current implementation provides OpenTelemetry-compatible data structures:
- **Traces** - Each entity represents a logical trace
- **Spans** - Test runs are represented as spans
- **Attributes** - Test metrics are stored as span attributes
- **Status** - Success/failure indicated via status codes

This format allows integration with standard OpenTelemetry collectors and visualization tools.

## Usage Examples

### Fetching User Results

```javascript
const response = await fetch('/api/my/results', {
  headers: {
    'Authorization': 'Bearer <token>'
  }
});
const data = await response.json();
console.log(data.summary.wallets);
```

### Fetching Public Results

```javascript
const response = await fetch('/api/all-results');
const data = await response.json();
console.log(data.summary.pipelines);
```

## Future Enhancements

Potential improvements:
- Real-time updates via WebSocket
- Filtering and sorting options
- Date range selection for historical data
- Export functionality (CSV, JSON, OpenTelemetry format)
- Integration with external OpenTelemetry collectors
- Custom visualization charts (line graphs, pie charts)
- Comparison views between entities
- Detailed error logs and debugging information
