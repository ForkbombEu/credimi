# Codebase Concerns

**Analysis Date:** 2026-05-27

## Tech Debt

**Legacy OTel scoreboard (dead code):**
- Issue: `pkg/internal/apis/handlers/scoreboard_handler.go` and `scoreboard_handler_test.go` are fully block-commented (`/*` … `*/`) but still contain placeholder aggregators that return hard-coded “Example Wallet/Issuer/Verifier/Pipeline” data. Routes are commented out in `pkg/internal/apis/RoutesRegistry.go` (`ScoreboardRoutes`, `ScoreboardPublicRoutes`).
- Files: `pkg/internal/apis/handlers/scoreboard_handler.go`, `pkg/internal/apis/handlers/scoreboard_handler_test.go`, `pkg/internal/apis/RoutesRegistry.go`, `docs/src/content/docs/software-architecture/scoreboard.md`
- Impact: Confuses readers and grep hits; risks accidental re-enablement of fake metrics or an unauthenticated `/api/all-results` surface.
- Fix approach: Delete the commented files or move to an `archive/` package; keep pipeline scoreboard in `pkg/internal/apis/handlers/scoreboard.go` as the single source of truth.

**Mixed HTTP error response shapes:**
- Issue: Handlers often return errors via `apierror.New(...).JSON(e)` (`{status, error, reason, message}`), while routes wrapped with `middlewares.ErrorHandlingMiddleware` normalize to `{ apiVersion: "2.0", error: { code, message, errors: [...] } }`. Some handlers mix `return err` (middleware path) and direct `.JSON(e)` in the same file.
- Files: `pkg/internal/apierror/apierror.go`, `pkg/internal/middlewares/errors.go`, `pkg/internal/apis/handlers/pipeline_handler.go` (and many other handlers)
- Impact: API clients and the webapp must handle two shapes; inconsistent UX and brittle error parsing.
- Fix approach: Pick one contract per route group; migrate handlers to return `*apierror.APIError` and let middleware wrap, or standardize on direct JSON everywhere and drop double formatting.

**Frontend Svelte 4 / typing debt:**
- Issue: Multiple `TODO` / `@ts-expect-error` markers for Svelte 5 migration and loose types; `webapp/src/modules/components/ui-custom/search.svelte` still targets Svelte 4 patterns.
- Files: `webapp/src/modules/components/ui-custom/search.svelte`, `webapp/src/modules/components/ui-custom/selectInput.svelte`, `webapp/src/modules/forms/fields/selectField.svelte`, `webapp/src/modules/utils/files.ts`, `webapp/src/lib/workflows/queries.ts`
- Impact: Type unsafety, harder refactors, possible runtime surprises after upgrades.
- Fix approach: Incremental Svelte 5 migration per module; replace `@ts-expect-error` with proper types; remove the `canBeTerminated` HACK in `queries.ts` by fixing the workflow model/store boundary.

**Record share / backend coupling:**
- Issue: `recordShare.svelte` documents a needed full refactor spanning UI and backend.
- Files: `webapp/src/modules/collections-components/manager/record-actions/recordShare.svelte`
- Impact: Sharing behavior may be inconsistent or hard to secure/extend.
- Fix approach: Design a single sharing API + PB rules, then refactor UI to match.

**Validation middleware gap:**
- Issue: TODO to expose validated values on request context beyond current binding.
- Files: `pkg/internal/middlewares/validation.go` (line ~132)
- Impact: Handlers may re-parse or duplicate validation logic.
- Fix approach: Extend context helpers in `pkg/internal/routing/routing.go` and adopt in new handlers.

**EUDI wallet test URL hardcoding:**
- Issue: `eudiw.go` builds `tests/wallet/eudiw` under `app_url` with `// TODO use the correct one`.
- Files: `pkg/workflowengine/workflows/eudiw.go`
- Impact: Wrong deeplink/QR base in non-default deployments.
- Fix approach: Drive path from pipeline config or canonified wallet record metadata.

**Monolithic handler files:**
- Issue: Very large files concentrate queue, CI temp records, and scoreboard logic.
- Files: `pkg/internal/apis/handlers/scoreboard.go` (~1363 lines), `pkg/internal/apis/handlers/pipeline_queue_handler.go` (~1246 lines)
- Impact: High merge conflict risk, difficult testing and review.
- Fix approach: Extract subpackages (`scoreboard/`, `queue/`, `ci_temp/`) with focused tests.

**Automigrate enabled at runtime:**
- Issue: `migratecmd.MustRegister` uses `Automigrate: true` in `pkg/routes/routes.go`.
- Impact: Production startups may apply migrations implicitly; risky for controlled releases.
- Fix approach: Gate `Automigrate` behind env (dev-only) and run migrations explicitly in deploy.

