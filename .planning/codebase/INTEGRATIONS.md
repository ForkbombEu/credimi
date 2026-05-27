# External Integrations

**Analysis Date:** 2026-05-27

## APIs & External Services

**Temporal (workflow engine):**
- **What:** Pipeline runs, conformance checks, mobile-runner semaphore, org namespaces
- **Client:** `go.temporal.io/sdk` — `pkg/internal/temporalclient/client.go`, `pkg/workflowengine/hooks/hook.go`
- **Connection:** `TEMPORAL_ADDRESS` (default gRPC `localhost:7233`)
- **Server (dev/prod stack):** Docker images `temporalio/auto-setup`, `temporalio/ui`, `temporalio/admin-tools` (`docker-compose.yaml`)
- **UI:** Bundled Temporal UI image on port 8280; webapp also uses `@forkbombeu/temporal-ui` Git package

**Cloudflare Turnstile (bot protection):**
- **What:** CAPTCHA on user registration (non-OAuth signups)
- **Implementation:** `pkg/internal/apis/turnstile.go` — POST to `https://challenges.cloudflare.com/turnstile/v0/siteverify`
- **Auth:** `TURNSTILE_SECRET_KEY` (server); `PUBLIC_TURNSTILE_SITE_KEY` (webapp build-time, `webapp/src/routes/(nru)/register/+page.svelte`)
- **Client header:** `X-Turnstile-Token` on PocketBase user create

**OpenID conformance / OpenIDNet:**
- **What:** Automated wallet/issuer/verifier conformance workflows
- **Implementation:** `pkg/workflowengine/workflows/conformance_check.go`, `openid4vp_wallet.go`, `openid4vci_issuer.go`, `openid_conformance.go`
- **Auth:** `OPENIDNET_TOKEN` bearer token (required in several workflow paths via `utils.GetEnvironmentVariable(..., true)`)
- **Schemas/templates:** `schemas/openidnet/`, `schemas/webuild/`, `schemas/ewc/`, generated into `config_templates/`

**Mobile runners (external devices):**
- **What:** APK fetch, pipeline result upload, device automation for `mobile-automation` pipeline steps
- **Private SDK:** `github.com/forkbombeu/credimi-extra/mobile` (`pkg/workflowengine/activities/mobileflow.go`, build tag `credimi_extra`)
- **Discovery:** PocketBase `mobile_runners` collection; internal lookup `GET /api/mobile-runner` (`pkg/internal/apis/handlers/mobile_runners_handlers.go`)
- **Runner HTTP API (external service, not in this repo):**
  - `POST {runner_url}/fetch-apk-and-action`
  - `POST {runner_url}/store-pipeline-result`
- **Task queues:** `${runner_id}-TaskQueue` in org Temporal namespace (`AGENTS.md`, `pkg/workflowengine/pipeline/mobile_automation_hooks.go`)

**SMTP (email):**
- **What:** Workflow-driven notification mail
- **Client:** `gopkg.in/gomail.v2` — `pkg/workflowengine/activities/email.go`
- **Config:** `SMTP_HOST` (default `smtp.apps.forkbomb.eu`), `SMTP_PORT` (default `1025`), `MAIL_SENDER`, optional `MAIL_USERNAME` / `MAIL_PASSWORD`
- **Tests:** `github.com/mocktools/go-smtp-mock` in `pkg/workflowengine/activities/email_test.go`

**Docker Engine:**
- **What:** Run containerized conformance/tool images from workflows
- **Client:** `github.com/docker/docker` — `pkg/workflowengine/activities/docker.go`
- **Deploy:** Docker socket mounted in Compose (`docker-compose.yaml` `credimi` service: `/var/run/docker.sock`)
- **Example image reference:** `ghcr.io/forkbombeu/appraccon:latest` in `pkg/workflowengine/workflows/wallet.go`

**GitHub (App API):**
- **What:** Post PR comments from automation
- **Implementation:** `pkg/internal/githubapp/pr_comments.go`
- **Auth:** `GITHUB_APP_PRIVATE_KEY`, `GITHUB_APP_CLIENT_ID` or `GITHUB_APP_ID`, optional `GITHUB_API_URL`
- **CI:** Tests in `pkg/internal/githubapp/pr_comments_test.go`

