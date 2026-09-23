<!--
SPDX-FileCopyrightText: 2025 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2025 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# HITL

Human-in-the-loop decision backlog for Credimi agents.

Use this file when an agent finds a convention, architectural rule, dependency contract, validation rule, design rule, or workflow expectation that is missing or ambiguous in `AGENTS.md`.

Do not treat an entry here as approved policy until a human maintainer resolves it and the decision is moved into `AGENTS.md` or another canonical project document.

## Template

```md
### YYYY-MM-DD - Short question

- status: open | resolved | rejected
- owner: human maintainer | agent | unknown
- context:
- question:
- options considered:
- default risk:
- decision:
- follow-up:
```

## Open Questions

### 2026-09-23 - Hub suite-grain facet UI vs #1402 check-grain facets

- status: resolved (agent restore during PR review)
- owner: agent
- context: #1402 wired protocol/sut/role/provider filters on the check-grain hub table; suite-table rewrite dropped that UI while query helpers for SuiteFacets remained. Spec review of PR #1404 flagged missing “at least one product surface filters via PB”.
- question: Which facet axes should the hub table expose?
- options considered: (1) restore check-grain protocol/sut/role/provider on suite table; (2) suite-grain standard/component/version/provider (HITL normalize + ADR-0002); (3) leave search-only.
- default risk: Leaving search-only fails #1402 AC; check-grain facets on suite rows would query the wrong collection.
- decision: Option (2). Hub `conformance-checks-table.svelte` filters via `listSuites` + SuiteFacets.
- follow-up: None for PR #1404; broader path-normalize facet rename remains under the open normalize-paths HITL.

### 2026-09-23 - Normalize conformance catalog paths (FCAF-as-suite, 2×3→4)

- status: open (partially settled)
- owner: human maintainer
- context: Boss domain cut — 2 standards (OpenID4VP, OpenID4VCI), 3 components (wallet, issuer, verifier), 4 derived combinations; FCAF is a Commission test suite family, not a standard. Today path identity is four segments mirroring `config_templates`: `openid4vp_wallet|openid4vp_verifier|openid4vci_wallet|openid4vci_issuer|fcaf|vlei` / version / suite / check. FCAF uses `fcaf/wallet_solution/relying_party/<test_id>` (SUT as version, FCAF party as suite). Classic `role` already holds component values; FCAF `role: relying_party` pollutes the same facet. Hub/pipeline/scoreboard/`parsePath` all assume exactly four segments and durable path strings in YAML + execution history.
- question: What is the target path identity, and how do we migrate filesystem + stored references?
- options considered:
  (A) `{standard}/{version}/{suite}/{check}` with `standard ∈ {openid4vp,openid4vci,…}`, component as projected field (not a path segment); FCAF suites become suites under the matching standard (e.g. OpenID4VP × wallet).
  (B) Five-segment paths inserting component: `{standard}/{component}/{version}/{suite}/{check}` — breaks `parsePath` and every URL.
  (C) Keep filesystem layout; projection/rewrite maps old FS paths → normalized identity (aliases for old paths).
  (D) Physical move of `config_templates` to match (A), plus redirect/alias table for old paths in pipelines/scoreboard.
