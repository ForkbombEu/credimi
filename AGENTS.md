<!--
SPDX-FileCopyrightText: 2025 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2025 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# AGENTS.md

## Build / Test

- `make dev` runs hivemind Procfile.dev (API + UI) after ensuring tools.
- `make test` executes Go unit suite with `-tags=unit`.
- `go test ./pkg/... -run TestName -tags=unit` runs a focused Go test.
- `make test.p TestName` watches and reruns the matching Go test via gow.
- `make lint` runs gomod tidy/verify, govet, govulncheck, golangci-lint.
- `make fmt` applies gofmt across Go packages.
- `go run ./main.go` starts the PocketBase-backed API locally.
- `make generate` triggers Go code generation prerequisites.

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
