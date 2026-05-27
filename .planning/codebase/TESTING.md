# Testing Patterns

**Analysis Date:** 2026-05-27

## Test Framework

**Go runner:**
- Standard `go test` orchestrated by `scripts/test-summary.sh` (invoked via `make test`).
- Default flags: `-tags=unit -race -short -buildvcs` on `./...`.
- Assertion library: `github.com/stretchr/testify` (`require` for hard failures, `assert` where soft checks help — `go.mod` pins `v1.11.1`).

**Webapp runner:**
- Vitest 4 via Vite config (`webapp/vite.config.ts`).
- Browser provider: `@vitest/browser-playwright` (Chromium headless).
- E2E: Playwright (`@playwright/test` in `webapp/package.json`).

**Run commands:**

```bash
make test                              # Go unit tests (race + short + unit tag)
make test.all                          # Go tests without -short (TEST_SHORT=0)
make test.p TestPipelineQueueEnqueue   # Watch single test name via gow
go test -tags=unit ./pkg/... -run TestName   # Focused package/test
make coverage                          # HTML coverage report (unit tag)
make coverage-check                    # Fail if total coverage < COVERAGE_MIN (80%)

cd webapp && bun run test:unit -- --run  # Vitest once (client + server projects)
cd webapp && bun run test:e2e            # Playwright (needs running stack)
cd webapp && bun run check               # svelte-check typecheck
```

## Test File Organization

**Go:**
- Co-located `*_test.go` beside implementation under `pkg/...` and `cmd/...` (~146 test files).
- Same package as code under test (`package handlers`, `package workflows`) for white-box access.
- Shared test constants: `testDataDir` pointing at repo-root `test_pb_data/` (e.g. `const testDataDir = "../../../../test_pb_data"` in `pkg/internal/apis/handlers/shared.go`). Read-only fixture DB; also mirrored under `fixtures/test_pb_data/` for licensing metadata — prefer `test_pb_data` paths already used in tests.

**Webapp unit:**
- `src/**/*.test.ts` — Node environment (pure TS).
- `src/**/*.svelte.test.ts` / `*.svelte.spec.ts` — browser project only.
- Exclude `src/lib/server/**` from browser project.

**Webapp E2E:**
- `webapp/e2e/nru/` — unauthenticated flows.
- `webapp/e2e/logged/` — uses `storageState` from setup project.
- Setup: `webapp/e2e/**/*.setup.ts` (Playwright `testMatch` in `webapp/playwright.config.ts`).

## Build Tags & Test Tiers

**Unit (default CI / `make test`):**
- Build tag `unit` — most tests compile with `-tags=unit`.
- A small set of files use `//go:build unit` explicitly (e.g. `pkg/workflowengine/activities/github_pr_comment_test.go`).

**Integration / heavy (excluded from default unit run):**
- `//go:build !unit` — run only without `-tags=unit` (e.g. `pkg/workflowengine/activities/cesr_test.go`, `pkg/workflowengine/workflows/zenroom_test.go`, `pkg/internal/apis/templating_test.go`).
- These may invoke external binaries, real subprocesses, or longer I/O; not in default `make test`.

**Short vs long:**
- `scripts/test-summary.sh` passes `-short` unless `TEST_SHORT=0` (`make test.all`).
- Use `testing.Short()` in tests that should skip under `-short` when adding long scenarios.

## Test Structure

**Go — table-driven + subtests:**

```go
func TestDecodeFromTemporalPayload(t *testing.T) {
	t.Parallel()
	require.Equal(t, "hello", DecodeFromTemporalPayload(encoded))
}

func TestFetchWorkflowFailure(t *testing.T) {
	t.Parallel()
	t.Run("no events", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.On("GetWorkflowHistory", mock.Anything, "wf-1", "run-1", false, enums.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT).
			Return(&fakeHistoryIterator{}, nil).Once()
		result := fetchWorkflowFailure(context.Background(), mockClient, "wf-1", "run-1")
		require.Nil(t, result)
		mockClient.AssertExpectations(t)
	})
}
```