**math/rand for clone suffixes:**
- Issue: `clone_record.go` uses `math/rand` (not crypto-rand) for `_copy####` suffixes.
- Files: `pkg/internal/apis/handlers/clone_record.go`
- Impact: Predictable suffixes under contention (low severity for names, not secrets).
- Fix approach: Use `crypto/rand` or UUID fragment for uniqueness.

## Known Bugs

**Placeholder scoreboard data (if legacy routes re-enabled):**
- Symptoms: `/api/my/results` and `/api/all-results` would return static example entities, not real runs.
- Files: `pkg/internal/apis/handlers/scoreboard_handler.go` (commented)
- Trigger: Uncommenting `ScoreboardRoutes` / `ScoreboardPublicRoutes` in `RoutesRegistry.go`
- Workaround: Use pipeline scoreboard endpoints under `PipelineRoutes` (`/api/pipeline/scoreboard/...`) implemented in `scoreboard.go`.

**Workflow UI termination state (webapp):**
- Symptoms: Svelte reactivity issues around `canBeTerminated` on workflow executions.
- Files: `webapp/src/lib/workflows/queries.ts` (`/* HACK */` overwrites getter)
- Trigger: Listing workflows and using terminate actions
- Workaround: Hard-coded `canBeTerminated: true` (may show terminate when unsafe)

## Security Considerations

**CI temporary wallet version orphan records (V1):**
- Risk: Temp `wallet_versions` rows created for `POST /api/pipeline/run-wallet-apk` may remain if enqueue succeeds but workflow cleanup never runs; no durable `temporary` or `expires_at` on records.
- Files: `pkg/internal/apis/handlers/pipeline_wallet_ci_handler.go`, `pkg/workflowengine/pipeline/temp_wallet_cleanup_hook.go`, `pkg/internal/apis/handlers/pipeline_queue_handler.go` (`deleteTempWalletVersionForOwner`), `AGENTS.md`
- Current mitigation: Cleanup via workflow hook + internal `DELETE /api/wallet/temp-version/{record_id}`; queued cancel deletes temp version when not yet running.
- Recommendations: Add PB fields (`temporary`, `expires_at`); scheduled retention job (`pipeline_retention_handler.go` pattern); metrics on orphan count.

**Similar temp records for issuer/verifier CI:**
- Risk: Same orphan pattern for temporary credentials and use-case verifications.
- Files: `pkg/internal/apis/handlers/pipeline_issuer_ci_handler.go`, `pkg/internal/apis/handlers/pipeline_ci_helpers.go`, `pkg/workflowengine/pipeline/temp_record_cleanup_hook.go`
- Recommendations: Unified temp-record metadata and retention sweep.

**Turnstile bypass when secret unset:**
- Risk: User registration proceeds without captcha verification if `TURNSTILE_SECRET_KEY` is empty (logged and skipped).
- Files: `pkg/internal/apis/turnstile.go`, `pkg/routes/routes.go` (`HookTurnstileVerification`)
- Current mitigation: Production should always set the secret; tests cover hook wiring.
- Recommendations: Fail closed in non-dev environments; separate `TURNSTILE_SKIP` explicit flag for local dev.

**Unauthenticated route groups with handler-level auth:**
- Risk: `AuthenticationRequired: false` on route groups is easy to misread; security depends on per-route middleware (`RequireInternalAdminAPIKey`, `RequireInternalAdminOrAuth`) or handler checks (`clone_record.go`).
- Files: `pkg/internal/routing/routing.go`, `pkg/internal/apis/handlers/deeplink_handler.go`, `pkg/internal/apis/handlers/conformance_check_handlers.go`, `pkg/internal/apis/handlers/canonify.go`, `pkg/internal/apis/handlers/api_key.go`
- Current mitigation: Internal Temporal routes use `CREDIMI_INTERNAL_ADMIN_KEY`; clone uses `canDuplicateIfRequestIsFromOwner`.
- Recommendations: Audit deeplink/conformance/canonify for abuse (enumeration, SSRF via YAML workflows); document threat model per public endpoint.

**Deeplink and conformance GET endpoints (no auth on group):**
- Risk: `DeepLinkRoutes` and `ConformanceCheckRoutes` expose GET handlers without group-level auth; may leak workflow/deeplink state if IDs are guessable.
- Files: `pkg/internal/apis/handlers/deeplink_handler.go`, `pkg/internal/apis/handlers/conformance_check_handlers.go`
- Recommendations: Rate limiting, signed tokens, or auth for sensitive deeplink flows.

