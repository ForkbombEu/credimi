# Technology Stack

**Analysis Date:** 2026-05-27

## Languages

**Primary:**
- Go 1.25.x — Backend API, Temporal workers/activities, CLI, code generation (`go.mod` specifies `go 1.25.3`; `.mise.toml` pins `go 1.25.5`)
- TypeScript — SvelteKit webapp (`webapp/`), Vitest/Playwright tests, codegen scripts
- JavaScript — PocketBase migrations (`pb_migrations/`), MJML email build scripts, some tooling

**Secondary:**
- YAML — Pipeline definitions, config templates, Temporal dynamic config (`schemas/`, `config_templates/`, `deployment/dynamicconfig/`)
- Markdown — Docs site content (`docs/src/content/docs/`)
- Svelte — UI components (`webapp/src/**/*.svelte`)
- Shell — Dev/prod process orchestration (`scripts/`, `Makefile`, `Procfile*`)

## Runtime

**Environment:**
- **Production / Docker:** Debian 12 slim container running compiled `credimi` binary + built SvelteKit UI served via Bun (`Dockerfile`, `Procfile`)
- **Local dev:** Host Go process (`go tool gow run -tags=credimi_extra main.go serve`) + Bun Vite dev server; Temporal via Docker Compose (`make dev`)

**Package Managers:**
- **Go modules** — `go.mod` / `go.sum` (module `github.com/forkbombeu/credimi`)
- **Bun** — `webapp/bun.lock` (primary JS runtime; `webapp/package.json` also documents `pnpm` in `engines` but scripts use `bun`)
- **Bun** — `docs/bun.lock` for Astro docs (install via `cd docs && bun i`)

