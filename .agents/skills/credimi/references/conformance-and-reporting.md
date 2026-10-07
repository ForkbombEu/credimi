# Conformance checks, results, scoreboard, reports

Verified against source. Where the manual pages disagree with the code, the disagreements are listed at the end — trust the code.

## Suites and providers

Shipped matrix under `config_templates/` (a suite directory must have `version.yaml` and `metadata.yaml` to be catalogued):

| Filesystem path                                                                                                                    | Provider          | Upstream engine                                                                                                   |
| ---------------------------------------------------------------------------------------------------------------------------------- | ----------------- | ----------------------------------------------------------------------------------------------------------------- |
| `openid4vci_wallet/1.0/openid_conformance_suite`                                                                                   | OpenID Foundation | OpenID certification API                                                                                          |
| `openid4vp_wallet/1.0/openid_conformance_suite`, `openid4vp_wallet/draft-24/openid_conformance_suite`                              | OpenID Foundation | OpenID certification API                                                                                          |
| `openid4vci_issuer/1.0/openid_conformance_suite`                                                                                   | OpenID Foundation | OpenID certification API                                                                                          |
| `openid4vp_verifier/1.0/openid_conformance_suite`                                                                                  | OpenID Foundation | OpenID certification API                                                                                          |
| `openid4vci_wallet/1.0/webuild`, `openid4vp_wallet/1.0/webuild`, `openid4vci_issuer/1.0/webuild`, `openid4vp_verifier/1.0/webuild` | WEBUILD           | `webuild.api.forkbomb.eu` (wallet standards) / `webuild.wallet-client.forkbomb.eu` (verifier, issuer)             |
| `openid4vci_wallet/draft-15/ewc`, `openid4vp_wallet/draft-23/ewc`                                                                  | EWC               | `ewc.api.forkbomb.eu` (`/verificationStatus`, `/issueStatus`, `/session-status/{sessionId}`, `/logs/{sessionId}`) |
| `openid4vp_verifier/draft-23/eudiw`                                                                                                | EUDIW             | `verifier-backend.eudiw.dev/ui/presentations` + Credimi callback page `/tests/wallet/eudiw`                        |
| `vlei/version/vlei`                                                                                                                | vLEI              | Local Go CESR validation                                                                                          |
| `fcaf/wallet_solution/relying_party`                                                                                               | FCAF              | Local Go engine (`pkg/fcaf/engine`) via the `fcaf-validation` pipeline step                                       |

Provider slugs are constants in `pkg/workflowengine/workflows/conformance_check.go:21-33`; short labels come from `config_templates/providers.yaml` (`openid_conformance_suite`→OpenID, `webuild`→WEBUILD, `ewc`→EWC, `eudiw`→EUDIW, `vlei`→vLEI, `fcaf`→FCAF).

`visible_in` decides where a suite/check shows up (`manual`, `pipeline`; empty = both). eudiw and vlei are `manual`-only; FCAF and most OpenID suites are `pipeline`-only (`pkg/conformancecatalog/catalog.go:178-198`).

Prerequisites: the `stepci-captured-runner` binary for every classic provider (env `BIN`, default `.bin/`), `OPENIDNET_TOKEN` for the OpenID certification API, and the PocketBase Settings **Application URL**: starters copy it into workflow config as `app_url` (`workflowengine.WithAppConfig`), and workflows use it only to build links (Temporal UI, emails, the EUDIW callback page). Live check logs reach the webapp through the `Send realtime logs` activity on the PocketBase realtime topic `<workflowID>{openidnet,eudiw,ewc}-logs`.

## How a check runs

1. Start: `POST /api/compliance/{protocol}/{version}/save-variables-and-start` (auth required) — `pkg/internal/apis/handlers/compliance_handlers.go:38`, handler `HandleSaveVariablesAndStart`.
2. Dispatch by suite: `workflowRegistry` maps `ewc`, `webuild`, `openid_conformance_suite`, `eudiw`, `vlei` to their workflows (`start_conformance_check_handler.go:93-134`).
3. Execution: the conformance workflow renders the suite's StepCI YAML, runs it through the StepCI activity, then either polls the OpenID certification log API or spawns a child status workflow (`PARENT_CLOSE_POLICY_ABANDON`) for EWC/WE BUILD (`pkg/workflowengine/workflows/conformance_check.go:258-523`).
4. Polling: OpenID every 5s against `/api/log/<rid>` until `FINISHED`/`INTERRUPTED`, failing on `INTERRUPTED` or `testmodule_result == FAILED`; EWC/WE BUILD statuses `success`/`pending`/`failed` (pending requires `reason == "ok"`).
5. The check can also be run from a pipeline with the `conformance-check` step (`check_id` + optional `test`, `parameters`).