- default risk: Path strings are identifiers in pipeline YAML, Temporal/search, scoreboard `conformance_checks`, hub URLs, and FCAF `test_ids` pairing. Silent rename without aliases breaks historical runs and saved pipelines. Coupling FS path to identity makes (A)+(D) a large mechanical move; (C) risks two sources of truth if incomplete — especially if `path` stays FS-shaped while `standard`/`version`/`suite` become normalized and diverge from path segments (`nestChecks` / hub URLs / `parsePath` assume they align).
- decision (2026-09-23, partial): **Do not rename filesystem paths or durable path strings for now.** Normalize **only in the in-memory catalog projection** (ephemeral rows / logical suite·check collections). Physical `config_templates` layout and stored pipeline/scoreboard path references stay as today until a later explicit cutover.
- follow-up: Still decide whether ephemeral `path` stays FS-identical while normalized dimensions live in other columns (`standard`/`component`/…), and how hub URLs + nest group when those diverge. Related: drop/rename hub `role` facet to `component`; suites vs checks projections (#1396 grill).
- update (2026-09-23, agent): Additive check columns `norm_standard` / `component` / `norm_version`; FS `standard`/`version`/`suite`/`path` unchanged. Suites projection at `conformance_suites`. FCAF relying_party pack → OpenID4VP×wallet with **empty** `norm_version` (no pinned profile; FS `wallet_solution` is SUT, not standard version). Hub facets: Standard/Component/Version/Provider.
- update (2026-09-23, agent): Hub suite cell uses existing suite `metadata.yaml` `name` → `suite_name` (column header already i18n `Suite` → "Test suite"). **Do not add a second label field** unless product wants a short display name distinct from full `name`.
- update (2026-09-23, human): Nest/detail grouping when FS ≠ product stays **FS nest (A)**. Hub table = product suites; detail/pickers = FS nest; load each only on the route that needs it (ADR-0002 dual browse). Product nest deferred.

### 2026-09-23 - Suite hub label: reuse metadata `name` vs new field

- status: open
- owner: human maintainer
- context: Hub Test suite column needs a proper human label. Suite `metadata.yaml` already authors `name` (denormed as `suite_name`). Earlier polish briefly showed raw `provider` UID instead because full names felt repetitive next to a provider subtitle.
- question: Should hub use existing `name`, or add a new metadata field (e.g. `label` / `short_name`)?
- options considered:
  (A) Reuse `name` as the Suite cell primary label (recommended default).
  (B) Add `label`/`short_name` in metadata for denser table text; keep `name` for suite page titles.
  (C) Derive from `provider` UID only (rejected for product-facing hub — looks like an identifier).
- default risk: (B) duplicates authorship burden across every suite `metadata.yaml` for little gain if `name` is already good; (C) looks unfinished next to logos.
- decision: pending human — agent defaulted to (A) for hub table; FCAF `metadata.yaml` `name` updated from "Relying Party" → "FCAF Functional Conformance Assessment" (display-only; path/uid unchanged).
- follow-up: Confirm short-name field still unwanted; polish other suite `name`s if hub density needs it.
### 2026-09-21 - Conformance catalog boot rebuild soft-fail (#1397)

- status: resolved (agent default for cleanup commit 4)
- owner: agent
- context: Spec #1397 requires catalog rebuild from filesystem on boot, but `Register` only logged a warning on rebuild failure, leaving durable `conformance_checks` rows potentially stale.
- question: Should boot hard-fail on any rebuild error, or tolerate missing templates for test apps / empty checkouts?
- options considered: (1) always return bootstrap error; (2) warn-only (shipped originally; rejected); (3) skip when templates dir is missing, fail bootstrap when the dir exists but ensure/rebuild errors.
- default risk: Option (1) breaks PocketBase test apps without `config_templates`; option (2) silently serves stale rows after a failed production boot rebuild.
- decision: Option (3). `bootRebuild` skips missing dirs with a warn log; ensure/rebuild failures return from `OnBootstrap`. Documented in `pkg/conformancecatalog/hooks.go`.
- follow-up: None unless product wants hard-fail even when templates are absent.

### 2026-09-21 - Conformance catalog facet completeness (#1402)

- status: resolved (classic suite metadata backfill; map removed)
- owner: human maintainer
- context: #1402 removes `/api/template/blueprints` and populates `protocol`/`sut`/`role`/`provider` on `conformance_checks` so the hub can filter via PocketBase queries. Orthogonal axes: `protocol` (wire/spec family), `sut` (EUDI product class), `role` (party under test), `provider` (suite author/runner UID).
- question: Should maintainers backfill explicit facet fields into every suite/check YAML (and retire `knownStandardFacets`), or keep the UID map until a path-layout redesign lands?
- options considered: (1) authored metadata everywhere (preferred long-term); (2) keep/grow UID map (not preferred); (3) parse standard UIDs heuristically (rejected as long-term model).
- default risk: New classic suites without authored `protocol`/`role`/`provider` in suite `metadata.yaml` will have empty protocol/role until metadata is added; provider still falls back to suite UID / `fcaf`.
- decision (2026-09-22): Honest shared table locked. Classic OpenID: author `protocol`/`role`/`provider` in suite `metadata.yaml` only; leave `sut` empty (do not twin role). vLEI: `protocol`/`provider` only. FCAF: keep in-file `suite.sut`/`suite.role`; provider defaults to `fcaf`; do not mass-edit FCAF YAMLs. Removed `knownStandardFacets`; resolve precedence is file → suite metadata → provider fallback.
- follow-up: Hub facet filter field labels use paraglide (`Protocol`/`SUT`/`Role`/`Provider`/`Clear_filters`); option values remain UID literals until product wants humanized facet values. FCAF suite `metadata.yaml` now authors `provider: fcaf` (protocol still omitted). #1399 meta denorm remains separate.

### 2026-09-21 - Nested picker metadata from flat catalog rows (#1399)

- status: superseded (see `docs/adr/0002-suite-grain-owns-catalog-display-metadata.md`)
- owner: human maintainer
- context: #1399 cuts start-checks and pipeline pickers to `pb.collection('conformance_checks')`. Check rows carry facets plus denormalized suite display fields from suite `metadata.yaml`.
- question: Should nested FE trees keep UID-as-name + empty URL/logo fallbacks until catalog rows grow metadata (or #1400)?
- options considered: (1) UID fallbacks only (shipped initially); (2) ~~dual-fetch blueprints~~ superseded by #1402; (3) denorm suite/standard/version meta onto check rows; (4) separate meta projection (rejected for v1 — dual-fetch pain).
- default risk: Denorm duplicates suite strings across FCAF rows; acceptable in process-private `:memory:` cache. Standard/version names remain humanized UIDs until a follow-up.
- decision (2026-09-22): Option (3) suite-first. Project `suite_name`/`suite_logo`/`suite_homepage`/`suite_repository`/`suite_help`/`suite_description` onto each check; `nestChecks` prefers them over humanized suite UIDs. Empty suites stay absent.
- follow-up: Optionally denorm standard.yaml / version.yaml authored names next if pickers still look UID-y. Do not resurrect empty suites or blueprints.
- amendment (2026-09-21): Option (2) is obsolete after #1402 deleted the blueprints API and nesting path.
- amendment (2026-09-21, cleanup): Nest surfaces catalog `title` on suite `titles[]` and humanized UIDs for standard/version (and suite when suite_name empty).
- amendment (2026-09-22): Suite display denorm shipped as above.
- amendment (2026-09-23): Superseded. Suite collection owns display meta; nest/pickers build from suite rows (`fs_*` grouping). Do not re-denorm suite fields onto checks — see ADR-0002.

### 2026-09-21 - Blueprints nested metadata storage (#1398)

- status: superseded (#1402)
- owner: agent
- context: Flat `conformance_checks` rows cannot represent empty suites or full standard/version/suite.yaml metadata formerly required by `/api/template/blueprints`.
- question: Where should nested blueprints metadata live after the catalog adapter replaces the handler filesystem walk?
- options considered: (1) re-read YAML metadata per blueprints request; (2) store nested tree only in the in-memory catalog snapshot beside flat checks; (3) widen PB collection schema for nested JSON.
- default risk: Option (1) reintroduces a duplicate walk; option (3) couples PB schema to a compatibility DTO.
- decision (historical #1398): Option (2). `LoadFromDir` built checks + nested `Blueprints` once; `Catalog.Blueprints(surface)` projected/filtered with no per-request FS walk. PB collection remained the flat query cache.
- supersession (2026-09-21, #1402): Blueprints nesting and `/api/template/blueprints` were deleted. Catalog surface is flat `conformance_checks` rows (+ facet fields) only. Do not restore nested blueprints snapshot or the blueprints HTTP adapter. Durable decisions that remain: filesystem SoT, process-private `:memory:` query cache + fake PB collection URL (see #1397), UID/facet fields on checks (see #1402).
- follow-up: None — blueprints path closed.

### 2026-09-21 - Conformance catalog query cache storage (#1397)

- status: resolved (human 2026-09-22 — reverse agent durable-table default)
- owner: human maintainer
- context: #1396/#1397 require an ephemeral SQLite query cache (`:memory:` / equivalent), not a durable second catalog in `pb_data`. Probe showed SQLite rejects permanent `CREATE VIEW` over ATTACH, so a PB view collection over ATTACH is non-viable. An earlier agent default wrote a replaceable base collection in `data.db` against the explicit ephemeral requirement.
- question: Where should the PocketBase-queryable projection live?
- options considered: (1) ATTACH + view collection — impossible in SQLite; (2) ATTACH + per-connection TEMP VIEW shadowing — rejected as production hack; (3) durable base collection replaced on rebuild — rejected (not ephemeral); (4) process-private `:memory:` SQLite + Credimi-owned routes on the PocketBase collection URL, using `pocketbase/tools/search` for filter/sort/page; FE keeps `pb.collection('conformance_checks')`; no durable collection shell (codegen can stub types).
- default risk: Option (4) means Credimi must keep the collection URL contract; Admin will not show a real collection; typegen must be patched/stubbed for the fake collection name.
- decision: Option (4). Drop any `conformance_checks` collection shell. Serve list/get at `/api/collections/conformance_checks/records[/:id]`; reject writes. Rebuild fills `:memory:` only. Documented in `AGENTS.md` Dev Runtime and `pkg/conformancecatalog`.
- follow-up: None — facets via suite-grain hub filters (#1402); suite display on suite projection (ADR-0002). Durable-shell cleanup removed 2026-09-23 (see update below).
- update (2026-09-23): Removed in-branch durable-shell cleanup (`1790011000_*` migration + boot `dropCollectionShell`). No PB collection is created for the catalog; ephemeral `:memory:` + Credimi routes only. Historical main delete of the old real collection remains `1758704859_deleted_conformance_checks.js`.

### 2026-09-17 - Workflow timestamp presentation ownership

- status: resolved
- owner: human maintainer (user chose long-term option in chat)
- context: Pipeline/workflow list APIs localized `startTime`/`endTime`/`enqueuedAt` (and schedule `next_action_time`) to `DD/MM/YYYY, HH:mm:ss` server-side. Dense table UI wants Date + Start + End (`HH:mm`) with a next-day marker, without repeating the calendar date.
- question: Should Credimi keep server-side prettified datetimes, add display-specific DTO fields, or send RFC3339 and let the webapp format?
- options considered: (1) frontend-only parse of localized strings; (2) additive API display fields; (3) stop API prettifying, return RFC3339/ISO, UI owns formatting.
- default risk: Changing string formats breaks clients that assumed localized datetimes; keeping prettifying locks table layout into the API.
- decision: Option (3). List/summary APIs return RFC3339 instants; webapp formats Date/Start/End (and full timestamps elsewhere) in the user timezone.
- follow-up: Keep legacy localized-string parse fallback in `parseExecutionTime` until no cached clients remain; consider documenting in AGENTS.md under Routes/DTOs.

### 2026-09-17 - Webapp async fetch: TanStack Query vs custom Runed wrappers

- status: resolved
- owner: agent (per user handoff: least code to maintain; new dependency OK)
- context: Pipeline card empty-state flash came from Runed `resource` refetching when `$effect` invalidated even if the source value was unchanged (`Object.is`). A growing Credimi stack (`hydratedResource` / `PolledResource` / source-equality gates) was accumulating in `webapp/src/lib/utils/state.svelte.ts`. User priority: maintainability over inventing a mini Query clone. Research: Runed removed source equality ([PR #248](https://github.com/svecosystem/runed/pull/248)); Solid `createResource` memos with `===`; TanStack Query Svelte v6 has explicit `queryKey`, `isPending` vs `isFetching`, `refetchInterval`.
- question: Should Credimi adopt `@tanstack/svelte-query` for keyed/polled client fetches, keep expanding custom wrappers, or stay on Runed with upstream equality?
- options considered: (1) adopt `@tanstack/svelte-query` and delete the custom wrappers; (2) keep growing `hydratedResource`/`PolledResource`; (3) contribute Runed source `Object.is` and stay on Runed for thin fetches.
- default risk: Dual-maintaining Query and a fat wrapper; or locking into an undocumented Credimi async API.
- decision: Option (1) — add `@tanstack/svelte-query`, provide `QueryClient` in `webapp/src/routes/+layout.svelte` (`queries.enabled: browser`), migrate pipeline scoreboard, polled workflow lists, bulk wallet versions, and conformance-check form fetches to `createQuery`. Delete `webapp/src/lib/utils/state.svelte.ts`. Preserve sheet pause via reactive `activeSheet.count` gating `refetchInterval`. Leave `PipelineListExecutions` as its dedicated multi-id store.
- follow-up: Optionally document a one-liner convention in `webapp/AGENTS.md` once the migration is validated in UI; do not reintroduce Credimi resource wrappers without an HITL revisit.
- amendment (2026-09-18): Removed `activeSheet` / sheet-state pause. Root cause of FCAF Close failures was Credimi `ui-custom/sheet.svelte` asymmetric `bind:open` rejecting `false`; fixed with plain `bind:open` + `beforeClose` reopen. Polling no longer pauses for open sheets.

### 2026-08-28 - FCAF runner-to-device identifier mapping

- status: resolved
- owner: human maintainer
- context: Merging the FCAF implementation from main into the multi-device refactor exposes bundled pipeline values such as `forkbomb-bv-andrea/usb` under `runtime.global_runner_id`. The multi-device contract requires `runtime.global_device_id` using a concrete `<organization>/<runner>/<device>` identifier, but the repository contains no mapping from the legacy runner identifier to a device child.
- question: Which concrete mobile device identifier should replace each bundled FCAF runner identifier, especially `forkbomb-bv-andrea/usb`?
- options considered: (1) provide the intended concrete device path; (2) replace defaults with empty device identifiers and require `credimi fcaf --device-id`; (3) infer a child name from the runner identifier.
- default risk: Inferring a child name can silently target a non-existent or wrong physical device; preserving the runner path violates the device-scoped pipeline contract.
- decision: Replace every bundled runner path `<organization>/<runner>` with `<organization>/<runner>/device`; specifically, `forkbomb-bv-andrea/usb` becomes `forkbomb-bv-andrea/usb/device` and `forkbomb-bv-andrea/fcaf` becomes `forkbomb-bv-andrea/fcaf/device`.
- follow-up: Convert FCAF CLI flags, queue status query parameters, runtime keys, and bundled templates.

### 2026-07-22 - Login Turnstile gating scope

- status: resolved
- owner: human maintainer
- context: The requested login CAPTCHA must both prevent login content from loading until verification and appear where the current `Log in` button is. The login screen also offers OAuth and WebAuthn authentication paths.
- question: Should Turnstile gate the entire login experience (including OAuth and WebAuthn), or only password login; and should its success reveal the form or merely enable its submit button?
- options considered: (1) CAPTCHA-first gate that reveals all login methods after success; (2) render the existing fields and replace the password submit button with CAPTCHA until success; (3) protect password login only and leave OAuth/WebAuthn unchanged.
- default risk: Choosing the wrong scope can either leave an authentication endpoint unprotected or impose CAPTCHA on login methods that were not intended to require it.
- decision: The existing login experience is visible only after a successful Turnstile challenge. Password login sends the resulting token to the PocketBase API, which verifies it before authentication.
- follow-up: API coverage added. Frontend type checking remains blocked because Bun is unavailable in this environment.

### 2026-08-27 - FCAF direct-validation ownership after precondition removal

- status: resolved
- owner: human maintainer
- context: FCAF pipelines now embed `fcaf-validation` and pass aggregate `pipeline_outputs` directly, while all precondition YAML files were intentionally deleted. Catalog loading, `/api/fcaf/run`, `fcaf run --tests-file`, and the legacy assessment workflow still depended on precondition definitions for pipeline IDs, output decoders, and four shared assertion gates. Many tests also appeared in multiple aggregate or diagnostic pipelines.
- question: Should legacy assessment orchestration be removed in favor of running self-validating pipeline YAML directly, or preserved through a new explicit test-to-pipeline ownership manifest?
- options considered: Remove `/api/fcaf/run`, assessment/precondition workflows, and `--tests-file`; preserve them with a new ownership manifest; infer ownership from embedded `fcaf-validation.test_ids` despite duplicate owners.
- default risk: Inferring one owner from duplicate aggregate pipelines can execute wrong evidence flow; silently retaining legacy paths leaves production entrypoints broken when precondition files are absent.
- decision: Remove `/api/fcaf/run`, assessment and precondition workflows, and `fcaf run --tests-file`. Keep embedded `fcaf-validation`; validators consume aggregate pipeline results directly. Primary product goal is one FCAF pipeline run covering all feasible tests through multiple sequential scenarios and one final aggregate validation, not one wallet interaction reused across incompatible tests.
- follow-up: Completed in current worktree: four assertion gates migrated inline; legacy registrations/types removed; 559 tests assigned exactly once; 112 maintained scenarios generate one deployable aggregate pipeline with one final validation step.

### 2026-07-14 - FCAF trusted-authorities issuer fixture

- status: open
- owner: human maintainer
- context: `WS_RP_IA_MainInteraction__003` requires more than one wallet credential from the requested issuer and proof that each returned credential matches an AKI in the DCQL `trusted_authorities` array. The repository has no documented controlled issuer certificate/AKI fixture or `oid4vp.trusted_authorities_match` validator implementation.
- question: Which issuer action and certificate chain should be the canonical fixture for AKI-based `trusted_authorities` tests, and should its AKI be configured in file-backed pipeline YAML or resolved dynamically from issued credential evidence?
- options considered: Hardcode the current reference issuer AKI; derive AKI dynamically from each issued credential's X.509 chain; add a dedicated mock-issuer credential action with a stable documented certificate chain.
- default risk: A UI-only Maestro success or a hardcoded deployment-specific AKI would mark the test ready without proving the normative issuer-match condition.
- decision:
- follow-up: Implement the multiple-credential issuance flow, AKI request, and `oid4vp.trusted_authorities_match` validator after the canonical issuer fixture is selected.

### 2026-07-13 - FCAF manual pipeline source files

- status: resolved
- owner: human maintainer
- context: FCAF pipeline preconditions fetch pipeline YAML from PocketBase records by `pipeline_id`, but the repository has no documented source-of-truth folder or import path for the pipeline record YAML. The 2026-07-13 manual DCQL work needs concrete Maestro-driven pipeline bodies for `forkbomb-bv-andrea/fcaf-wallet-solution-relying-party-dcql-*`.
- question: Should manual FCAF pipeline YAML templates live under `config_templates/fcaf/wallet_solution/relying_party/pipelines/`, and should a seed/import command be added to publish them into PocketBase pipeline records?
- options considered: Store manual templates beside the FCAF catalog without runtime import; add a first-class pipeline seed/import command; keep pipeline bodies only in live PocketBase state.
- default risk: File-backed templates without an import path can drift from live PocketBase pipeline records, while live-only pipeline records make FCAF catalog changes hard to review and reproduce.
- decision: Do not store these manual precondition implementations in SQLite/PocketBase. Keep reusable standalone Maestro YAML scripts under `config_templates/fcaf/wallet_solution/relying_party/maestro-preconditions/`.
- follow-up: Replace the temporary DCQL verifier deeplink defaults once the exact request-generation endpoint for each DCQL variant is confirmed.

### 2026-07-03 - FCAF canonical source and pipeline inventory

- status: open
- owner: human maintainer
- context: `FCAF_REAL_EXECUTION_PLAN.md` requires exact normative references from the source FCAF markdown and real reusable pipeline preconditions such as `/org-owner/fcaf-wallet-solution-relying-party-pid-sdjwt-presentation-success`. In the current workspace, the catalog still contains placeholder `TODO/` pipeline identifiers, `_implementation/*.md` summaries, and no local definitions for the named FCAF pipeline IDs were found.
- question: Should the implementation proceed by treating the current in-repo `_implementation` markdown and existing Credimi pipeline assets as the temporary source of truth, or is there another canonical local/external source for the exact FCAF markdown and the real pipeline inventory that this implementation must target?
- options considered: Proceed from `_implementation` drafts and placeholder pipeline mappings, then refine later; stop until the canonical source repo and pipeline inventory are provided; implement only the engine/DSL rewrite now and defer catalog/pipeline concretization.
- default risk: If the wrong source of truth is assumed, the catalog will encode incorrect normative references and pipeline bindings, which would make the FCAF graph executor structurally correct but semantically wrong.
- decision:
- follow-up:

### 2026-07-02 - FCAF temporary conversion files

- status: resolved
- owner: human maintainer
- context: The FCAF implementation plan needs per-test draft YAML and per-test implementation notes for the wallet-solution relying-party tests. `AGENTS.md` generally forbids leaving temporary folders in the repository, but the maintainer wants these files colocated with the test YAML during implementation and deleted later.
- question: Where should temporary FCAF conversion and implementation-note files live while implementing the FCAF DSL catalog?
- options considered: Use `/tmp/fcaf-wallet-rp-work` outside the repository; use `config_templates/fcaf/wallet_solution/relying_party/tests/_implementation/` beside the test YAML; create a separate top-level temporary folder.
- default risk: In-repo temporary files can accidentally be committed or treated as permanent catalog files.
- decision: Temporary implementation files may live beside the FCAF test YAML for now, under the test catalog folder, and must be deleted during implementation once consumed.
- follow-up: Implementation agents must keep the temporary folder clearly named and remove it before finalizing production-ready FCAF catalog work unless the maintainer explicitly keeps it.

## 2026-08-30 — Scoreboard success-rate sparkline

- **Question:** Should the public scoreboard success-rate column show a sparkline of execution trends?
- **Context:** `pipeline_scoreboard_cache` holds only aggregate stats (total_runs, total_successes, success_rate, first_execution); no per-run time series is exposed to the scoreboard API. A real sparkline needs a new data source (e.g. per-pipeline execution history endpoint or cached trend buckets) — cross-cutting API/schema change.
- **Options considered:** (a) honest band-colored progress bar + score pill (implemented now); (b) sparkline fed by pipeline_results history (needs new backend query, N+1 risk on 20-row pages); (c) cached trend column in pipeline_scoreboard_cache populated by the scoreboard cache hook (schema + hook change).
- **Default risk:** Current bar shows only the aggregate ratio, not direction/trend over time.
- **Owner:** puria — **Status:** open

## 2026-09-10 — PocketBase v0.40.3 upgrade follow-ups

- **Question:** Accept the validation-tooling and generated-output changes that came with the PocketBase v0.26.4 → v0.40.3 upgrade?
- **Context:** PB v0.40 requires Go 1.27, whose `encoding/json` v2 retrofit changed `omitempty`/`UnmarshalTypeError` behavior, and its `apis.NewRouter` now binds UI routes per call. Tooling fallout: `golangci-lint v2.12.1` panics on Go 1.27 AST (bumped to `v2.13.2`, which also bumped `gofumpt`/`golines` formatting); `make lint` runs with `fix: true` and reformatted files with long lines. The regenerated `schemas/pipeline/pipeline_schema.json` (via `invopop/jsonschema v0.14`) drops `"required": ["IPv4Address", "IPv6Address"]` on the mdoc namespace object. PB v0.40's heavier per-app migration bootstrap makes the `-race` handlers suite ~8x slower (~11.5m), exceeding go test's default 10m timeout; `-timeout 30m` added to `scripts/test-summary.sh`.
- **Options considered:** Keep new formatter versions and regenerated schema (repo-owned entrypoints produce them); pin old formatters/suppress deprecations; hand-revert schema JSON.
- **Default risk:** Formatting churn touches files outside the upgrade scope; the schema JSON change relaxes pipeline-schema validation for mdoc namespace objects; full `-race` suite now needs >10m.
- **Owner:** puria — **Status:** open

### 2026-09-10 — Temporal callbacks behind Cloudflare WAF

- status: resolved
- owner: human maintainer
- context: Temporal activities used the persisted public `app_url`, causing Cloudflare WAF/browser challenges to block server-to-server callbacks. Public links must remain on `app_url`, while callbacks need an origin-reachable URL.
- question: How should deployments provide a callback URL without making persisted workflow links private?
- options considered: Replace `app_url` globally with a compose hostname; add a separate optional internal URL with public fallback; add Cloudflare allow rules for every worker egress IP.
- default risk: A private hostname in `app_url` breaks browsers, external runners, schedules, and cross-instance workers; WAF allowlists are operationally brittle.
- decision: Use `CREDIMI_INTERNAL_APP_URL`, injected as separate workflow config `internal_app_url`; callback consumers prefer it and fall back to public `app_url`. Production deployments must provision all required credentials explicitly; Docker Compose does not add development host aliases.
- follow-up: Non-Compose deployments must set `CREDIMI_INTERNAL_APP_URL` to a DNS name reachable from every Temporal worker that executes these workflows.

## 2026-09-10 — Test app lifecycle: per-test `tests.NewTestApp` retained

- **Question:** Should handler/API test suites share a single suite-level PocketBase test app (TestMain + `DisableTestAppCleanup`) instead of creating one per test?
- **Context:** Unit tests were slow (~100s for `pkg/internal/apis/handlers`). Profiling showed the dominant cost was NOT the per-test pattern but a stale `test_pb_data/data.db`: after the PocketBase v0.40.3 upgrade, two new core migrations re-ran on every `tests.NewTestApp` bootstrap (~175ms per app × 287 apps). Refreshing the fixture dropped per-app cost to ~10ms and the handlers suite from 100.5s to ~23s. Sharing one app across scenarios would additionally break isolation: scenarios mutate `Settings().Meta.AppURL`, collection schema fields (`ensure*Field` helpers), and seed records, so a shared DB would introduce test-order dependence.
- **Options considered:** (a) keep per-test apps + refreshed test data (chosen); (b) full suite-level shared app conversion; (c) hybrid shared app for read-only suites.
- **Default risk:** Per-test apps re-create a fresh isolated DB per scenario (~10ms each); any future PocketBase upgrade with new core migrations re-introduces the ~10x per-app cost unless `make testdata.refresh` is run and `test_pb_data/data.db` recommitted.
- **Owner:** puria — **Status:** resolved (decision: keep per-test apps; run `make testdata.refresh` after PocketBase or pb_migrations changes)

### 2026-09-15 - Parallel worktree local-dev contract

- status: resolved
- owner: human maintainer
- context: Multiple git worktrees need isolated Compose projects and host ports without editing tracked Procfile/compose per checkout. Recent Worktrunk-style tooling was reviewed in `.agents/research/2026-09-15-git-worktree-utilities.md`.
- question: How should Credimi expose ports and bootstrap copies for parallel worktrees?
- options considered: (1) port offset knob; (2) explicit ports in `.env.worktree` with classic defaults centralized; (3) adopt Coasts daemon for runtime isolation; (4) Worktrunk optional vs required.
- default risk: Offset math is opaque; Coasts adds a large runtime before Compose parameterization is proven; keeping a plain-git copy shim duplicates Worktrunk.
- decision: Keep classic ports in `scripts/dev-ports.env`. Override only via `.env.worktree`. Copy allowlist includes `.env`, `webapp/.env`, `webapp/node_modules/`, and `pb_data/`. Recreate `.bin` via `make tools`. **Require Worktrunk** for parallel worktrees (`wt step copy-ignored`, `hash_port` seeds); keep Credimi scripts for ports/Procfile/Compose/`sync-urls`. Primary checkout `make dev` remains Worktrunk-free. Do not enable Worktrunk commit/merge automation.
- follow-up: Pilot two local worktrees; measure whether node_modules copy is worth keeping.

### 2026-09-17 - Close Play Store after store-install pipeline runs

- status: resolved
- owner: puria
- context: Pipelines that install the wallet from the Play Store leave the store UI open on shared physical devices. The gate signal already exists server-side (`markExternalInstallSteps` sets `with.config.detect_external_install` for `version_id: installed_from_external_source` steps whose wallet action category is `install-app`), but the adb execution lives in the private `credimi-extra` module (`mobile/cleanup.go`), consumed through `CleanupDeviceActivity`. `credimi-2` pins `credimi-extra v1.15.1` with no `replace` directive, so the server-side payload key is inert until extra is released and bumped.
- question: Confirm the cross-repo contract for closing the Play Store: additive `close_play_store` cleanup-payload key decided server-side, with `adb shell am force-stop com.android.vending` executed by `credimi-extra`?
- options considered: (a) server sets `close_play_store` in the existing cleanup payload and extra force-stops the store (implemented; one cleanup round-trip, mirrors `reenable_play_store`, server-side testable); (b) extra/runner closes the store unconditionally at every Android device cleanup (no server change, fires on runs that never opened the store); (c) append a Maestro step to every install action (no Go change, per-action authoring burden, skipped on failure paths).
- default risk: `DecodePayload` is plain `json.Unmarshal`, so an un-updated extra ignores the new key silently — the feature is a no-op until `credimi-extra` is released and `go.mod` is bumped, and nothing surfaces that gap at runtime.
- decision: Option (a) — the server decides the gate and sets an additive `close_play_store` key in the existing `CleanupDeviceActivity` payload; `credimi-extra` executes `adb shell am force-stop com.android.vending` in `mobile.CleanupDevice`. The gate stays narrow: only external-source steps whose wallet action category resolves to `install-app`, Android devices only, and skipped when `reenable_play_store` is set (a store disabled for the whole run was never opened). Inline `action_code` steps without a resolvable `action_id` remain excluded, consistent with `detect_external_install`.
- follow-up: Maintainer must release `credimi-extra` and bump the `go.mod` requirement; release/version work was intentionally not performed by the agent. Until then the key is inert and deploying `credimi-2` alone is a safe no-op.
