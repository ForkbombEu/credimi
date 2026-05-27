# Coding Conventions

**Analysis Date:** 2026-05-27

## Naming Patterns

**Files:**
- Go source: `snake_case.go`; tests: `*_test.go` co-located with implementation (`pkg/internal/apis/handlers/pipeline_handler_test.go`).
- Svelte components: `PascalCase.svelte`; Svelte modules: `kebab-case.svelte.ts` or `camelCase.svelte.ts` (`webapp/src/lib/pipeline-form/execution-target/state.svelte.ts`).
- Webapp unit tests: `*.test.ts` next to the module (`webapp/src/lib/pipeline/runner/query.test.ts`).
- E2E: `webapp/e2e/nru/*.spec.ts` (not logged in), `webapp/e2e/logged/*.spec.ts` (authenticated).

**Functions (Go):**
- Exported: `PascalCase` (`GetUserOrganizationID` in `pkg/internal/apis/handlers/utility_test.go`).
- Unexported helpers: `camelCase` (`setupPipelineApp`, `fetchWorkflowFailure`).
- Test names: `Test<Subject>_<Scenario>` for focused cases (`TestPipelineQueueEnqueue_RollbackOnPartialFailure` in `pkg/internal/apis/handlers/pipeline_queue_handler_test.go`); top-level `Test<Name>` with `t.Run` subtests for tables (`Test_CredentialsIssuersWorkflow` in `pkg/workflowengine/workflows/credentials_test.go`).

**Functions (TypeScript):**
- `camelCase` for functions and utilities (`parseSelectorResponse`, `getExceptionMessage`).
- Vitest: `describe('module', () => { it('does X', () => { ... }) })`.

**Variables:**
- Go: short locals (`app`, `err`, `coll`); struct fields with JSON tags matching API contracts (`pkg/internal/apierror/apierror.go`).
- TS: `camelCase`; API snake_case mapped to camelCase at boundaries (`is_owned` → `isOwned` in `webapp/src/lib/pipeline/runner/query.test.ts`).

**Types:**
- Go: exported structs/interfaces `PascalCase`; CRE codes via `pkg/internal/errorcodes`.
- TS: `type` aliases and interfaces `PascalCase`; strict mode enabled (`webapp/tsconfig.json`).

## Code Style

**Formatting (Go):**
- Tabs for indentation (`.editorconfig` `[*.go]`).
- Enforced via `golangci-lint` formatters in `.golangci.yaml`: `gofumpt` (module `github.com/forkbombeu/credimi`), `gci`, `goimports`, `golines`.
- `make fmt` runs `go fmt` on `./pkg/...` and `./cmd/...`.
- `make lint` runs `go mod tidy -diff`, `go mod verify`, `go vet`, `govulncheck`, `golangci-lint run`.

**Formatting (Webapp):**
- Prettier: tabs, single quotes, `printWidth: 100`, no trailing commas (`webapp/.prettierrc`).
- Plugins: `prettier-plugin-svelte`, `prettier-plugin-tailwindcss` (class order).
- `bun run format` / `bun run lint` (Prettier check + ESLint).

**Linting:**
- Go: broad linter set in `.golangci.yaml` (revive, staticcheck, gocritic, etc.); test files get relaxed rules under `path: _test\.go`.
- Webapp: flat ESLint 9 config (`webapp/eslint.config.js`) — TypeScript recommended, Svelte recommended, `eslint-config-prettier`, `perfectionist/sort-imports` as error.
- Ignored ESLint paths: generated UI shadcn (`src/modules/components/ui/**`), Paraglide (`src/paraglide/**`).

**License headers:**
- SPDX comment block at top of every source file (enforced by pre-commit `reuse` hook in `.pre-commit-config.yaml`).

## Import Organization

**Go (gci / project convention):**
1. Standard library
2. Third-party modules
3. `github.com/forkbombeu/credimi/...` internal packages

Example from `pkg/internal/apis/handlers/pipeline_handler_test.go`:

```go
import (
	"context"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)
```

**TypeScript / Svelte:**
- ESLint `perfectionist/sort-imports` enforces sorted import groups.
- SvelteKit path aliases come from generated `.svelte-kit/tsconfig.json` (extends `webapp/tsconfig.json`).
- Prefer `$lib/...` and module paths under `src/modules/...` per existing files.

## Error Handling

**Go — domain HTTP errors:**
- Handlers return `*apierror.APIError` from `pkg/internal/apierror/apierror.go` with shape `{status, error, reason, message}` and `.JSON(r)` for responses.
- Construct with `apierror.New(code, domain, reason, message)`.