Upstream status vocabulary: `FINISHED`, `INTERRUPTED`, `FAILED` (OpenID); `success`, `pending`, `failed` (EWC/WE BUILD). Temporal statuses surfaced by the API are `Running, TimedOut, Completed, Failed, ContinuedAsNew, Canceled, Terminated, Unspecified, Queued`.

## Where results live

The per-check collections were **deleted**: `conformance_checks_results` (`pb_migrations/1758704866_deleted_conformance_checks_results.js`) and `reports` (`1758704843_deleted_reports.js`). Do not reference them; `migrations/pb_schema.json` is a stale snapshot that still lists them.

Results are now `pipeline_results` rows (see the pipelines reference for the full field list): `workflow_id`, `run_id`, `pipeline`, `type` (`manual|scheduled|CI`), device relations, and the artifact files `video_results`, `screenshots`, `maestro_screenshots`, `logcats`, `ios_logstreams`, `report`, `fcaf_report`, `fcaf_report_pdf`.

Records are written in-process by `pkg/internal/pipeline_results` (no HTTP route): `Create` (from `POST /api/pipeline/queue` and the `StartQueuedPipelineActivity`; idempotent on `(workflow_id, run_id)`, `ErrConflict` for another owner/pipeline, validates the run type; device access is checked before the run is queued), `StoreEvidence` (the `Extract pipeline conformance evidence` activity), `StoreReport` (the `Generate pipeline conformance report` activity). The FCAF assessment is stored by the `fcaf-validation` activity (`pkg/workflowengine/activities/fcaf_report_store.go`).

Runner artifacts are appended to an existing record by `POST /api/wallet/store-pipeline-result` (`RequireInternalAdminOrAuth`): `credimi-runner` sends its `CREDIMI_USER_API_KEY`, falling back to the internal admin key. It uploads `result_video`, `last_frame` and `logfile` into `video_results`, `screenshots` and `logcats`/`ios_logstreams`. This route and `POST /api/pipeline/store-step-screenshots` share `authorizePipelineResultDeviceStoreAccess`: every caller, internal admin included, must name a device listed in the record's `devices` (otherwise 403 `device did not run this pipeline result`); a non-admin caller must also belong to the record's organization or own that device's published runner while the record's organization is published.

## Scoreboard

- Aggregation: `AggregateScoreboardWorkflow` (queue `AggregateScoreboardTaskQueue`) runs four activities that read and write the database directly (`pkg/internal/apis/handlers/scoreboard_activities.go`; names in `pkg/workflowengine/workflows/scoreboard.go`): `List organization namespaces for the scoreboard`, `Get the pipeline scoreboard of a namespace` per namespace, recomputes rates, enriches with `Get scoreboard execution details`, then `Save aggregated scoreboard results` (truncates and rewrites the cache). It runs in the Temporal `default` namespace and can be started/scheduled with `POST /api/pipeline/scoreboard/aggregate/start?schedule=<seconds>` (cancel: `DELETE /api/pipeline/scoreboard/aggregate/schedule/{schedule_id}`).
- The two remaining scoreboard routes are in the internal group (`AuthenticationRequired: false`) but each is guarded by the internal admin API key.
- Cache collection `pipeline_scoreboard_cache`: unique `pipeline` relation, `mobile_devices`, `total_runs`, `total_successes`, `manually_executed_runs`, `scheduled_runs`, `minimum_running_time_seconds`, `first_execution`, `latest_execution`, related entity relations, and `expanded_data` — a deliberately narrow display snapshot (pipeline, devices, wallets, versions, issuers, verifiers, credentials, use case verifications, custom integrations, `latest_execution{created, artifacts}`).
- The cache is **public-read** (`listRule`/`viewRule` = `""`); writes stay internal.
- Only `published` pipelines are aggregated or returned.
- A run counts as a success only when Temporal status is `Completed`; `Running`, `Canceled`, `Terminated` are excluded from the stats. `success_rate = round(successes/runs*10000)/100`.
- Webapp: public `/scoreboard` page (sorted by `-latest_execution.created`), plus a random sample on the public homepage.

## Reports

