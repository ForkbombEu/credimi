---
name: credimi
description: What Credimi is and what you can do with it — domain concepts (wallet, wallet action, wallet version, credential issuer, credential, verifier, use case verification, conformance check, conformance suite, provider, pipeline, runner/device, scoreboard, schedule), the capability map, and how to create assets, author and run pipelines, run conformance checks, and read results/reports (GUI, CLI, HTTP API, CI). Use when a user wants to understand Credimi or do something with it — adding an issuer/verifier/wallet/credential, writing Maestro actions or pipeline YAML, running conformance checks, or finding the scoreboard/report/live view. Deep detail lives in references/.
---

# Credimi

Credimi tests, evaluates and debugs credential **issuers**, **verifiers** and **wallets** against the conformance standards of the EUDI / decentralized-identity ecosystem (OpenID4VCI, OpenID4VP, OpenID Federation, EWC/WE BUILD, vLEI, FCAF).
A public **Hub** provides discovery and manual trying; the logged-in **Conformance & Interop** area runs checks and interop flows; the **Testing Automation** area chains StepCI (service side) and Maestro (wallet UI side) into **pipelines** orchestrated by Temporal and executed on devices through **Credimi runners**.

Canonical product vocabulary is `CONTEXT.md` — use those terms, including its "Avoid" lists, for any user-facing copy.

## References (read the one you need)

| Read                                                                               | For                                                                                                                                                                                                               |
| ---------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [references/pipelines.md](references/pipelines.md)                                 | Pipeline YAML grammar, every step type and its payload, expressions/outputs, failure semantics, queue/run paths, CLI, storage, scheduling, and how to verify syntax against the latest code                       |
| [references/conformance-and-reporting.md](references/conformance-and-reporting.md) | Supported suites and their upstream engines, how a check executes, result/status vocabulary, the scoreboard, markdown/FCAF reports, FCAF presentation and taxonomy, publication boundaries                        |
| [references/catalog-and-hub.md](references/catalog-and-hub.md)                     | The conformance catalog (paths, facets, `metadata.yaml`, surfaces), the served read-only collection URLs, the manual-check input form mechanism, the Hub entity model, StepCI integrations, deeplinks, publishing |
| [references/runners-and-api.md](references/runners-and-api.md)                     | Runners and devices, liveness and flags, the Credimi→runner HTTP contract, live view, auth/API keys, the full HTTP endpoint catalog, and the webapp route map                                                     |

For writing FCAF tests use the dedicated skill `fcaf-definitions`.

## The cast of things

| Thing                              | What it is                                                                                                                                                                                   |
| ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Wallet**                         | The mobile app under test (Android/iOS); publishable on the Hub                                                                                                                              |
| **Wallet version**                 | One installable build: `tag`, `platform`, uploaded installer; a pipeline installs it on the device automatically. `installed_from_external_source` as `version_id` means "already installed" |
| **Wallet action** (Maestro action) | A reusable Maestro YAML flow driving the wallet UI (open app, onboarding, PIN, open deeplink, accept/present). Categories include the special `install app`                                  |
| **Credential issuer**              | Hub parent for credentials; can be imported from a static `.well-known`                                                                                                                      |
| **Credential**                     | A StepCI script that calls an issuer and produces an offer deeplink (dynamic), or a stored deeplink (static)                                                                                 |
| **Verifier**                       | The requesting party; publishes **use case verifications** (each with its own StepCI script producing a presentation request)                                                                |
| **Conformance check**              | One runnable catalog entry with a durable filesystem-shaped `path`, a title and browse facets. It does not own suite display metadata                                                        |
| **Conformance suite**              | Checks grouped under a product axis (standard × component × version) with authored display metadata                                                                                          |
| **Provider**                       | The org/source facet for browse filters; the durable wire value is a slug, the human label is authored in `providers.yaml`                                                                   |
| **Catalog surface**                | `manual` or `pipeline` — where a suite/check may be browsed. Empty means both                                                                                                                |
| **Pipeline**                       | An ordered sequence of steps reusing the assets above; outputs pass between steps                                                                                                            |
| **Pipeline run**                   | One execution: a Temporal timeline plus video, screenshots and logs, stored as a `pipeline_results` row                                                                                      |
| **Credimi runner**                 | Hosts device/emulator/simulator execution; devices are addressed `<org>/<runner>/<device>`. At most one emulator and one simulator per runner                                                |
| **Schedule**                       | Either `runtime.schedule.interval` in the YAML or a stored `schedules` record (daily/weekly/monthly)                                                                                         |
| **Scoreboard**                     | Aggregate run statistics per published pipeline, cached in `pipeline_scoreboard_cache` and shown publicly                                                                                    |

