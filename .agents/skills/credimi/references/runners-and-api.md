# Runners, devices, auth, API and webapp surface

Verified against source. Where `AGENTS.md` or the manual pages disagree with the code, the disagreement is stated explicitly — trust the code.

## Runner and device model

- Collections `mobile_runners` and `mobile_devices`.
- `mobile_runners`: `owner`, `name`, `description`, `type`, `ip` (required), `port`, `serial`, `published`, `canonified_name`, `online`, `last_heartbeat_at`, `admin_managed`, `disabled`. The runner URL is `ip` plus `:port` when set.
- `mobile_devices`: `owner`, `runner`, `name`, `canonified_name`, `description`, `type` (select), `serial`, `online`, `live_stream`.
- Device types: `android_emulator`, `ios_simulator`, `android_phone`, `redroid` (plus `ios_phone` in pipeline normalization).
- Per-runner limits: **at most one `android_emulator` and one `ios_simulator` per runner**; `android_phone`/`redroid` need unique serials per runner. There is no enforced global "one runner per device" rule — that phrasing is product guidance only.
- Device identity `(owner, runner, name)` is immutable; renaming/moving is rejected with `device_identity_immutable`, and a device's owner must equal its runner's owner.

Flags:

- `disabled` — administratively hidden from every catalog surface and from worker starts.
- `online` — durable liveness, driven by heartbeats (runner posts every 30s) and a background monitor.
- `published` — visible to other orgs; non-published callers see `owner = me OR (published AND admin_managed)`, published callers see `owner = me OR published`.
- `admin_managed` — set when a superuser creates the runner; such runners serve **every** namespace, non-admin runners only published orgs.

Liveness env vars: `MOBILE_RUNNER_SELECTOR_HEARTBEAT_TTL` (default `60s`, governs whether a runner counts as recently alive for selectors), `MOBILE_RUNNER_LIFECYCLE_HEARTBEAT_TIMEOUT` (default `1h`, marks stale runners offline and pauses their device semaphores), plus `MOBILE_RUNNER_LIFECYCLE_MONITOR_INTERVAL` (30s) and `MOBILE_RUNNER_LIFECYCLE_SHUTDOWN_AFTER` (7d).

Registration (runner agent → Credimi, usually not the browser):

1. `POST /api/mobile-runner/preview-id` → next canonical runner id.
2. `POST /api/mobile-runner` → create/update the runner.
3. `POST /api/mobile-device/preview-id`, then `POST /api/mobile-device` per device.
4. `POST /api/mobile-device/reconcile` → declares the full inventory; absent devices are deleted.
5. `DELETE /api/mobile-device` → remove one device.

At run time every chosen device's runner must answer `GET {runner_url}/health` with 200, or the run is rejected with `503 device runner is offline`. Catalog surfaces use heartbeat freshness instead of probing; `/api/mobile-runners` does probe (2s timeout, concurrency 8) and reports `offline`/`misconfigured` instead of failing the list.

### Device addressing

Canonical `<org>/<runner>/<device>`: `mobile_runners` path length 2 under `organizations/owner`, `mobile_devices` path length 3 under `mobile_runners/runner`. Validate with `POST /api/canonify/identifier/validate`, resolve with `GET /api/canonify/identifier/get`. Non-superusers can only create devices under runners in their own organization.

## Credimi → runner HTTP contract

All of it goes through the `mobile-runner-http-request` activity (it injects the internal admin credential and resolves `*.trycloudflare.com` via Cloudflare DNS):

| Method | Path                                         | Body                                                                                    | Notes                                                                                                                |
| ------ | -------------------------------------------- | --------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| GET    | `{runner_url}/health`                        | —                                                                                       | liveness probe (called directly by the list/run-path helpers)                                                        |
| POST   | `{runner_url}/credimi/installer-action`      | `{version_identifier, action_identifier, platform, device_identifier, skip_installer?}` | expected 200, 300s timeout; returns the action code                                                                  |
| POST   | `{runner_url}/credimi/pipeline-result`       | `{video_path, last_frame_path, run_identifier, device_identifier, platform, log_path?}` | expected 200, 300s; returns `result_video_urls` / `screenshot_urls`                                                  |
| POST   | `{runner_url}/credimi/execution-screenshots` | `{run_identifier, device_identifier, step_id, screenshot_paths[]}`                      | expected 200, 300s; returns `screenshot_urls[]`                                                                      |
| POST   | `{runner_url}/credimi/live-view`             | `{device_identifier, serial, namespace, workflow_id, run_id}`                           | **called directly**, not via the activity; header `Credimi-Api-Key`; expected `{path: "/live/<token>"}`, 15s timeout |
| POST   | `{runner_url}/worker/{namespace}`            | `{old_namespace}`                                                                       | worker-manager start; expected **202** (idempotent — "already running" is fine)                                      |