- Markdown conformance report: generated by the activity `Generate pipeline conformance report` through the external module `credimi-conformance-assessment`, only when the run has an evidence step; produced in a cleanup hook on a disconnected context and stored by the same activity as `pipeline_results.report` (`pipelineresults.StoreReport`).
- FCAF assessment: the `fcaf-validation` activity writes `fcaf_report` as `fcaf-assessment.json` (enriched with the presentation projection) and renders `fcaf_report_pdf` as `fcaf-assessment.pdf` (renderer in `pkg/fcaf/reportpdf`; a PDF failure keeps the JSON). 50 MiB PDF limit; JSON falls back to the PocketBase default (5 MiB).
- Files are served from `GET /api/files/pipeline_results/{recordId}/{filename}`; the webapp exposes JSON and PDF downloads from the FCAF report sheet. The file fields are not `protected`, so anyone holding a file URL can download it; the public scoreboard relies on this through the URLs stored in its cache.
- Retention: `POST|DELETE /api/pipeline/retention/schedule` (internal admin) manage the retention workflow, whose `Delete old pipeline result files` activity (`pipelineresults.DeleteFilesOlderThan`) clears `video_results, screenshots, maestro_screenshots, logcats, ios_logstreams, report, fcaf_report, fcaf_report_pdf`; defaults 30 days, batch 100, interval 1.

## FCAF artifacts

- `engine.Report`: `suite`, `selected_test_ids`, `status`, `tests[]` (id, title, status, suite, assertions, normative references, evidence, message), `executed_tests[]`, `evidence` (name → record), `summary`, `failures[]`, `presentation`.
- Per-test statuses: `pass, fail, blocked, skipped, not_applicable, inconclusive, error`; aggregate verdict precedence `error > fail > blocked > inconclusive > not_applicable > pass`.
- Presentation projection: `{deeplink, screenshots[]{url,label,test_ids}, summary_filters[]}` — built from Maestro screenshots and evidence deeplinks. Render from the presentation; do not scrape raw evidence.
- Taxonomy: derived from the test id prefix `WS_RP_<CATEGORY>_<SUBGROUP>_…`; canonical category order `DM, MS, IA, SM, SH, UC, OTHER` (Data model, Message structure, Interaction, Security mechanisms, Shared, Use cases, Other); unknown ids fall back to `OTHER`.
- Test id convention: `WS_RP_<Area>_<Subgroup>_<specifics>_<NNN>`, e.g. `WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001`; path identity `fcaf/<version>/<suite>/<test_id>`.
- The catalog does **not** map test ids to scenarios — that mapping lives in each FCAF pipeline's `fcaf-validation` step (`test_ids:` plus `pipeline_outputs:` evidence keys such as `pipeline.pid.presentation.sdjwt.all-claims`), fed by the scenario files under `config_templates/fcaf/wallet_solution/relying_party/scenarios/`.
- Validators are dotted ids in the namespaces `sdjwt.*`, `mdoc.*`, `oid4vp.*`, `pid.*` (e.g. `sdjwt.claim_present`, `mdoc.namespace_element_present`, `oid4vp.dcql_response_satisfies_constraints`, `pid.sdjwt_vct_pid`).

For authoring FCAF tests use the dedicated skill `fcaf-definitions`.

## Publication boundaries

- Every publishable entity has a `published` boolean (pipelines, wallets, credential issuers, credentials, verifiers). Read rules follow "published OR org member".
- `pipeline_results` is superuser-only through the raw collection API (list/view rules are `null`); users read results through the execution API and webapp pages, and artifact files stay downloadable by URL. The scoreboard cache is public-read but only aggregates published pipelines.
- Manual checks run under the caller's organization; results require matching owner + pipeline.
- Open caveat: the scoreboard's `expanded_data` embeds related entity names/logos without filtering those relations by `published`. Treat that as unresolved, not as intended behaviour.

## Doc/code disagreements (do not repeat the docs)

1. `manual/compliance-checks.md` advertises W3C-VC (VC-API) issuer/verifier suites and "OpenID4CI" — no such suite directories exist; the real providers are listed above.
2. `manual/conformance/index.md` advertises a "PagoPA Wallet Conformance Test" — no PagoPA suite exists under `config_templates/`.
3. Scoreboard docs cite `$lib/scoreboard/functions.ts` / `loadData()` and describe a per-namespace GET as "internal/trusted" — the helper is `Records.loadPage` in `webapp/src/lib/scoreboard/records/index.ts`, and the per-namespace scoreboard is now an activity, not an HTTP route.
4. `migrations/pb_schema.json` is stale (lists `conformance_checks_results`, `reports`, `standards`, `test_suites`; lacks `pipeline_results`, `pipeline_scoreboard_cache`, `custom_checks`, `mobile_devices`).
5. `pipeline_scoreboard_cache.last_execution_date` is written by `setBasicFields` but no migration defines the field — present in the API response, possibly not persisted.
