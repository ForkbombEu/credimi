# Pipelines — authoring, running, verifying

Everything here is verified against source. `pkg/internal/pipeline/types.go` is the authoritative grammar; the JSON schema is **generated** from it.

## Verify syntax against the latest code

| Purpose                                              | Command / file                                                                                                                                                                                                                                                                                                        |
| ---------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Regenerate the schema (what CI/`make generate` does) | `go run main.go pipeline schema -o schemas/pipeline/pipeline_schema.json` — wired at `pkg/gen.go:17`, run by `go generate ./pkg` / `make generate`                                                                                                                                                                    |
| Print the current schema                             | `credimi pipeline schema` (JSON to stdout); `-o file.yaml` or `-o file.yml` emits YAML; `-o` is the only flag (`cmd/cli/schema.go:26-75`)                                                                                                                                                                             |
| Grammar source of truth                              | `pkg/internal/pipeline/types.go` (`WorkflowDefinition`, `StepSpec`, `StepDefinition`, `StepInputs`, `RuntimeConfig`, `FinallyDefinition`)                                                                                                                                                                             |
| Step `use:` catalog                                  | `pkg/workflowengine/registry/registry.go:31-131` (author-facing) and `:133-197` (internal-only)                                                                                                                                                                                                                       |
| Schema consumers                                     | The **pipeline editor validates YAML with Ajv** against the generated schema: `webapp/src/lib/pipeline/validate-yaml.ts:5-31` (used by the steps builder and inline manual editor). TS types: `bun run generate:schema-types` → `webapp/src/lib/pipeline/types.generated.ts` (`webapp/src/lib/codegen/index.ts:9-18`) |

The published schema has one `oneOf` variant per registry key plus `debug` and `child-pipeline`; the internal-only steps are **not** in it (`cmd/cli/schema.go:91`, `345-352`).

**Runtime is not strict.** `ParseWorkflow` is a plain `yaml.Unmarshal` with no known-fields check (`pkg/internal/pipeline/parser.go:14-21`). An unknown `use:` fails only at execution (`unknown step type: %s`, `pkg/workflowengine/pipeline/utils.go:229-236`), and unknown keys inside `with` are silently ignored. So the schema/editor is the real pre-flight check. Real validators: `ValidateFinallySteps` at workflow start, `ValidateDeviceIDYAML` in the CLI and the mobile setup hook, and route input structs via `routing.GetValidatedInput`.

## Top level

| Key       | Type                 | Notes                                                                                                                     |
| --------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `name`    | string, **required** | Workflow ID `Pipeline-<canonify(name)>-<uuid>` (`pkg/workflowengine/pipeline/pipeline.go:878-882`)                        |
| `version` | string               | Parsed, never read by the engine                                                                                          |
| `runtime` | object               | See below                                                                                                                 |
| `config`  | map                  | Merged into every step's `with.config`; the step's own config wins (`pkg/internal/pipeline/resolver.go:16-25`, `303-337`) |
| `steps`   | list                 | See below                                                                                                                 |
| `finally` | list or map          | See below                                                                                                                 |

Root is `additionalProperties: false` with `required: [name]`.

## Step

| Key                       | Type                 | Notes                                                                                           |
| ------------------------- | -------------------- | ----------------------------------------------------------------------------------------------- |
| `id`                      | string               | Key used in output references                                                                   |
| `use`                     | string               | A registry key (or `debug` / `child-pipeline`)                                                  |
| `with`                    | object               | Payload/config, see merge rule                                                                  |
| `activity_options`        | object               | Merged over the runtime/global activity options (`pkg/workflowengine/pipeline/utils.go:87-142`) |
| `metadata`                | map                  | Free-form                                                                                       |
| `continue_on_error`       | bool (default false) |                                                                                                 |
| `on_error` / `on_success` | list of steps        | Failure/success hooks                                                                           |

Schema step variants require `id`, `use`, `with` and forbid extra keys.

### `with` merge rule

`with.config` stays config. `with.payload` is merged into the payload. **Every other key under `with` is copied into the payload** (`pkg/internal/pipeline/parser.go:72-108`), so `with: {method: GET}` ≡ `with: {payload: {method: GET}}`. `config` must be a map; `payload` may be null.

### `finally`

Two shapes (`pkg/internal/pipeline/parser.go:23-70`): a flat list (all entries become `always`), or a map `{always, on_success, on_failure}`. Anything else errors with `invalid finally definition: expected list or map, got YAML node kind N`.

Only `email` and `http-request` are allowed inside `finally`; anything else fails the whole run before the first step with `finally step '%s' uses '%s' which is not allowed. Only email and http-request are allowed` (`pkg/workflowengine/pipeline/pipeline.go:147-149`, `1154-1169`).

## Outputs and expressions

