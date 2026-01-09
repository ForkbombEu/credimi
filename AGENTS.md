<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# AGENTS.md - LLM Guide to Credimi

## Project Overview

**Credimi** is a compliance checker platform for decentralized identity (SSI) ecosystems. It validates credential issuers and verifiers against industry standards, provides debugging tools, and enables periodic compliance checks.

**Core Technologies:**
- **Backend**: Go 1.25.3 + PocketBase (embedded SQLite)
- **Frontend**: SvelteKit 5 + TypeScript + Vite + Bun runtime
- **Workflow Engine**: Temporal.io for orchestration
- **Cryptography**: Zenroom (zencode execution)
- **Testing**: testify (Go), Vitest + Playwright (frontend)

---

## Architecture

### 1. Backend (Go + PocketBase)

**Entry Point**: `main.go` → `cmd/credimi.go` → `cmd.Start()`

#### Module Structure (`pkg/`)

- **`routes/`**: HTTP routing + reverse proxy setup
  - Binds PocketBase hooks (`OnServe`, `OnRecordCreate`, etc.)
  - Proxies frontend requests to UI server (default: `localhost:5100`)
  
- **`workflowengine/`**: Temporal.io workflow orchestration
  - `workflow.go`: Workflow definitions and input/output structures
  - `activity.go`: Temporal activities (units of work)
  - `workflows/`: Domain-specific workflows (verifier, credentials, mobile)
  - `activities/`: Reusable activity implementations
  - `hooks/`: PocketBase event hooks integration
  - `registry/`: Workflow and activity registration
  - `pipeline/`: Pipeline execution logic
  
- **`credential_issuer/`**: SSI credential issuance workflows
  - `workflow/`: Credential issuer workflow definitions and activities
  
- **`templateengine/`**: Dynamic template rendering
  - Supports EUDIW, EWC, OpenIDNet templates
  - Uses `go-sprout` for template functions
  - Generates schemas and configuration fields dynamically
  
- **`utils/`**: Cross-cutting utilities
  - `error.go`: Custom error types (`CredimiError`)
  - `env.go`: Environment variable helpers
  - `url.go`: URL manipulation utilities
  
- **`internal/`**: Private internal packages (non-importable outside `credimi`)
  - `apis/handlers/`: HTTP endpoint handlers
  - `temporalclient/`: Temporal client initialization
  - `pb/`: PocketBase collection helpers
  - `apierror/`: API error response formatting
  - `errorcodes/`: Standard error code constants
  - `middlewares/`: HTTP middleware (auth, logging, etc.)
  - `routing/`: Advanced routing utilities
  - `canonify/`: Data normalization utilities
  - `logo/`: Logo rendering utilities

#### Key Dependencies

- `pocketbase/pocketbase`: Embedded backend framework (auth, CRUD, realtime)
- `go.temporal.io/sdk`: Workflow orchestration
- `hypersequent/zen`: Zenroom zencode execution
- `spf13/cobra`: CLI framework
- `docker/docker` + `docker/go-connections`: Docker integration for sandboxed execution
- `go-playground/validator`: Input validation
- `invopop/jsonschema` + `santhosh-tekuri/jsonschema`: JSON Schema validation

#### Build & Run

```bash
make dev          # Runs API + UI + Temporal via hivemind
make test         # Go unit tests (-tags=unit)
make lint         # golangci-lint + govulncheck
go run main.go    # Starts PocketBase on :8090
```

---

### 2. Frontend (SvelteKit + TypeScript)

**Location**: `webapp/`

#### Structure

- **`src/routes/`**: SvelteKit pages (file-based routing)
- **`src/lib/`**: Shared libraries
  - `codegen/`: Type generation from JSON schemas
  - `components/`: Reusable UI components
  - `content/`: Content management utilities
  - `credentials/`: Credential handling logic
  - `layout/`: Layout components
  - `marketplace/`: Marketplace features
  - `pages/`: Page-specific components
  - `pipeline-form/`: Pipeline configuration forms
  - `standards/`: Standards compliance utilities
  - `start-checks-form/`: Compliance check forms
  - `temporal/`: Temporal.io frontend integration
  - `types/`: TypeScript type definitions
  - `utils/`: Frontend utilities
  - `wallet-test-pages/`: Wallet testing utilities
  - `workflows/`: Workflow management UI