**Tool version manager:** [mise](https://mise.jdx.dev/) — `.mise.toml` pins `go`, `bun`, `node`, `pre-commit`, and `aqua:dyne/slangroom-exec`

## Frameworks

**Core (backend):**
- [PocketBase](https://pocketbase.io/) v0.26.4 (`github.com/pocketbase/pocketbase`) — HTTP server, SQLite persistence, auth, collections, hooks; entry via `cmd/credimi.go` → `pocketbase.New()` + `routes.Setup`
- [Temporal Go SDK](https://github.com/temporalio/sdk-go) v1.36.0 — Workflow orchestration, org-scoped namespaces, pipeline and conformance workflows (`pkg/workflowengine/`)
- [Cobra](https://github.com/spf13/cobra) v1.10.2 — CLI subcommands (`cmd/cli/`)

**Core (frontend):**
- [Svelte](https://svelte.dev/) 5.x + [SvelteKit](https://kit.svelte.dev/) 2.x — SPA/SSR app (`webapp/`)
- [Vite](https://vitejs.dev/) 7.x — Bundler and dev server (`webapp/vite.config.ts`)
- [Tailwind CSS](https://tailwindcss.com/) 4.x — Styling (`@tailwindcss/vite`)
- [svelte-adapter-bun](https://github.com/gornostay25/svelte-adapter-bun) — Production adapter (`webapp/svelte.config.js`)

**Docs:**
- [Astro](https://astro.build/) 6.x + [Starlight](https://starlight.astro.build/) — Documentation site (`docs/package.json`)

**Testing:**
- Go: `github.com/stretchr/testify` — Unit tests (`go test -tags=unit ./...` via `make test`)
- Webapp: [Vitest](https://vitest.dev/) 4.x (unit) + [Playwright](https://playwright.dev/) 1.57 (E2E) — `webapp/package.json` scripts `test:unit`, `test:e2e`

**Build / Dev:**
- [Hivemind](https://github.com/DarthSim/hivemind) — Multi-process dev (`make dev` → `Procfile.dev`)
- [gow](https://github.com/mitranim/gow) — Go file watcher for API in dev (`Procfile.dev`)
- [golangci-lint](https://golangci-lint.run/) — `.golangci.yaml` (invoked via `make lint`)
- [GoReleaser](https://goreleaser.com/) — Release binaries (`.github/workflows/release.yml`)
- [pre-commit](https://pre-commit.com/) — Git hooks (mise tool)

## Key Dependencies

**Critical (Go):**
| Package | Version | Role |
|---------|---------|------|
| `github.com/pocketbase/pocketbase` | 0.26.4 | App framework, DB, routing base |
| `go.temporal.io/sdk` | 1.36.0 | Workflows and activities |
| `github.com/forkbombeu/credimi-extra` | 1.10.4 | Private module: mobile automation, external runner integration (`//go:build credimi_extra`) |
| `github.com/docker/docker` | 28.3.3 | Docker activity for conformance/tooling (`pkg/workflowengine/activities/docker.go`) |
| `github.com/go-playground/validator/v10` | 10.26.0 | Request/payload validation |
| `github.com/swaggest/openapi-go` | 0.2.60 | OpenAPI spec generation (`pkg/generate_client/`) |
| `gopkg.in/gomail.v2` | — | SMTP mail activity (`pkg/workflowengine/activities/email.go`) |
| `github.com/joho/godotenv` | 1.5.1 | `.env` loading at startup (`cmd/credimi.go`) |

**Critical (webapp):**
| Package | Version | Role |
|---------|---------|------|
| `pocketbase` (JS SDK) | 0.26.6 | Client to backend API |
| `effect` | 3.19.x | Async/data-flow utilities |
| `zod` | 4.3.x | Runtime validation |
| `zenroom` | 5.28.x | Zencode/crypto in browser |
| `@forkbombeu/temporal-ui` | Git dep | Embedded Temporal UI views |
| `@sjsf/form` + `ajv` | — | JSON Schema forms |
| `@inlang/paraglide-js` | 2.6.x | i18n (`project.inlang`) |

**Infrastructure / tooling binaries (installed in Docker image or via mise/Makefile):**
- `stepci-captured-runner` — Conformance HTTP checks (`Dockerfile`, `RUN_STEPCI_PATH`)
- `et-tu-cesr` — CESR-related tooling (`pkg/workflowengine/activities/cesr.go`, `github.com/ForkbombEu/et-tu-cesr` in go.mod)
- `slangroom-exec` — Slangroom/Zencode execution (mise: `aqua:dyne/slangroom-exec`)

**Build tags:**
- `credimi_extra` — Required for mobile automation activities and production Docker build (`-tags=credimi_extra`)
- `unit` — Excludes integration tests in default `make test`

## Configuration

**Environment:**
- Root `.env` loaded via `godotenv.Load()` in `cmd/credimi.go` (never commit secrets; `.env.example` documents variable names only)
- Webapp public vars via SvelteKit `$env/static/public` (e.g. `PUBLIC_POCKETBASE_URL`, `PUBLIC_TURNSTILE_SITE_KEY`)
- Central accessor: `pkg/utils/env.go` (`GetEnvironmentVariable`, `GetEnvironmentVariableAsInteger`)

**Key configs (names only — see `.env.example` and `AGENTS.md`):**
| Variable | Purpose |
|----------|---------|
| `ROOT_DIR` | Repo root for templates/schemas |
| `TEMPORAL_ADDRESS` | Temporal gRPC host:port (default `localhost:7233`) |
| `ADDRESS_UI` | Reverse proxy target for UI (`pkg/routes/routes.go`) |
| `DATA_DB_PATH` | PocketBase SQLite path (Docker: `/app/pb_data/data.db`) |
| `CREDIMI_INTERNAL_ADMIN_KEY` | Internal HTTP activity auth |
| `OPENIDNET_TOKEN` | OpenID conformance network API |
| `TURNSTILE_SECRET_KEY` / `PUBLIC_TURNSTILE_SITE_KEY` | Cloudflare Turnstile |
| `CREDIMI_ELASTIC_PASSWORD` | Elasticsearch for Temporal stack (Compose) |
| `SMTP_*`, `MAIL_*` | Outbound email defaults |
| `MOBILE_RUNNER_SEMAPHORE_*` | Queue/semaphore behavior |

**Build:**
- `Makefile` — `dev`, `test`, `lint`, `fmt`, `generate`, `docker`, coverage
- `docker-compose.yaml` + `docker-compose.override.yml` — Temporal, Postgres, Elasticsearch, credimi app; optional Prometheus/Grafana
- `Dockerfile` — Multi-stage Go build + Bun UI build + mise tools
- `Procfile` / `Procfile.dev` — Process definitions for Overmind/Hivemind
- `webapp/vite.config.ts`, `webapp/svelte.config.js`, `webapp/tsconfig.json`
- `.golangci.yaml`, `webapp/.prettierrc`, `webapp/eslint.config.js` (via flat config in package)

**Codegen:**
- `go generate` on `pkg/gen.go` — Config templates, OpenAPI, pipeline JSON schema
- `webapp` scripts `generate:*` — PocketBase types, features, roles, API client (`webapp/package.json`)

## Platform Requirements

**Development:**
- mise (or manual install of Go 1.25+, Bun 1.3+, Temporal CLI optional)
- Docker + Docker Compose — Temporal stack for `make dev` (Postgres 16, Elasticsearch 7.17, Temporal 1.29.x images pinned in `Makefile` `dev` target)
- Git with access to private `github.com/forkbombeu/credimi-extra` (PAT for CI/local: `CREDIMI_EXTRA_PAT`)
- Submodules: `make dev` runs `git submodule update` (e.g. `webapp/client_zencode`)

**Typical dev ports:**
| Service | Port | Source |
|---------|------|--------|
| PocketBase API | 8090 | `Procfile.dev` / Compose |
| SvelteKit UI | 5100 | `webapp` Vite default |
| Temporal gRPC | 7233 | Docker Compose |
| Temporal UI | 8280 | Docker Compose |
| Grafana (optional) | 8085 | `docker-compose.override.yml` |
| Prometheus (optional) | 9090 | `docker-compose.override.yml` |

**Production:**
- **Hosting:** Coolify deploy webhook on `main` VERSION bump (`.github/workflows/release.yml` — `COOLIFY_CREDIMI_WEBHOOK_URL`, `COOLIFY_WEBHOOK_TOKEN` secrets)
- **Container:** Single `credimi` image (API + static UI); Temporal/DB often external or Compose-managed
- **Release artifacts:** GoReleaser + cosign attestations on tag from `VERSION` file

**Persistence (runtime data, not in repo):**
- PocketBase: `pb_data/` (SQLite `data.db`, uploads)
- Temporal dev (optional standalone): `pb_data/temporal.db` per `AGENTS.md`

---

*Stack analysis: 2026-05-27*