Context per step: `inputs` (the run's input payload), one key per already-executed step id whose shape is `{outputs: …}`, plus `pipeline_output`, `pipeline_name`, `pipeline_url`, `result` (`success`/`failed`), `date`, `workflow_id`, `run_id`, `organization_id` (`pkg/workflowengine/pipeline/pipeline.go:441-482`, `execute.go:411-438`).

Syntax `${{ … }}` (`pkg/internal/pipeline/resolver.go:39`):

- Whole string is one expression → the resolved value keeps its native type; embedded in text → stringified.
- Dotted and indexed paths: `step.outputs.body.items[0].id`.
- Pipeline functions after `|`: `upper`, `lower`, `url_encode`, `url_decode`, `optional`, `slice(...)`, `replace(old,new)`.
- `| optional` **as the first function** swallows a missing reference and yields `null`; otherwise a missing reference is an error (`ref not found: …`).
- Errors inside an embedded (non-whole-string) expression do not fail the step — they stringify as `ERR(<message>)`.
- Interpolation is deliberately skipped for `rest-chain`'s `yaml` and `conformance-check`'s `config`/`template` (`resolver.go:27-30`, `288-301`).

Separate, older substitution: `${fixture.<key>}` (no `${{ }}`), keys `issuer_url` (default `https://issuer-backend.eudiw.dev`), `verifier_url` (default `https://verifier-backend.eudiw.dev`), `log_checker` (default `eudiw`); override via `runtime.fixture` (`pkg/internal/pipeline/fixture.go:12-48`).

## Step catalog

| `use:`                           | Kind                         | Payload fields (`with.*`)                                                                                                                                                     | Output                                          |
| -------------------------------- | ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| `http-request`                   | activity                     | `method`_, `url`_, `query_params{str:str}`, `timeout` (seconds string), `headers{str:str}`, `body` (any), `expected_status`, `outputs{name:{xpath\|selector\|cookie\|regex}}` | map `{status, headers, body, …output rules}`    |
| `container-run`                  | activity                     | `image`\*, `cmd[]`, `user`, `env[]`, `ports[]`, `mounts[]`, `containerName`, `networkConfig`                                                                                  | map                                             |
| `email`                          | activity                     | `sender`, `recipient`\*, `subject`, `body`, `template`, `data{}`                                                                                                              | string                                          |
| `rest-chain`                     | activity                     | `yaml`, `data{}`, `env`; **`with.config.template` is required** (`pkg/workflowengine/activities/stepci.go:117-128`)                                                           | StepCI JSON map                                 |
| `json-parse`                     | activity                     | `rawJSON`_, `struct_type`_ (only `map` registered)                                                                                                                            | map                                             |
| `jsonschema-validation`          | activity                     | `schema`_, `data`_ (map), `subschema`                                                                                                                                         | map                                             |
| `credential-issuer-validation`   | activity                     | `base_url`\*                                                                                                                                                                  | map                                             |
| `cesr-parse`                     | activity                     | `rawCESR`\*                                                                                                                                                                   | map                                             |
| `cesr-validate`                  | activity                     | `events`\*                                                                                                                                                                    | any                                             |
| `appstore-url-validation`        | activity                     | `url`\*                                                                                                                                                                       | map                                             |
| `fcaf-validation`                | activity                     | `test_id` or `test_ids[]`, `suite`, `catalog_root`, `pipeline_outputs`\* (non-empty), `runtime{}`                                                                             | map `{report}`                                  |
| `conformance-check`              | workflow                     | `check_id`\*, `test`, `parameters{}`                                                                                                                                          | workflow output                                 |
| `custom-check`                   | workflow                     | `yaml` **xor** `check_id`, `parameters{}`                                                                                                                                     | workflow output                                 |
| `credential-offer`               | workflow                     | `credential_id`\*                                                                                                                                                             | offer deeplink string                           |
| `use-case-verification-deeplink` | workflow                     | `use_case_id`\*                                                                                                                                                               | deeplink string                                 |
| `mobile-automation`              | workflow, **needs a runner** | `action_id` or (`action_code` + `version_id`), `parameters{str:str}`, `device_id`                                                                                             | `{test_run_url, result_video_url, flow_output}` |
| `debug`                          | activity                     | —                                                                                                                                                                             | prints current state; non-fatal                 |
| `child-pipeline`                 | —                            | `pipeline_id`, config `app_url`                                                                                                                                               | nested pipeline result                          |

`*` = required by the activity payload's validator tags. Output shaping for activities follows `OutputKind` (`AsMap`/`AsString`/`AsSliceOf*`/`AsBool`; `OutputAny` returns the whole `{output,log,secrets}` result) — child workflows return their output verbatim (`pkg/workflowengine/pipeline/execute.go:119-138`, `216`).

**`mobile-automation` is the only step that requires a runner**, and a pipeline containing one may only be started through the queue/semaphore path — starting it directly errors `mobile-runner pipelines must be started via queue/semaphore` (`pkg/workflowengine/pipeline/mobile_automation_hooks.go:234-252`).

## `mobile-automation` in detail

- Inputs validated at `pkg/workflowengine/pipeline/mobile_automation_hooks.go:619-680`: empty `with.payload` → `missing payload for step %s: expected with.action_id or with.payload.action_id`; with `action_code` → `version_id` required; without it → `action_id` required. IDs are canonified.
- `parameters` is strictly `map[string]string`; a non-string value errors with `with.payload.parameters must contain only string values; parameter %q resolved to %s` (`pkg/workflowengine/pipeline/utils.go:284-300`). Values reach the Maestro flow as env vars.
- Device resolution: step `device_id`, else `runtime.global_device_id`. **Exactly one** must be set — both → `global_device_id is set, but step %q defines device_id…`; neither → `global_device_id is not set and step %q is missing device_id…` (`pkg/internal/pipeline/mobile_device_ids.go:14-40`). The CLI checks the same rule before storing/starting (`cmd/cli/pipeline.go:232`, `389`). The device record is resolved server-side via the internal `GET /api/mobile-device?device_identifier=…`.
- Device types normalized: `emulator`→`android_emulator`, `physical`→`android_phone`, `ios_simulator`, `ios_phone`, `redroid`.
- The child workflow stays on the pipeline task queue; runner-directed work goes to `<canonified runner_id>-TaskQueue` (`pkg/workflowengine/pipeline/mobile_automation_hooks.go:1743-1745`).
- HTTP to the runner through the `mobile-runner-http-request` activity: `POST {runner_url}/credimi/installer-action` (`{version_identifier, action_identifier, platform, device_identifier[, skip_installer]}`) and `POST {runner_url}/credimi/pipeline-result` (`{video_path, last_frame_path, run_identifier, device_identifier, platform[, log_path]}` → `result_video_urls`, `screenshot_urls`); the rest of device work (install APK, start/stop recording, cleanup) is activities on the runner queue.
- `version_id: installed_from_external_source` means "app already installed" and sets `skip_installer` (`mobile_automation_hooks.go:34`, `553`).
- Setup hook order: `MobileAutomationSetupHook → ConformanceCheckSetupHook → PipelineEvidenceSetupHook`; cleanup: `PipelineReportCleanupHook → MobileAutomationCleanupHook → ConformanceCheckCleanupHook → tempCredentialsCleanupHook → tempUseCaseVerificationsCleanupHook → tempWalletVersionCleanupHook` (`pkg/workflowengine/pipeline/hooks.go:31-46`).

## `runtime`

| Key                          | Effect                                                                                                                                                                                                                        |
| ---------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `global_device_id`           | Fallback device for every `mobile-automation` step; also feeds the device search attribute                                                                                                                                    |
| `temporal.execution_timeout` | Workflow execution timeout; default `24h`                                                                                                                                                                                     |
| `temporal.activity_options`  | Global activity options: `schedule_to_close_timeout` (10m), `start_to_close_timeout` (5m), `heartbeat_timeout` (30s), `retry_policy{maximum_attempts: 5, initial_interval: 5s, maximum_interval: 1m, backoff_coefficient: 2}` |
| `debug`                      | Runs a debug activity after every regular step; failures are logged, not fatal                                                                                                                                                |
| `fixture`                    | Overrides `${fixture.<key>}` values                                                                                                                                                                                           |
| `schedule.interval`          | If set, `Start` creates a **Temporal schedule** (id `schedule_id_<Pipeline-…>`) instead of one run                                                                                                                            |
| `disable_android_play_store` | Disables the Play Store on Android/Redroid devices before recording                                                                                                                                                           |

## Failure semantics

- `continue_on_error: true` records the failure, still runs `on_error`, and continues. The run then fails at the end with `PipelineExecutionError` **unless** `config.collect_pipeline_step_failures` is set, in which case it finishes successfully with the failures attached.
- `on_error` runs for every failed step (with or without `continue_on_error`); `on_success` after a successful step. Hook failures are appended to the failure list and never abort the pipeline.
- `finally` conditions: `always` always; `on_success` only for `success`; `on_failure` only for `failed`. **A canceled run runs only `always`.** Finally steps execute in a disconnected context so they survive cancellation; their failures surface as `finally_errors`.

## Running

**Queue (what the GUI and CLI use).** `POST /api/pipeline/queue` with `{pipeline_identifier, yaml}` (both required), auth required (`pkg/internal/apis/handlers/pipeline_queue_handler.go:34-37`).

- No device steps → started directly, response `status: "running"` with `workflow_id`/`run_id`, and a `pipeline_results` row created immediately.
- Device steps → one ticket per device on that device's semaphore; responses carry `ticket_id`, `status`, `position`, `line_len`, ids and URLs (`PipelineQueueResponse`).
- Guards: runner ownership (`404 runner not found`, `403 device_id is not accessible` — unless the device is yours, or its runner is `published` and either your organization is published or the runner is `admin_managed`; the same rule as the runner list), liveness (`503 device runner is offline`), queue cap (`409 queue_limit`).
- `GET`/`DELETE /api/pipeline/queue/{ticket}` require the `device_ids` query parameter (repeated or comma separated).
- Statuses: `queued`, `starting`, `running`, `failed`, `canceled`, `not_found`. `position` is **0-based** from the API; the wallet-APK response and the CLI display `position + 1`.
- Cancel of a not-yet-running wallet-APK ticket deletes the temporary wallet version.

**CI, wallet APK.** `POST /api/pipeline/run-wallet-apk`: `pipeline_identifier` required, `metadata.sha` required (becomes `commit_sha`), `device_id` or `device_type`, and exactly one of `apk_file` (multipart) or `apk_url` (max 1 GiB, 30s download timeout). It creates a temporary `wallet_versions` record for that SHA, rewrites the run YAML to it, forces run type `CI`, and enqueues through the semaphore. Siblings: `/run-issuer`, `/run-verifier`.

**Ephemeral execute.** `POST /api/pipeline/execute` takes YAML in the body, query `deeplink=true` / `redirect=true`. Only `http-request` steps are allowed; no `pipeline_results` row, no search attributes; 2m timeout.

**Live view.** `POST /api/pipeline/live-view` `{workflow_id, run_id[, device_id]}` — only for **running** dynamic pipeline workflows; `409` if the mobile devices are not initialized yet; `422` for `ios_simulator`; returns `{streams:[{device_id, device_name, url}]}` where `url` is the runner's `/live/<token>` path.

**CLI.** `credimi pipeline -k <api-key> [-p file.yaml]` (stdin if no `-p`) finds-or-creates the pipeline record and queues it; `credimi pipeline store` only stores; `-i` sets the instance (default `https://credimi.io`). Auth: API key → `GET /api/apikey/authenticate`, then `GET /api/organizations/my`.

**GUI.** Pipelines page → pick the runner via the gear next to **Run** → **Run**; the run opens as a Temporal timeline with video, screenshots and logs.

There is **no** `POST /api/pipeline/start` route in this worktree (grep over `pkg/internal/apis`); `AGENTS.md` still names it. The GUI always queues, and the queue handler starts device-less pipelines directly.

## Where runs live

- Collection `pipeline_results`: `owner`, `pipeline`, `workflow_id`, `run_id`, `canonified_identifier`, `type` (`manual|scheduled|CI`), `devices`, files `video_results`, `screenshots`, `logcats`, `ios_logstreams`, `maestro_screenshots`, `report`, `fcaf_report`, `fcaf_report_pdf`, plus `credential_well_knowns`, `presentation_results`. The current unique index is `(canonified_identifier, owner)`; `canonified_identifier` is derived from `workflow_id`.
- Created and filled by the internal `POST /api/pipeline/pipeline-execution-results` (and `/{evidence,report,fcaf-report}` sub-routes), gated by the internal admin key. Runners append video, screenshot and log files through `POST /api/wallet/store-pipeline-result` (user or internal-admin auth; the device must be one of the record's `devices`).
- List/view rules are `null` (superuser-only); users read results through the execution API below. Artifact files are unprotected and downloadable by URL.
- Listed via `GET /api/pipeline/list-executions`, `/list-executions/{id}`, `/executions/{id}/{workflow_id}/{run_id}`; artifacts are enriched from files.
- Temporal search attributes: `PipelineIdentifier`, `DeviceIdentifiers`, `ActionsID`, `VersionsID`, `CredentialsID`, `UseCaseID`, `ConformanceCheckID`, `CustomCheckID` (`pkg/workflowengine/search_attributes.go:13-24`).
- Cancellation policy signal `pipeline-cancellation-policy` with `{reason, skip_device_cleanup, skip_device_cleanup_ids[]}`; skipping cleanup appends `cleanup_warnings`.
- Video recordings are limited to 30 minutes (`result_video_warning` is pre-seeded when a `mobile-automation` step exists).

## Scheduling

1. **In-YAML:** `runtime.schedule.interval` (e.g. `1m`) turns `Start` into a Temporal schedule.
2. **Stored schedules:** `POST /api/my/schedules/start` `{pipeline_id, schedule_mode, global_device_id}` where `schedule_mode` = `{mode: daily|weekly|monthly, day}`; also `GET /api/my/schedules`, `POST /api/my/schedules/{scheduleId}/{cancel,pause,resume}`. A `schedules` record keeps `temporal_schedule_id`, `pipeline`, `mode`, `owner`.