## What you can do

**Hub (no account).** Browse published wallets, issuers/credentials, verifiers/verifications and pipelines; search and filter; open an item for details, versions and capabilities; try an issuance or verification flow through the QR/deeplink produced by the publisher's StepCI integration. Compliance scores are marked "coming soon". Legacy `/marketplace` URLs redirect to `/hub`.

**Conformance & Interop (logged in).** Start manual conformance checks (`Developer dashboard → (Tools) Manual Conformance Checks`; no-setup checks from the Hub's conformance-checks tab), fill the generated input form, then track runs under `/my/tests/runs` and open one for details and outputs. Run interop flows between wallet, issuer and verifier.

**Testing Automation.** Compose pipelines from existing assets (StepCI integration → Maestro actions → pipeline), run them on a chosen runner (gear next to **Run**), and inspect the execution timeline and the captured video, screenshots and logs. Use live view for a running device.

**Automation.** Queue runs by API/CLI, trigger wallet-APK runs from CI (and the issuer/verifier variants), schedule recurring runs, and get completion notifications.

**Results and reports.** Per-run artifacts (video, screenshots, Logcat/iOS log streams), a markdown conformance report when the run has an evidence step, and for FCAF a JSON assessment plus a PDF with the presentation (deeplink, screenshots, summary filters).

**Publishing.** Flip `published` on wallets, credential issuers, credentials, verifiers and pipelines to surface them (and their published children) on the Hub.

## Verify things against the latest code

| Question                      | Where to look                                                                                                                                                                                                                                                                                            |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Exact pipeline YAML syntax    | `credimi pipeline schema` (JSON to stdout, `-o file.yaml` for YAML) — generated from `pkg/internal/pipeline/types.go` + the step registry, wired at `pkg/gen.go:17`, regenerated by `make generate`. The pipeline editor validates with Ajv against the produced `schemas/pipeline/pipeline_schema.json` |
| Which step types exist        | `pkg/workflowengine/registry/registry.go` (`Registry` = author-facing, `PipelineInternalRegistry` = internal)                                                                                                                                                                                            |
| Who serves an endpoint        | `pkg/internal/apis/handlers/*Routes*` definitions plus `pkg/internal/apis/RoutesRegistry.go`                                                                                                                                                                                                             |
| What a collection holds       | `pb_migrations/*`                                                                                                                                                                                                                                                                                        |
| Product vocabulary and intent | `CONTEXT.md`, then `docs/adr/`                                                                                                                                                                                                                                                                           |
| User-facing walkthroughs      | `docs/src/content/docs/manual/**` — but several pages are stale or placeholders                                                                                                                                                                                                                          |

`make generate` also regenerates the catalog wire and the FCAF aggregate pipeline; `graft ask "<query>" --source` is the fastest way to find the owning code.

## Ground rules when answering Credimi questions

- **Code beats docs.** Known drift: there is no `POST /api/pipeline/start` route (the GUI and CLI use `POST /api/pipeline/queue`); `AGENTS.md` still documents the runner result endpoint as `POST {runner_url}/store-pipeline-result` with `logcat_path`/`instance_url` (real: `/credimi/pipeline-result` with `log_path` and `platform`); suites such as W3C-VC/VC-API and PagoPA have no catalog directory.
- Do not promise W3C/PagoPA suites or Hub compliance scores; both are absent from the code.
- `pipeline_results` is superuser-only through the raw collection API (list/view rules are `null`); users read results only through the Credimi API and the webapp pages. Its artifact files are unprotected and downloadable by anyone holding the URL (the public scoreboard links to them).
- Repository-engineering rules (Temporal namespaces, migrations, validation matrix, commit contract) live in the root `AGENTS.md`, not here.