- **`src/modules/`**: Feature modules (domain-driven)
  - `auth/`: Authentication logic
  - `brand/`: Branding utilities
  - `collections-components/`: PocketBase collection components
  - `components/`: Shared component library
  - `did/`: Decentralized Identifier utilities
  - `features/`: Feature flag management
  - `forms/`: Form handling utilities
  - `i18n/`: Internationalization (Paraglide JS)
  - `keypairoom/`: Keypair management (Zenroom integration)
  - `organizations/`: Multi-tenancy organization logic
  - `pocketbase/`: PocketBase SDK integration
    - `types/`: Auto-generated TypeScript types from DB schema
    - `collections-models/`: Collection model generators
  - `qr/`: QR code generation
  - `utils/`: Module-specific utilities
  - `webauthn/`: WebAuthn biometric authentication

#### Key Dependencies

- `@sveltejs/kit`: Meta-framework (SSR, routing, build)
- `pocketbase`: JavaScript SDK (~0.25.2)
- `zenroom`: Cryptographic VM (Zencode execution)
- `effect`: Functional effect system (async/validation)
- `zod`: Runtime type validation
- `bits-ui` + `formsnap`: Form and UI primitives
- `@inlang/paraglide-js`: i18n library
- `tailwindcss` + plugins: Styling
- `shiki` + `carta-md`: Code highlighting + markdown editor
- `vitest` + `@playwright/test`: Testing

#### Build & Run

```bash
cd webapp
bun install              # Install dependencies
bun run dev              # Vite dev server on :5173
bun run build            # Production build → ./build/
bun run check            # SvelteKit typecheck
bun run lint             # Prettier + ESLint
bun run test:unit        # Vitest tests
bun run test:e2e         # Playwright E2E tests
```

---

### 3. Database & Migrations

- **Database**: SQLite (embedded via PocketBase)
- **Data Dir**: `pb_data/` (ignored in git, seeded locally)
- **Migrations**: `migrations/` (Go migrations), `pb_migrations/` (PocketBase admin UI migrations)
- **Schemas**: `schemas/` (JSON schemas for validation)
- **Hooks**: `pb_hooks/` (JavaScript hooks executed by PocketBase)

#### Key Collections (inferred from hooks)

- `users`: User accounts
- `organizations`: Multi-tenant organizations
- `organizations_invites`: Org invitation system
- `organizations_request_membership`: Membership requests
- `organizations_authorizations`: Role-based access control
- `credential_issuers`: Credential issuer registry
- `features`: Feature flag management
- `oauth`: OAuth configuration

---

### 4. Workflow Engine (Temporal.io)

**Purpose**: Orchestrate long-running, fault-tolerant compliance checks and credential workflows.

**Components**:
- **Workflows**: Durable, versioned business logic (Go functions)
- **Activities**: Stateless, retriable tasks (HTTP calls, DB queries, cryptographic operations)
- **Workers**: Processes that poll Temporal and execute workflows/activities
- **Schedules**: Periodic compliance checks (e.g., daily, weekly)

**Local Dev**: Temporal server runs in Procfile.dev on port `7233` (embedded SQLite: `pb_data/temporal.db`)

**Registration**: `pkg/workflowengine/registry/` defines workflow names and activity mappings.

---

### 5. Template Engine

**Purpose**: Generate dynamic schemas and configuration forms for different SSI standards.

**Supported Standards**:
- **EUDIW**: EU Digital Identity Wallet
- **EWC**: European Wallet Consortium
- **OpenIDNet**: OpenID for Verifiable Credentials

**Templates**: `pkg/templateengine/*_template.go` files define field mappings, validators, and schema generators.

---

### 6. Testing Strategy

#### Backend (Go)

- **Unit Tests**: `-tags=unit` (mocked dependencies)
- **Table-Driven**: Use `testify/require` and `testify/assert`
- **Run**: `make test` or `make test.p TestName` (watch mode)

#### Frontend (SvelteKit)

- **Unit**: Vitest (`bun run test:unit`)
- **E2E**: Playwright (`bun run test:e2e`)
- **Typecheck**: `bun run check` (svelte-check)

---

### 7. Code Style

#### Go

- **Formatting**: gofumpt, gofmt, gci (import ordering), golines (line wrapping)
- **Linting**: golangci-lint, govulncheck, govet
- **Errors**: Wrap with `fmt.Errorf("context: %w", err)`, use `CredimiError` for domain errors
- **Imports**: stdlib → third-party → internal
- **Testing**: Table-driven, use `require`/`assert`, mock interfaces
- **Constructors**: Return interfaces, favor dependency injection

#### TypeScript/Svelte

- **Formatting**: Prettier (tabs, single quotes, width 100)
- **Linting**: ESLint (perfectionist for import sorting)
- **Style**: Tailwind CSS (sorted by tailwindcss-plugin)
- **Validation**: Zod + Effect for async flows
- **Types**: TypeScript-first, prefer type aliases for unions