**StepCI / HTTP conformance runners:**
- **What:** Execute StepCI scripts against issuers/verifiers
- **Binary:** `stepci-captured-runner` (downloaded in `Dockerfile` to `.bin/`)
- **Env:** `RUN_STEPCI_PATH` (Compose: `pkg/OpenID4VP/stepci/runStepCI.js`)
- **Activity:** `pkg/workflowengine/activities/stepci.go`

**CESR tooling:**
- **Package:** `github.com/ForkbombEu/et-tu-cesr` in go.mod
- **Binary:** `et-tu-cesr` in `.bin/` (`pkg/workflowengine/activities/cesr.go`)

**Zenroom / Slangroom (cryptographic scripts):**
- **Browser:** `zenroom` npm package in webapp
- **Server workflows:** `pkg/workflowengine/workflows/zenroom.go` (Docker-based execution paths)
- **CLI:** `slangroom-exec` via mise (`.mise.toml`)

**Internal credimi HTTP (Temporal activities → API):**
- **What:** Queued pipeline start, pipeline results, temp wallet cleanup, etc.
- **Auth:** `X-Api-Key` with `CREDIMI_INTERNAL_ADMIN_KEY` — `pkg/workflowengine/activities/internal_http.go`, `queued_pipeline.go`
- **Middleware:** `RequireInternalAdminAPIKey` on internal route groups (`pkg/internal/apis/`)

## Data Storage

**Databases:**

| Store | Technology | Location / connection |
|-------|------------|------------------------|
| Application data | SQLite (PocketBase) | `pb_data/data.db` (env `DATA_DB_PATH` in Docker) |
| Temporal persistence (Compose) | PostgreSQL 16 | Service `postgresql` in `docker-compose.yaml` |
| Temporal visibility (Compose) | Elasticsearch 7.x | Service `elasticsearch`; password `CREDIMI_ELASTIC_PASSWORD` |
| Temporal dev (standalone) | SQLite file | `pb_data/temporal.db` (documented in `AGENTS.md` for `temporal server start-dev`) |

**ORM / client:** PocketBase `dbx` / record APIs — no separate ORM layer; migrations in `pb_migrations/`

**File storage:**
- PocketBase local filesystem under `pb_data/` (uploads, assets)
- Pipeline/run artifacts may be stored on mobile runners and referenced by URL in workflow results

**Caching:**
- Not detected as a dedicated cache layer (no Redis/Memcached in dependencies or Compose)

## Authentication & Identity

**Auth provider:** PocketBase built-in auth (email/password, OAuth2)

**Patterns:**
- **User/API routes:** `Authorization` bearer token or `X-Api-Key` / `Credimi-Api-Key` header (`AGENTS.md`, OpenAPI generation in `pkg/generate_client/generate_client.go`)
- **OAuth2 signups:** Bypass Turnstile via `isOAuth2RecordCreateRequest` in `pkg/internal/apis/turnstile.go`; OAuth UI in `webapp/src/modules/auth/oauth/oauth.svelte`
- **Feature flag:** `oauth` seeded in `pb_migrations/1685000002_seed_features.js`
- **Internal/Temporal routes:** `CREDIMI_INTERNAL_ADMIN_KEY` only
- **Tenancy:** Organizations map to Temporal namespaces (`organizations.canonified_name`) — `pkg/internal/pb/namespaces.go`

**JWT:** `github.com/golang-jwt/jwt/v5` in go.mod (used where PocketBase/JWT flows require it)

## Monitoring & Observability

**Metrics:**
- Prometheus — optional service in `docker-compose.override.yml`, config `deployment/prometheus/config.yml`

**Dashboards:**
- Grafana — custom image `deployment/grafana/`, provisioned Temporal/SDK dashboards (`deployment/grafana/dashboards/`)
- **Proxy:** `ADDRESS_GRAFANA` env in Compose (`http://grafana:8085`); app may expose monitoring under subpath (`GF_SERVER_ROOT_URL` includes `/monitoring`)

**Logs:**
- Standard Go `log` / PocketBase logging; Turnstile failures logged in `pkg/internal/apis/turnstile.go`
- Temporal server `LOG_LEVEL=debug` in Compose

**Error tracking:**
- No Sentry/Datadog SDK detected in dependencies
- CRE error codes: `pkg/internal/errorcodes/errorcodes.go`; Temporal `ApplicationError` wrapping in `pkg/workflowengine/`

## CI/CD & Deployment

**Hosting:**
- Coolify — deploy triggered on `VERSION` bump to `main` (`.github/workflows/release.yml`)