**Go — middleware-wrapped errors:**
- `pkg/internal/middlewares/errors.go` maps `*apierror.APIError` to Google-style JSON: `{ apiVersion: "2.0", error: { code, message, errors: [{domain, reason, message}] } }`.
- Follow the error shape already used by the endpoint you modify (handlers sometimes write `APIError` JSON directly without middleware).

**Go — wrapping:**
- Use `fmt.Errorf("context: %w", err)` for non-domain errors (`pkg/internal/pb/schedules.go`, `pkg/internal/canonify/resolver.go`).

**Go — Temporal / workflows:**
- Activities: `workflowengine.BaseActivity` helpers → `temporal.NewApplicationError` with CRE-related `errorType` (`pkg/workflowengine/activity.go`).
- Workflows: `workflowengine.NewWorkflowError` for UI metadata; parse with `workflowengine.ParseWorkflowError` where handlers expose workflow failures.
- CRE codes: `pkg/internal/errorcodes/errorcodes.go` as source of truth.

**TypeScript:**
- Utilities in `webapp/src/modules/utils/errors.ts` — `getExceptionMessage`, `exceptionToError`.
- `effect` / `Either` for async validation flows in standards and pipeline code (`webapp/src/lib/standards/index.ts`).
- Zod schemas in PocketBase and form modules (`webapp/src/modules/pocketbase/zod-schema/`).

## Logging

**Go:**
- Standard library `log` in middleware (`pkg/internal/middlewares/errors.go` logs handled vs unhandled errors).
- Prefer structured context in error messages over ad-hoc debug prints in handlers.

**Webapp:**
- No centralized logging framework; rely on browser devtools and test assertions for unit tests.

## Comments

**When to Comment:**
- Non-obvious business rules (pipeline `with` payload merge, semaphore ticket metadata, internal-only config keys).
- Skip comments that restate function names.

**Go doc:**
- Package comments on larger packages (`pkg/workflowengine/activity.go`).

**Tests:**
- Table `name` fields and `t.Run` descriptions carry scenario intent; avoid redundant test header comments.

## Function Design

**Size:**
- `golangci` `funlen` disabled globally but configured (90 lines / 50 statements) when enabled; keep handlers focused and extract helpers (pattern in `pkg/internal/apis/handlers/shared.go`).

**Parameters:**
- Route handlers receive `*core.RequestEvent`; validated input via `routing.GetValidatedInput[T](e)` (`pkg/internal/routing/routing.go`).
- Workflow/activity inputs use `workflowengine.WorkflowInput` / `ActivityInput` with `Config` and `Payload` maps.

**Return Values:**
- HTTP handlers: `error` from PocketBase route func; success via `r.JSON(...)`.
- Activities: `(ActivityResult, error)`.

## Module Design

**Exports:**
- Route groups registered in `pkg/internal/apis/RoutesRegistry.go`; each domain has a `*Routes` `RouteGroup` with `Add(app)`.
- Handlers exposed as `HandlerFactory` closures for OpenAPI/validation binding (`pkg/internal/routing/routing.go`).

**Dependency injection:**
- Prefer constructor functions returning interfaces (API key crypto/hasher in `pkg/internal/apis/handlers/api_key_service_test.go`).
- Test doubles: `temporalmocks.Client`, local stub structs (e.g. `queueStub` in `pipeline_queue_handler_test.go`).

**Barrel files:**
- Webapp feature modules use `index.ts` under `webapp/src/lib/...` where present; do not barrel-export generated code.

## Webapp-Specific Patterns

**Svelte 5:**
- Use runes (`$state`, `$derived`, `$effect`) in `.svelte.ts` state modules (`webapp/src/lib/utils/state.svelte.ts`).
- Run `svelte-check` via `bun run check` before merge.

**Styling:**
- Tailwind v4 via `@tailwindcss/vite`; Prettier sorts Tailwind classes.
- Tabs in `.svelte` / `.ts` per `webapp/.editorconfig`.

**Generated code:**
- Do not hand-edit `*.generated.ts`, Paraglide output, or `src/modules/components/ui/**` (regenerate via `bun run generate:definitions` and UI tooling).

## Pre-Commit & Tooling

**Hooks (`.pre-commit-config.yaml`):**
- `check-yaml`, `check-json`, `check-toml`, `check-xml`, large files, merge conflicts.
- `reuse` for SPDX compliance.

**Dev setup:**
- `make devtools` installs pre-commit hooks.
- `mise` pins Go 1.25.x and Bun (`.mise.toml`); `make tools` runs `mise install`.

---

*Convention analysis: 2026-05-27*