Runner URLs for the worker manager come from `GET /api/mobile-runner/list-urls`, which only returns runners eligible for worker start. Device/emulator activities (`ListInstalledApps`, `StartRecording`, `StopRecording`) live in the closed `credimi-extra` module, so their concrete runner paths are not visible here.

**Doc drift:** `AGENTS.md` documents `POST {runner_url}/fetch-apk-and-action` and `/store-pipeline-result`; the code calls `/credimi/installer-action` and `/credimi/pipeline-result`.

## Live view

`POST /api/pipeline/live-view` `{workflow_id, run_id, device_id?}` (user auth):

- Only for a **running** dynamic-pipeline execution, and only after the mobile setup has populated its device map, else a not-ready error; `ios_simulator` is rejected with `422 live view is not supported for ios_simulator devices`.
- Response `{streams: [{device_id, device_name, url}]}` where `url` is `{runner_url}/live/<token>`.
- Error mapping: transport failure → `503 device runner is offline`; runner `503` → `503 live view is unavailable on the device runner`; runner `400` → `422 runner refused live view`; runner `401`/`403` → `502` with "Check that both use the same internal admin key."; other non-2xx → `502` forwarding the runner's `message`; a missing or malformed `/live/` path → `502 invalid live view address`.
- Requires `CREDIMI_INTERNAL_ADMIN_KEY` on both sides.

## Auth

- The API-key header is **`Credimi-Api-Key`** (not `X-Api-Key`, which several docs still say).
- Keys are bcrypt hashes in the `api_keys` collection with `key_type` = `user` or `internal_admin`, plus `revoked` / `expires_at`.
- Generate: `POST /api/apikey/generate` `{name}` (logged-in users on the `users` or `_superusers` collection); the key is shown once. UI: `/my/profile/api-keys`.
- Authenticate: `GET /api/apikey/authenticate` (user key → PocketBase auth token); `GET /api/apikey/authenticate-internal-admin` (internal key).
- Middlewares: `RequireAuthOrAPIKey` (Bearer token **or** user API key — applied to every route group with `AuthenticationRequired: true`), `RequireInternalAdminAPIKey` (internal-admin scope), `RequireInternalAdminOrAuth` (internal key first, else user auth), `OptionalAuthOrAPIKey` (never blocks).
- `CREDIMI_INTERNAL_ADMIN_KEY` is the runtime internal credential sent to runners as `Credimi-Api-Key`.
- A user-scoped key hitting an internal-admin route gets `403 insufficient_api_key_scope`.

## Endpoint catalog

`U` = user auth or user API key, `P` = public, `I` = internal admin key.

**Custom integrations.** `POST /api/custom-integrations/run` (U).

**API keys.** `POST /api/apikey/generate` (needs `e.Auth`), `GET /api/apikey/authenticate` (reads the key header), `GET /api/apikey/authenticate-internal-admin` (I).

**Canonify.** `POST /api/canonify/identifier/validate`, `GET /api/canonify/identifier/get` (P).

**Templates / organizations.** `POST /api/clone-record` (P), `POST /api/template/placeholders` (U), `GET /api/organizations/my`, `GET /api/organizations/visible-namespaces` (U), `GET /api/organizations/namespaces` (I).

**Catalog.** `POST /api/conformance-catalog/rebuild` (I); `GET /api/collections/conformance_checks/records[/{id}]` and `GET /api/collections/conformance_suites/records[/{id}]` (P, writes rejected).

**Conformance runs.** `GET /api/conformance-check/deeplink` (P); `POST /api/compliance/{protocol}/{version}/save-variables-and-start`, `POST /api/compliance/send-temporal-signal`, `POST /api/compliance/send-{openidnet,eudiw,ewc}-log-update`, `GET /api/compliance/deeplink/{workflowId}/{runId}` (U).

**Deeplinks.** `POST /api/get-deeplink`, `GET /api/credential/deeplink`, `GET /api/verification/deeplink` (P).

**Issuers / credentials.** `POST /api/credentials_issuers/start-check`, `POST /api/credentials_issuers/import-fides` (U); `POST /api/credentials_issuers/store-or-update`, `…/store-or-update-extracted-credentials` (I); `GET /api/credential/get-credential-offer`, `DELETE /api/credential/temp/{record}` (I).

**Verifiers.** `GET /api/verifier/get-use-case-verification-deeplink`, `DELETE /api/verifier/temp-use-case/{record}` (I).

**Wallets.** `POST /api/wallet/start-check` (U); `POST /api/wallet/get-installer-md5-or-etag` (I); `POST /api/wallet/store-pipeline-result` (internal-or-auth); `DELETE /api/wallet/temp-version/{record}` (I).