Pattern source: `pkg/internal/apis/handlers/shared_test.go`.

**Go — HTTP handler scenarios (PocketBase):**

```go
scenarios := []tests.ApiScenario{
	{
		Name:           "missing pipeline_identifier parameter",
		Method:         http.MethodGet,
		URL:            "/api/pipeline/get-yaml",
		ExpectedStatus: 400,
		ExpectedContent: []string{`"pipeline_identifier is required"`},
		Headers:        map[string]string{"Credimi-Api-Key": "internal-test-api-key"},
		TestAppFactory: setupPipelineApp,
	},
}
```

Pattern source: `pkg/internal/apis/handlers/pipeline_handler_test.go`. Use `tests.NewTestApp(testDataDir)`, register routes/hooks in `TestAppFactory`, `defer app.Cleanup()`.

**Go — Temporal workflows:**

```go
func Test_CredentialsIssuersWorkflow(t *testing.T) {
	testCases := []struct {
		name           string
		input          workflowengine.WorkflowInput
		mockActivities func(env *testsuite.TestWorkflowEnvironment)
		expectedErr    bool
	}{ /* ... */ }
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			env := testsuite.WorkflowTestSuite{}.NewTestWorkflowEnvironment()
			tc.mockActivities(env)
			env.ExecuteWorkflow(CredentialsIssuersWorkflow, tc.input)
			// require / assert on env outcome
		})
	}
}
```

Pattern source: `pkg/workflowengine/workflows/credentials_test.go`. Register activities with `env.RegisterActivityWithOptions` and stub with `env.OnActivity`.

**Webapp — Vitest:**

```typescript
import { describe, expect, it } from 'vitest';
import { parseSelectorResponse } from './query';

describe('parseSelectorResponse', () => {
	it('maps snake_case API body to RunnerRecord', () => {
		expect(parseSelectorResponse({ runners: [{ is_owned: true }] })).toEqual([
			{ isOwned: true }
		]);
	});
});
```

Pattern source: `webapp/src/lib/pipeline/runner/query.test.ts`. Vitest config sets `expect: { requireAssertions: true }` — every test must assert.

**Webapp — Playwright:**

```typescript
import { expect, test } from '@playwright/test';

test('login page shows required fields', async ({ page }) => {
	await page.goto('/login');
	await expect(page.getByRole('button', { name: 'Log in' })).toBeVisible();
});
```

Pattern source: `webapp/e2e/nru/auth-forms.spec.ts`. Prefer role/placeholder selectors; E2E does not consistently use `data-testid` today.

## Mocking

**Go — Temporal:**
- `go.temporal.io/sdk/mocks` for `client.Client` (`shared_test.go`, pipeline tests).
- `go.temporal.io/sdk/testsuite` for workflow environments.

**Go — HTTP / PocketBase:**
- `httptest` for raw handlers where needed.
- PocketBase `tests.ApiScenario` drives full router + DB without a live server.
- Seed internal admin key via helpers like `seedInternalAdminKey` (pipeline tests).

**Go — local stubs:**
- Small interface implementations in test files (`queueStub` in `pkg/internal/apis/handlers/pipeline_queue_handler_test.go`).

**What to mock:**
- Temporal client, workflow history iterators, queue/semaphore boundaries.
- External network in unit tests — use stubs or `httptest`.

**What NOT to mock in unit tests:**
- Pure parsing/canonicalization logic (test directly).
- PocketBase collections when `test_pb_data` already has fixtures — use real test DB file.

**Webapp:**
- Vitest server project: no DOM; test pure functions and schema mapping.
- Browser project: real Chromium via Playwright provider — component/Svelte tests run in browser.
- Avoid mocking Svelte runes; test logic in `.ts` modules where possible.

## Fixtures and Factories

**PocketBase test data:**
- Directory: `test_pb_data/` at repository root (SQLite `data.db`).
- Known users/orgs in scenarios (e.g. `userA@example.org`, `usera-s-organization/pipeline123` in pipeline handler tests).
- Keep fixtures read-only; do not mutate committed DB for local experiments.