**Large upload body limits:**
- Risk: Wallet pipeline result storage allows very large bodies (`500 << 20` in `wallet_handler.go`; wallet APK CI uses `1000 << 20` in tests/handler patterns).
- Files: `pkg/internal/apis/handlers/wallet_handler.go`, `pkg/internal/apis/handlers/pipeline_handler.go`
- Recommendations: Enforce limits at reverse proxy; stream to object storage; monitor disk.

**Internal admin key in env:**
- Risk: `CREDIMI_INTERNAL_ADMIN_KEY` gates internal HTTP activities and routes; compromise grants broad internal API access.
- Files: `pkg/workflowengine/activities/internal_http.go`, `pkg/internal/middlewares` (RequireInternalAdmin*)
- Recommendations: Rotate keys, short-lived tokens, mTLS between Temporal workers and API where possible.

**Reverse proxy catch-all:**
- Risk: `pkg/routes/routes.go` proxies `/{path...}` to `ADDRESS_UI`; misconfiguration could expose wrong upstream or log sensitive paths when `DEBUG` is set.
- Files: `pkg/routes/routes.go`
- Recommendations: Ensure API routes register before catch-all; restrict `DEBUG` in production.

## Performance Bottlenecks

**Temporal workflow listing default page size 1000:**
- Problem: `pipelineListWorkflowsDefaultLimit = 1000` caps list requests; large tenants may still hit heavy Temporal queries and PB joins when building pipeline execution views.
- Files: `pkg/internal/apis/handlers/pipeline_handler.go`, `pkg/internal/apis/handlers/pipeline_results_handler.go`, `pkg/internal/apis/handlers/scoreboard.go`
- Cause: List + enrich executions with pipeline metadata and children (`childWorkflowParentQueryPageSize = 1000`).
- Improvement path: Cursor-based pagination, narrower projections, cache pipeline identifier index.

**Aggregate scoreboard batch processing:**
- Problem: Scoreboard aggregation walks pipeline records in batches (`scoreboardPipelineRecordBatchSize = 250`) and Temporal history—expensive for busy namespaces.
- Files: `pkg/internal/apis/handlers/scoreboard.go`, `pkg/workflowengine/workflows/scoreboard.go`
- Improvement path: Materialized scoreboard table updated incrementally on pipeline completion (hook on `pipeline_results`).

**Scoreboard relation graph walks:**
- Problem: `errScoreboardRelationSkipped` indicates optional relations skipped during aggregation—partial stats under load or schema drift.
- Files: `pkg/internal/apis/handlers/scoreboard.go`
- Improvement path: Explicit metrics when relations skip; tighten schema contracts.

## Fragile Areas

**Mobile runner semaphore + queue pipeline start:**
- Files: `pkg/workflowengine/workflows/mobile_runner_semaphore.go`, `pkg/internal/apis/handlers/pipeline_queue_handler.go`, `pkg/workflowengine/activities/queued_pipeline.go`
- Why fragile: Cross-namespace Temporal updates (`default` semaphore → org pipeline), ticket metadata for CI cleanup, and best-effort `pipeline_results` creation.
- Safe modification: Run `pipeline_queue_handler_test.go`, `queued_pipeline_test.go`, and mobile automation hook tests; use GitNexus impact on `EnqueueRun` / `StartQueuedPipelineActivity`.
- Test coverage: Good unit coverage on queue stubs; limited true Temporal integration in CI.

**Pipeline YAML parsing and mobile-automation invariants:**
- Files: `pkg/workflowengine/pipeline/parser.go`, `pkg/workflowengine/pipeline/mobile_automation_hooks.go`
- Why fragile: `with.payload` / `with.config` merge rules and `runner_id` / `global_runner_id` requirements affect every run.
- Safe modification: Extend `schemas/pipeline/pipeline_schema.json` and parser tests together.
- Test coverage: Strong unit tests in `pkg/workflowengine/pipeline/*_test.go`.

**Canonified paths and multi-tenant namespaces:**
- Files: `pkg/internal/canonify/`, `pkg/internal/pb/namespaces.go`, `pkg/workflowengine/hooks/hook.go`
- Why fragile: `organizations.canonified_name` maps to Temporal namespace; org create/update must keep workers in sync.
- Safe modification: Always test org lifecycle hooks; never rename canonify rules without migrations.

**Dynamic pipeline workflow registry:**
- Files: `pkg/workflowengine/registry/registry.go`, denylist for `mobile-automation` on pipeline worker
- Why fragile: Runner workers must poll `${runner_id}-TaskQueue` in the org namespace while pipeline worker stays separate.
- Test coverage: Registry tests; runner contract documented in `AGENTS.md`.