---

### 8. Development Workflow

#### Local Development

```bash
make dev                # Start all services (API + UI + Temporal)
make kill-pocketbase    # Kill stale PocketBase instances
make purge              # Wipe database (reset to clean state)
make seed               # Seed database with fixtures
```

#### Testing

```bash
make test               # Backend tests
cd webapp && bun run test:unit    # Frontend unit tests
cd webapp && bun run test:e2e     # Frontend E2E tests
```

#### Linting & Formatting

```bash
make lint               # Backend lint
make fmt                # Backend format
cd webapp && bun run lint         # Frontend lint + format check
cd webapp && bun run format       # Frontend auto-format
```

---

### 9. Deployment

- **Docker**: `docker-compose.yaml` orchestrates services
- **Build**: `make docker` builds production image
- **Binary**: `make credimi` produces standalone executable
- **UI Binary**: `make credimi-ui` produces Bun-compiled frontend binary

---

### 10. Documentation

- **Location**: `docs/`
- **Format**: Markdown (VitePress)
- **Serve**: `cd docs && bun i && bun run docs:dev --host`
- **Sections**:
  - `Manual/`: User guides (compliance checks, marketplace)
  - `Software_Architecture/`: Architecture docs (dev setup, building blocks, implementation)
  - `Legal/`: Terms & privacy policy

---

## Key Conventions

### Naming

- **Go**: PascalCase (exported), camelCase (private), descriptive full words
- **TypeScript**: camelCase (variables), PascalCase (types/components), kebab-case (files)
- **Avoid**: 1-2 letter identifiers (except idiomatic `i`, `err`, `ctx`)

### Error Handling

- **Go**: Return errors, don't swallow; use `CredimiError` for domain errors
- **TypeScript**: Use `Effect` for async error handling, avoid throwing

### Control Flow

- **Early Returns**: Handle edge cases first, reduce nesting
- **Validation**: Input validation at API boundaries (go-playground/validator, zod)

### Comments

- **Explain "Why"**: Not "how" (self-explanatory code)
- **Godoc**: Public APIs must have godoc comments
- **JSDoc**: Optional, prefer TypeScript types

---

## Common Tasks

### Add New Workflow

1. Define workflow in `pkg/workflowengine/workflows/`
2. Define activities in `pkg/workflowengine/activities/`
3. Register in `pkg/workflowengine/registry/registry.go`
4. Add UI form in `webapp/src/lib/workflows/`

### Add New PocketBase Collection

1. Create migration in `migrations/` (Go) or use PocketBase admin UI
2. Add hooks in `pb_hooks/` (JavaScript)
3. Regenerate types: `cd webapp && bun run generate:types`

### Add New Compliance Standard

1. Create template in `pkg/templateengine/{standard}_template.go`
2. Add tests in `pkg/templateengine/{standard}_template_test.go`
3. Register in `pkg/templateengine/templates.go`
4. Add UI schema form generator

---

## Troubleshooting

### Backend

- **Port 8090 busy**: `make kill-pocketbase`
- **Migration errors**: `make purge` then `make seed`
- **Temporal not starting**: Check port `7233`, ensure `temporal` CLI installed

### Frontend

- **Type errors after DB change**: `cd webapp && bun run generate:types`
- **Missing env vars**: Copy `webapp/.env.example` to `webapp/.env`
- **Module not found**: `cd webapp && bun install`

---

## References

- **PocketBase Docs**: https://pocketbase.io/docs/
- **Temporal Docs**: https://docs.temporal.io/
- **SvelteKit Docs**: https://kit.svelte.dev/docs/
- **Zenroom Docs**: https://dev.zenroom.org/

---

## Notes for LLMs

- **Modular Architecture**: Backend and frontend are decoupled (reverse proxy in production)
- **Type Safety**: Auto-generated types (`webapp/src/modules/pocketbase/types/`) must not be manually edited
- **Workflow-First**: Complex operations are Temporal workflows (resumable, fault-tolerant)
- **Standards-Driven**: Templates enforce compliance with SSI standards (EUDIW, EWC, OpenIDNet)
- **Multi-Tenancy**: Organizations provide isolation; authorization via `organizations_authorizations`
- **Feature Flags**: `features` collection controls gradual rollouts
- **Cryptography**: Zenroom executes zero-knowledge proofs and cryptographic contracts
- **Real-Time**: PocketBase provides WebSocket-based realtime subscriptions

---

**Last Updated**: 2025-01-09