**Go test helpers:**
- `setupPipelineApp`, `setupUtilityTestApp`, `setupPipelineQueueApp` — centralize route registration and hooks (`pkg/internal/apis/handlers/pipeline_handler_test.go`, `utility_test.go`).

**Webapp:**
- No large shared factory module; inline minimal objects in each `describe` block.

## Coverage

**Requirements:**
- `make coverage-check` enforces **80%** minimum total coverage (`COVERAGE_MIN` in `Makefile`).
- Coverage packages exclude `webapp/node_modules` via `COVERAGE_PKGS`.

**View coverage:**

```bash
make coverage          # Opens coverage.html + treemap SVG
make coverage-check    # CI-style gate
```

## Test Types

**Unit Tests (Go):**
- Default `make test`; fast, deterministic, no live Temporal server.
- Parallel-safe tests call `t.Parallel()` at start of top-level tests when isolated (`shared_test.go`).

**Integration Tests (Go):**
- `//go:build !unit` files; run with `go test ./...` (no unit tag) when enabling full suite.
- Some handler tests named `*_Integration` but skipped (`TestGenerateApiKeyHandler_Integration` uses `t.Skip` in `api_key_integration_test.go`).

**Unit Tests (Webapp):**
- Vitest `client` + `server` projects; preferred for business logic and API mapping.

**E2E Tests (Webapp):**
- Playwright; `webServer` runs `webapp/scripts/e2e-webserver.sh` on port 5100.
- Logged tests depend on `setup` project writing `test-results/.auth/user.json`.
- Requires backend + deterministic fixtures for stable runs (documented in root `AGENTS.md`).

## CI

**Go (active):**
- `.github/workflows/go.yml` — `golangci-lint`, `govulncheck` (with ignore list), `make test` on PRs touching Go files.

**Webapp (disabled in repo):**
- `.github/workflows/webapp.yml` is fully commented out — unit tests not gated in CI currently.

## Common Patterns

**Async testing (Go):**
- Use `context.Background()` or test-scoped context; Temporal test suite handles workflow time.

**Error testing (Go):**

```go
require.Error(t, err)
require.ErrorContains(t, err, "Captcha verification failed")
```

Pattern: `pkg/internal/apis/turnstile_test.go`.

**Environment in tests:**

```go
t.Setenv("TURNSTILE_SECRET_KEY", "secret")
```

Use `t.Setenv` / `t.Cleanup` rather than global env mutation.

**Skipping:**

```go
t.Skip("reason")  // for broken or env-heavy tests
```

**Race detector:**
- Always on in `make test`; fix data races rather than disabling.

**Watch mode:**

```bash
make test.p TestFetchWorkflowFailure
```

Uses `go tool gow test` with regex `^TestName$` on `GODIRS` (`./pkg/... ./cmd/...`).

## Where to Add Tests

| Change type | Add test in |
|-------------|-------------|
| New HTTP handler | `pkg/internal/apis/handlers/<area>_test.go` with `tests.ApiScenario` or `httptest` |
| New workflow | `pkg/workflowengine/workflows/<name>_test.go` with `testsuite` |
| New activity | `pkg/workflowengine/activities/<name>_test.go` |
| Pipeline YAML logic | `pkg/workflowengine/pipeline/*_test.go` |
| Webapp pure TS | Co-located `*.test.ts` under `webapp/src/...` |
| Svelte UI behavior | `*.svelte.test.ts` (browser Vitest project) |
| Full user journey | `webapp/e2e/nru/` or `webapp/e2e/logged/` |

## Test Coverage Gaps (awareness)

- `pkg/internal/apis/handlers/scoreboard_handler_test.go` — active tests commented out; scoreboard logic lightly guarded.
- Webapp CI workflow disabled — regressions rely on local `bun run test:unit`.
- Integration-tagged Go tests not run in default CI.
- E2E depends on live stack; not part of `make test`.

---

*Testing analysis: 2026-05-27*