**credimi-extra private module:**
- Files: `go.mod`, `pkg/workflowengine/activities/mobileflow.go`, CI `CREDIMI_EXTRA_PAT` in `.github/workflows/go.yml`
- Why fragile: Local/CI builds fail without PAT; module must not be removed by `go mod tidy`.
- Safe modification: Preserve dependency per `AGENTS.md`; document PAT setup in contributor docs.

## Scaling Limits

**PocketBase SQLite (`pb_data/`):**
- Current capacity: Single-node SQLite suitable for dev/small deployments.
- Limit: Write contention, backup size, and concurrent pipeline result inserts under heavy CI.
- Scaling path: Managed Postgres (if PocketBase/supporting store migrated), retention policies (`pipeline_retention_handler.go`).

**Temporal dev DB co-located:**
- Current capacity: `pb_data/temporal.db` with dev server (`Procfile.dev`).
- Limit: Not a production HA topology; namespace-per-org multiplies worker memory.
- Scaling path: External Temporal cluster, dedicated search attributes, worker autoscaling per hot namespace.

**Mobile runner semaphore (per runner, `default` namespace):**
- Current capacity: One workflow per `runner_id` serializes runs.
- Limit: Queue depth grows if runners slow; `MOBILE_RUNNER_SEMAPHORE_WAIT_TIMEOUT` behavior.
- Scaling path: Multiple runners, capacity tuning on semaphore workflow, observability on `GET /api/mobile-runner/semaphore`.

## Dependencies at Risk

**github.com/forkbombeu/credimi-extra (private):**
- Risk: Unavailable without GitHub PAT; cannot be pruned by tidy; ties mobile automation to external runner contracts.
- Impact: CI and local `go test` fail; mobile pipeline steps break.
- Migration plan: None intended—treat as required platform dependency; mirror or vendor only if org policy changes.

**Ignored govulncheck IDs in CI:**
- Risk: `.github/workflows/go.yml` explicitly ignores several `GO-2026-*` vulnerability IDs.
- Impact: Known vulns may remain unaddressed until ignore list is reviewed.
- Migration plan: Periodic audit of `IGNORE_IDS`; upgrade stdlib/deps when fixes exist.

## Missing Critical Features

**Durable temp-resource lifecycle:**
- Problem: No first-class expiry/TTL on temporary wallet versions (and related CI artifacts).
- Blocks: Safe broad production use of wallet APK CI and similar one-shot runs without manual DB cleanup.

**Integration test suite in CI:**
- Problem: `make test` runs only `-tags=unit`; `//go:build !unit` tests (e.g. `pkg/workflowengine/workflows/zenroom_test.go`, `pkg/internal/apis/templating_test.go`) are not in the default CI job.
- Blocks: Confidence in Temporal, Zenroom, and full HTTP stacks without manual opt-in.

**Real aggregate dashboard for wallets/issuers/verifiers (legacy OTel tab):**
- Problem: Product docs reference retired `/api/my/results` style aggregations; replacement is pipeline-centric scoreboard only.
- Blocks: Cross-entity OTel tab unless rebuilt on `pipeline_results` + Temporal.

## Test Coverage Gaps

**API key integration tests skipped:**
- What's not tested: `GenerateApiKeyHandler` and `AuthenticateApiKeyHandler` integration paths.
- Files: `pkg/internal/apis/handlers/api_key_integration_test.go` (`t.Skip`)
- Risk: Regressions in API key auth bypass or token issuance.
- Priority: High

**Zenroom / CESR / templating integration:**
- What's not tested: Full workflow paths requiring external binaries or Temporal.
- Files: `pkg/workflowengine/workflows/zenroom_test.go`, `pkg/workflowengine/activities/cesr_test.go`, `pkg/internal/apis/templating_test.go`
- Risk: Cryptographic or template regressions ship unnoticed.
- Priority: Medium (run locally before releases)

**Legacy scoreboard handler tests commented:**
- What's not tested: Entire `scoreboard_handler_test.go` block-commented with dead handler code.
- Files: `pkg/internal/apis/handlers/scoreboard_handler_test.go`
- Risk: Low (dead code); confusion if revived.
- Priority: Low (delete dead code)

**E2E webapp against live backend:**
- What's not tested: Default CI does not run `cd webapp && bun run test:e2e` with full stack.
- Files: `webapp/` E2E specs, `AGENTS.md` test conventions
- Risk: UI/API contract drift (error shapes, pipeline queue UX).
- Priority: Medium

**Turnstile only partially tested:**
- What's not tested: Live Cloudflare verification; only hook behavior with missing secret.
- Files: `pkg/internal/apis/turnstile_test.go`
- Risk: Misconfigured production captcha.
- Priority: Medium

---

*Concerns audit: 2026-05-27*
