<!--
SPDX-FileCopyrightText: 2025 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2025 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# AGENTS.md

## Architecture (LLM-first, current branch)

### What runs (dev)

- Temporal dev server: `temporal server start-dev --db-filename pb_data/temporal.db` (Procfile: `BE`, gRPC at `localhost:7233`, UI port per Temporal defaults).
- PocketBase API: `go run main.go serve` (Procfile: `API`, default `localhost:8090`, data in `pb_data/`).
- Webapp (SvelteKit/Vite): `cd webapp && bun dev` (Procfile: `UI`, default `localhost:5100`).
- Reverse proxy: PocketBase proxies `/{path...}` to `ADDRESS_UI` (default `http://localhost:5100`) in `pkg/routes/routes.go`.

### Persistence

- PocketBase SQLite: `pb_data/`.
- Temporal dev DB: `pb_data/temporal.db`.

### Key env vars

- `TEMPORAL_ADDRESS` (Temporal host:port).
- `ADDRESS_UI` (UI reverse proxy target).
- `MOBILE_RUNNER_SEMAPHORE_DISABLED`.
- `MOBILE_RUNNER_SEMAPHORE_WAIT_TIMEOUT`.

## Build / Test

- `make dev` runs hivemind Procfile.dev (API + UI) after ensuring tools.
- `make test` executes Go unit suite with `-tags=unit`.
- `go test ./pkg/... -run TestName -tags=unit` runs a focused Go test.
- `make test.p TestName` watches and reruns the matching Go test via gow.
- `make lint` runs gomod tidy/verify, govet, govulncheck, golangci-lint.
- `make fmt` applies gofmt across Go packages.
- `go run ./main.go` starts the PocketBase-backed API locally.
- `make generate` triggers Go code generation prerequisites.

## Test Suites

- Go unit (default): `make test` or `go test -tags=unit ./...` (deterministic; no external services).
- Go integration (opt-in): `go test ./...` without `-tags=unit` to include `//go:build !unit` tests; requires external services (e.g., Temporal) if/when enabled. Currently deferred; no CI job runs these yet.
- Webapp unit: `cd webapp && bun run test:unit -- --run` (fast; pure module tests preferred).
- Webapp E2E (opt-in): `cd webapp && bun run test:e2e` (requires a running backend + deterministic fixtures).

## Webapp

- `cd webapp && bun install` syncs deps; bun is the default JS runtime.
- `cd webapp && bun run dev` starts Vite dev server (after `bun run predev`).
- `cd webapp && bun run lint` runs Prettier (check) then ESLint.
- `cd webapp && bun run test:unit -- -t "spec name"` executes targeted Vitest.
- `cd webapp && bun run test:e2e -- tests/path.spec.ts` runs Playwright per file.
- `cd webapp && bun run check` runs SvelteKit typecheck; use `--watch` to iterate.

## Docs

- `cd docs && bun i && bun run docs:dev --host` serves documentation.

## Code Style

- Go formatting via gofumpt/gofmt/gci/golines; tabs per .editorconfig; import order stdlib/third-party/internal.
- Wrap errors with `fmt.Errorf("context: %w", err)` and prefer `CredimiError` for domain surfaces.
- Tests use `stretchr/testify` with table-driven cases; rely on `require`/`assert` helpers.
- Favor dependency injection over globals; keep constructors returning interfaces.
- Svelte/TS code: Prettier tabs + single quotes + width 100; ESLint perfectionist sorts imports.
- Use TypeScript-first patterns, prefer type aliases for unions, rely on Tailwind (sorted by plugin).
- Reuse `effect`/`zod` utilities for async flows/validation; no Cursor/Copilot rule overrides present.

## Private Dependencies

- `github.com/forkbombeu/credimi-extra` is a **private module**.
- It must **never** be removed from `go.mod` or `go.sum`, even if:
    - `go mod tidy` marks it as unused.
    - CI cannot resolve it.
    - static analysis or lint tooling flags it.
- Any changes touching Go module files MUST preserve this entry unless explicitly instructed by a human maintainer.

## Test Conventions

- Go: table-driven `TestX_Y` names, `require` for hard failures, avoid IO in unit tests.
- Webapp: prefer pure-module Vitest tests; for E2E use stable selectors (`data-testid`) and avoid time-based waits.
- Fixtures: use `fixtures/test_pb_data` (or `test_pb_data` where already established) and keep fixtures read-only.
