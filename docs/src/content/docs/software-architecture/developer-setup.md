---
title: "🤖 Developer Setup"
description: ""
sidebar:
  order: 7
---
## **Prerequisites**

Before you begin, ensure you have the following tools installed:

-   [Git](https://git-scm.com/downloads)
-   [Make](https://www.gnu.org/software/make)
-   [Mise](https://mise.jdx.dev/getting-started.html)
-   [Temporal](https://docs.temporal.io/cli)
-   [Tmux](https://github.com/tmux/tmux/wiki/Installing)
-   [Pre-commit](https://pre-commit.com/)
-   [Golang](https://go.dev/doc/install)

### **Install `slangroom-exec`**

Download the appropriate `slangroom-exec` binary for your OS from the [releases page](https://github.com/dyne/slangroom-exec/releases).

Add `slangroom-exec` to PATH and make it executable:

```bash
wget https://github.com/dyne/slangroom-exec/releases/latest/download/slangroom-exec-Linux-x86_64 -O slangroom-exec
chmod +x slangroom-exec
sudo cp slangroom-exec /usr/local/bin/
```
### **Install `Pre-commit`**
For most Linux distrubution, just do: 
```bash
sudo apt install pre-commit
```


### **Install `slangroom-exec`**

## **Setup Workspace**

### **Clone the repository**

```bash
git clone https://github.com/ForkbombEu/credimi
```

### **Install dependencies**

```bash
cd credimi
mise trust
make credimi
```

## Edit your .env file 

Copy .env.example to .env

```bash
cp ./webapp/.env.example ./webapp/.env  
```

Then edit the .env file, particularly: 

1. set the absolute path in ROOT_DIR
1. Get a token from https://www.certification.openid.net and add it in TOKEN

Copy ./webapp/env.example to 

## **Start Development Server**

```bash
make dev
```

For a faster API/UI boot that skips Temporal worker registration (pipelines and workflows will not run):

```bash
make dev.noworkers
```

This sets `CREDIMI_TEMPORAL_WORKERS_DISABLED=1`. Temporal Docker still starts because the runtime Procfile waits on the Temporal port (classic `:7233`).

> [!TIP]
> Use `make help` to see all the commands available.

## Parallel worktrees

Parallel Credimi checkouts require [Worktrunk](https://worktrunk.dev/) for bootstrap (copy-ignored + unique ports). Classic ports stay centralized in `scripts/dev-ports.env` (`8090` / `5100` / `7233` / `8280` / `8281`). Other worktrees override them with a gitignored `.env.worktree`; worktrees created before the embedded Temporal UI port (`TEMPORAL_UI_EMBEDDED_PORT`, classic `8281`) fall back to the classic value, so add a unique one to `.env.worktree` when running several stacks at once.

Run pages embed the Temporal UI from `/temporal-ui`, served by PocketBase through an organization-scoped, read-only proxy to the `temporal_ui_embedded` container. In dev, Vite forwards `/temporal-ui` to PocketBase, so the iframe stays on the webapp origin and receives the `pb_auth` session cookie.

Checkout **location** is per user, not per repo. Do not commit a Worktrunk `worktree-path`. Set it in `~/.config/worktrunk/config.toml` if you use `wt switch` (see [Worktrunk config](https://worktrunk.dev/config/)). Cursor agent sandboxes usually live under `~/.cursor/worktrees/` and do not need that setting.

Install Worktrunk once (needed for both flows below):

```bash
brew install worktrunk && wt config shell install
```

### Disposable Cursor sandboxes

Intended flow: open an agent worktree, run `make dev`, check results, dispose.

`.cursor/worktrees.json` runs `make worktree-bootstrap` when Cursor creates a worktree (`/worktree`, Agents Window, or CLI). Then in that checkout:

```bash
make dev
```

When finished, stop Compose with `make worktree-down`. To remove a disposable worktree and its containers, volumes, copied data, and branch from the primary checkout, run `make worktree-destroy WORKTREE=<branch>`. Language servers are often off by default under `~/.cursor/worktrees/`; turn on “Enable LSPs for Worktrees” if you need full IDE features while checking.

### Worktrunk CLI worktrees

```bash
wt switch -c feat/my-thing
make dev
```

`.config/wt.toml` runs the same bootstrap on pre-start and `make purge` on removal, so Worktrunk removals discard that worktree's containers, volumes, copied data, and generated runtime files.

Bootstrap (also callable by hand) uses `wt step copy-ignored --require-include` for `.worktreeinclude` (`.env`, `webapp/.env`, `webapp/node_modules/`, `pb_data/`), writes unique ports into `.env.worktree` (Worktrunk `hash_port` seeds + collision walk), syncs PocketBase URLs in `webapp/.env`, rewrites checkout `.env` `ROOT_DIR` to this worktree’s absolute path (do not leave the primary’s path after copy-ignored), runs non-interactive `mise trust --yes` for this checkout’s `.mise.toml` (so `wt switch` / Cursor bootstrap is not blocked on a trust prompt), initializes submodules, and runs `make tools` when `.bin` is missing. `make dev` also exports that same `ROOT_DIR` so the API process does not keep a stale dotenv value.

Edit `.env.worktree` to change ports; regenerators never overwrite an existing file.

Prefer stopping PocketBase in the source worktree before copying `pb_data/` so SQLite is quiet during the copy.

Primary checkout does not need Worktrunk for `make dev` (classic ports). Worktrunk is required for additional parallel worktrees.

## Temporal server versions

The Compose stack runs `temporalio/server`, which does not create or upgrade its own schemas. The `temporal_schema` job in `docker-compose.yaml` does it on every start with the `temporalio/admin-tools` image of the same version, before the server starts. Versions are pinned in `scripts/dev-compose.env`.

Temporal upgrades existing data only one minor version at a time. Dev Temporal data is disposable: after a version bump, run `make purge` and then `make dev`.

## Temporal Visibility Search Attributes

Pipeline listings and filters rely on custom Temporal visibility search attributes. Credimi registers them
itself on every namespace it sets up (`hooks.EnsureNamespaceReady`, at server start and on organization
create/rename), so no manual `temporal operator search-attribute create` step is needed:

| Name | Type |
| --- | --- |
| `PipelineIdentifier` | `Keyword` |
| `DeviceIdentifiers` | `KeywordList` |
| `ActionsID` | `KeywordList` |
| `VersionsID` | `KeywordList` |
| `CredentialsID` | `KeywordList` |
| `UseCaseID` | `KeywordList` |
| `ConformanceCheckID` | `KeywordList` |
| `CustomCheckID` | `KeywordList` |

The list lives in `workflowengine.CustomSearchAttributeTypes`. If an attribute is added after workflows already
exist, trigger a Temporal visibility reindex to backfill historical data (see Temporal admin tooling docs for
your deployment).

## mobile device semaphore Ops (Internal)

### Defaults and knobs

- Default acquire wait timeout: 45m.
- Override timeout: `MOBILE_DEVICE_SEMAPHORE_WAIT_TIMEOUT=30m` (or any valid `time.ParseDuration` value).
- Disable semaphore (no-op acquire/release): `MOBILE_DEVICE_SEMAPHORE_DISABLED=1`.
- Internal admin key: an `internal_admin` key in `api_keys`. The Credimi server needs no env var for it; admin-managed runners and operators hold its plaintext (conventionally `CREDIMI_INTERNAL_ADMIN_KEY` in their own environment) to call the runner/operator routes guarded by it. Credimi never sends it to runners.
- Runner credential secret: `CREDIMI_RUNNER_CREDENTIAL_SECRET=<random secret>` (required in deployments, e.g. `openssl rand -hex 32`; `make dev` sets a dev default). Per-runner credentials sent to runners as `Credimi-Api-Key` are derived from it and delivered to each runner on registration and lifecycle responses; rotating it invalidates runner credentials until each runner's next registration or heartbeat.

### Internal admin API key rollout

1. Run DB migrations so `api_keys` supports `key_type`, `superuser`, `revoked`, `expires_at`.
2. Provision an internal admin key (hash stored in DB, plaintext returned once).
3. Set `CREDIMI_RUNNER_CREDENTIAL_SECRET` in the backend runtime secrets, and the internal admin key's plaintext as `CREDIMI_INTERNAL_ADMIN_KEY` on admin-managed mobile runners.
4. Deploy the backend with the internal admin key middleware enabled.
5. Smoke-check one authenticated user route and one runner/operator route guarded by the internal admin key.

### Emergency procedures

- Semaphore workflows live in the Temporal `default` namespace with IDs: `mobile-device-semaphore/<device_id>`.
- Query current state via Temporal UI (`GetState` query).
- To unstick a runner, terminate the semaphore workflow in Temporal; it will be recreated on the next acquire.