**Pipelines (user).** `POST /api/pipeline/queue`, `POST /api/pipeline/run-wallet-apk`, `/run-issuer`, `/run-verifier`, `GET|DELETE /api/pipeline/queue/{ticket}`, `GET /api/pipeline/list-executions[/{id}]`, `GET /api/pipeline/executions/{id}/{workflow_id}/{run_id}`, `POST /api/pipeline/live-view` (U); `POST /api/pipeline/execute` (group auth excluded; effectively public, `http-request` steps only).

**Pipelines (internal).** `/api/pipeline/store-step-screenshots`, `/get-yaml`, `/mobile-flow`, `/pipeline-execution-results` (+ `/evidence`, `/report`, `/fcaf-report`), `/scoreboard/{namespace}`, `/scoreboard/aggregate/start`, `/scoreboard/aggregate/schedule/{schedule_id}`, `/execution-details/{namespace}/{workflow_id}/{run_id}`, `/scoreboard/save-results`, `/retention/delete-files`, `/retention/schedule` — internal-admin, except `store-step-screenshots` (internal-or-auth) and `mobile-flow`, which currently has **no auth middleware** (flagged as a likely oversight).

**Workflows.** `GET /api/my/workflows/{workflowId}/runs[/{runId}]` (+ `/history`, `/export`, `/logs?action=start|stop`), `POST …/rerun`, `…/cancel`, `…/terminate`, `GET /api/list-workflows` (U; non-pipeline trees only).

**Schedules.** `POST /api/my/schedules/start`, `GET /api/my/schedules`, `POST /api/my/schedules/{scheduleId}/{cancel,pause,resume}` (U).

**Runners / devices.** `GET /api/mobile-runners`, `GET /api/mobile-devices` (internal-or-auth); `GET /api/mobile-runner/list-urls`, `GET /api/mobile-device`, `GET /api/mobile-device/semaphore`, `POST /api/mobile-device/validate-access` (I); registration and lifecycle routes (`/api/mobile-runner[...]`, `/api/mobile-device[...]`, `…/lifecycle/{resume,heartbeat,pause}`) are internal-or-auth.

**Web push.** `GET /api/web-push/vapid-public-key` (P), `POST /api/web-push/pipeline-completed` (I).

**UI proxy.** Any other path is reverse-proxied by PocketBase to `ADDRESS_UI` (default `http://localhost:5100`), preserving `Origin`/`Referer`.

Not registered despite being declared: the scoreboard HTTP routes `/api/my/results` and `/api/all-results` (commented out in `RoutesRegistry.go`); the UI reads the `pipeline_scoreboard_cache` collection instead.

## Webapp surface

**Public.** `/` (landing), `/hub`, `/hub/[...path]`, `/hub/conformance-checks/[...path]`, `/scoreboard`, `/news`, `/organizations`, `/pages/[...slug]`, `/pipeline-report`, `/tests/wallet/{eudiw,ewc,openidnet}`. `/marketplace*` only redirects to `/hub*`.

**Auth.** `/login`, `/login/webauthn`, `/register`, `/forgot-password`, `/reset-password-[token]`, `/verify-email-[token]`, `/logout`, `/organization-invite-…`, `/keypairoom[/regenerate]`.

**Dashboard — `/my/*`.** `/my` (dashboard home), `/my/wallets`, `/my/credential-issuers-and-credentials`, `/my/verifiers-and-use-case-verifications`, `/my/custom-integrations` (+ `new`, `[...path]/edit`, `[...path]/run`), `/my/pipelines` (+ `new`, `[...path]/edit`, `[...pipeline_path]`, `schedule`), `/my/tests/new`, `/my/tests/runs` (+ `[workflow_id]`, `/[run_id]`, `/[run_id]/temporal`), `/my/did`, `/my/organizations` (+ `create`, `join`, `[id]`, `/[id]/members`, `/[id]/settings`), `/my/organization`, `/my/profile` (+ `api-keys`, `public-keys`, `webauthn`), `/my/notifications` (link present but disabled).

There is no top-level `/dashboard`, `/executions` or `/workflows` route: "dashboard" is `/my`, "executions" is `/my/tests/runs`, "scoreboard" is `/scoreboard`, "settings" is `/my/organizations/[id]/settings` plus `/my/profile`. `/ui-tests/*` are dev-only pages.

Feature flags can hide sidebar sections (`ORGANIZATIONS`, `DID`).

## Doc drift to ignore

1. Runner HTTP endpoints: `AGENTS.md` says `fetch-apk-and-action` / `store-pipeline-result`; the code calls `/credimi/installer-action` / `/credimi/pipeline-result`.
2. The internal-admin header is `Credimi-Api-Key`, not `X-Api-Key` (wrong in `AGENTS.md`, `docs/.../scoreboard.md`, and a comment in `pkg/conformancecatalog/store.go`).
3. `POST /api/pipeline/start` does not exist in this worktree; the GUI and CLI both use `POST /api/pipeline/queue`.
4. The two legacy scoreboard routes are not registered.