**CI pipelines (GitHub Actions):**
| Workflow | Path | Role |
|----------|------|------|
| `go.yml` | `.github/workflows/go.yml` | Lint, govulncheck, unit tests |
| `release.yml` | `.github/workflows/release.yml` | Semantic release, GoReleaser, Coolify webhook, cosign |
| `webapp.yml` | `.github/workflows/webapp.yml` | Currently commented out |
| `docs.credimi.io.yaml` | `.github/workflows/docs.credimi.io.yaml` | Docs site |
| `pr-conventional-title.yml` | | PR title lint |
| `dependabot.yaml` | `.github/dependabot.yaml` | Dependency updates |

**Container registry:**
- Docker image built from root `Dockerfile`; GHCR publish workflow referenced but commented in `release.yml`

**Secrets (CI — names only):**
- `CREDIMI_EXTRA_PAT` — Private Go module access
- `COOLIFY_CREDIMI_WEBHOOK_URL`, `COOLIFY_WEBHOOK_TOKEN` — Deploy
- `GITHUB_TOKEN` — Releases

## Environment Configuration

**Required env vars (production / full stack):**

| Variable | Required when |
|----------|----------------|
| `CREDIMI_ELASTIC_PASSWORD` | Docker Compose Temporal stack (`docker-compose.yaml`) |
| `CREDIMI_INTERNAL_ADMIN_KEY` | Internal activities and trusted HTTP callbacks |
| `OPENIDNET_TOKEN` | OpenID conformance workflow runs |
| `CREDIMI_EXTRA_PAT` / git URL rewrite | Building with private `credimi-extra` module |
| `PUBLIC_POCKETBASE_URL` | Webapp build (Docker `ARG`) |
| `PUBLIC_TURNSTILE_SITE_KEY` | Registration UI (optional in dev with test keys) |

**Common optional / defaulted:**

| Variable | Default / notes |
|----------|-----------------|
| `TEMPORAL_ADDRESS` | `localhost:7233` |
| `ADDRESS_UI` | `http://localhost:5100` |
| `ROOT_DIR` | `.` or `/app` in container |
| `SMTP_HOST` / `SMTP_PORT` | `smtp.apps.forkbomb.eu` / `1025` |
| `MOBILE_RUNNER_SEMAPHORE_DISABLED` | Semaphore no-op when set |
| `DEBUG` | Verbose routing when set (`pkg/routes/routes.go`) |

**Secrets location:**
- Local: `.env` at repo root (gitignored; template `.env.example`)
- Webapp: may use `webapp/.env` copied from example via `make dev` → `$(WEBENV)` target
- CI/CD: GitHub Actions secrets
- Never commit `.env`, keys, or PATs

## Webhooks & Callbacks

**Incoming:**
- Coolify deploy webhook — GET from GitHub Actions release job (`.github/workflows/release.yml`)
- PocketBase hooks — e.g. Turnstile on `users` create (`pkg/internal/apis/turnstile.go`)
- REST API — extensive handler surface under `pkg/internal/apis/handlers/` registered via `pkg/internal/apis/RoutesRegistry.go`
- Internal unauthenticated routes — mobile runner lookup, some Temporal-adjacent endpoints per `AGENTS.md`

**Outgoing:**
- Cloudflare Turnstile siteverify
- SMTP delivery (workflow mail activity)
- HTTP to mobile `runner_url` endpoints (via credimi-extra)
- Internal credimi API calls from Temporal activities (`internal_http.go`, queued pipeline start)
- OpenIDNet / conformance HTTP (workflow activities with `OPENIDNET_TOKEN`)
- GitHub API (PR comments via GitHub App JWT)
- Docker pull/run (workflow Docker activity)
- Optional: remote APK URL fetch for CI wallet runs (`POST /api/pipeline/run-wallet-apk` per `AGENTS.md`)

## API Documentation & Clients

**OpenAPI:**
- Generated spec: `docs/public/API/openapi.yml` via `pkg/generate_client/generate_client.go`
- Webapp client codegen: `webapp/src/lib/credimi-client/generate.client.ts` (`generate:client` script)

**Public webapp → backend:**
- PocketBase JS SDK — `webapp/src/modules/pocketbase/index.ts` (`PUBLIC_POCKETBASE_URL`)
- Custom REST client for `/api/*` routes — generated credimi client

---

*Integration audit: 2026-05-27*
