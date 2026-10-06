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

### 2026-09-24 - CatalogSurface rename vs TemplateSurface

- status: resolved (agent during PR #1404 review fixes)
- owner: agent
- context: FE type was `TemplateSurface` while CONTEXT.md and ADRs use **Catalog surface**.
- question: Align the TypeScript name with domain language?
- options considered: (1) rename to `CatalogSurface`; (2) keep TemplateSurface + comments.
- default risk: Dual vocabulary invites wrong mental model (template file vs browse filter).
- decision: Option (1). `catalogSurfaceSchema` / `CatalogSurface`.
- follow-up: None.

### 2026-09-24 - Product-axis UID maps vs #1402 long-term authorship

- status: resolved for PR #1404 (interim accepted); open for later cutover
- owner: human maintainer
- context: Spec review of #1404 noted product `standard`/`component` still come from `classicFSPrefix` / `fcafPackIdentity` in `NormalizePathIdentity`, while #1402 says “filename/UID parsing is not the long-term model.” Check facets (`protocol`/`role`/`provider`) are authored; hub filters suite-grain product axes.
- question: Author product axes into suite metadata now, or keep projection maps until path-normalize HITL closes?
- options considered: (1) document maps as accepted interim; (2) author product axes in every suite `metadata.yaml` with map fallback.
- default risk: (2) before path-normalize lands duplicates authorship and may churn again.
- decision: Option (1) for this PR. Maps remain the sole product-axis projector until the open normalize-paths HITL chooses a durable model.
- follow-up: When normalizing paths, prefer authored metadata and shrink/delete the UID maps.

### 2026-09-24 - Hub deep links: FS path vs getOne by record id (#1396 US5)

- status: resolved (agent during PR #1404 review fixes)
- owner: agent
- context: #1396 US5 asks getOne by stable record id for hub detail/deep links; hub routes remain `/hub/conformance-checks/...` FS paths (US25 / ADR-0002).
- question: Switch hub URLs to catalog record ids?
- options considered: (1) keep FS path URLs; API getOne stays available; (2) id-based hub routes.
- default risk: (2) breaks marketplace redirects and path-stable URLs without a redirect table.
- decision: Option (1). List/get-by-id capability shipped; hub deep links stay filesystem-axis paths.
- follow-up: None unless product wants id URLs later.

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
- update (2026-09-24, agent): Product-axis projection via UID maps (`classicFSPrefix` / `fcafPackIdentity`) is the accepted interim for PR #1404; see HITL “Product-axis UID maps vs #1402 long-term authorship”.

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

### 2026-09-15 - FCAF acceptable JOSE algorithm set for ECCG ACM 5.2

- status: resolved
- owner: human maintainer
- context: `WS_RP_SM_DeviceBinding__012a` requires the KB-JWT `alg` to be on
  the ECCG ACM 5.2 acceptable-algorithm list. The vendored FCAF source names
  the external standard but does not reproduce that list, and the repository
  has no canonical mapping from ECCG ACM mechanisms to JOSE identifiers. The
  current beta Capture Wallet evidence is ES256, while the source explicitly
  identifies EdDSA as unacceptable in the related RP-integrity negative case.
- question: Which JOSE `alg` values are the canonical acceptable set for this
  FCAF assertion?
- options considered: (1) encode only ES256, matching current beta evidence;
  (2) encode a maintainer-approved JOSE mapping of ECCG ACM 5.2; (3) keep the
  test blocked until the source repository publishes the mapping.
- default risk: Guessing an allowlist can reject a conforming Wallet or accept
  a disallowed algorithm while reporting a conformance pass.
- decision: For the current beta Capture Wallet definition, require the
  observed acceptable JOSE algorithm `ES256`. Revisit a broader mapping only
  when a second supported Wallet algorithm is added to the beta fixture.
- follow-up: Add the ES256 predicate to `WS_RP_SM_DeviceBinding__012a`.

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
- follow-up: Superseded on 2026-10-06: workers no longer call Credimi over HTTP (typed activities with `core.App`), so `CREDIMI_INTERNAL_APP_URL` and `internal_app_url` were removed. Deployments no longer need to set it.

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

### 2026-09-25 - Record owner on UI creates after PocketBase v0.40

- status: open
- owner: human maintainer
- context: PocketBase v0.40.3 (#1366) evaluates `createRule` before `OnRecordCreateRequest` (v0.26.4 ran the rule inside the hook chain). `pb_hooks/add_owner.pb.js` still sets `owner` from the requesting user's organization, but only after the rule, so every collection whose `createRule` checks `owner.id` rejects creates that omit `owner` (`create rule failure: sql: no rows in result set`). Reproduced against an isolated copy for wallets, verifiers, custom_checks, pipelines, wallet_versions, wallet_actions, use_cases_verifications, credential_issuers; `credentials` is unaffected because its rule checks `credential_issuer.owner`. #1375 and #1384 plus the follow-up fix send `owner` from the client (`fieldsOptions.hide.owner` or an explicit `owner` in direct `pb.collection(...).create` calls).
- question: Should the client keep sending `owner` on every create, or should the server own it again (for example by rewriting create rules or moving the owner assignment ahead of rule evaluation)?
- options considered: (a) client sends `owner` on every create (current; the rule still verifies membership, and the hook still overwrites `owner` afterwards); (b) server-side fix so clients never need to send `owner`; (c) both.
- default risk: Under (a), any new create form or direct `create` call that omits `owner` fails with an opaque "Failed to create record." error; the `add_owner.pb.js` create hook is no longer what makes those creates pass the rule.
- follow-up: Decide whether to keep the client-side contract or restore server ownership.
- note (2026-09-25): `add_owner.pb.js` overwrites the rule-checked client `owner` with the user's first `orgAuthorizations` row. Harmless while each user has one org (current product assumption, confirmed by the user); the create/update hooks must stop overwriting `owner` before multiple org memberships per user are supported. The server does not enforce one org per user: `organizations.pb.js` creates an owner authorization for the creator without checking existing memberships, and invites/join requests add memberships.

### 2026-09-25 - Pipeline live view calls the runner over direct HTTP

- status: resolved
- owner: human maintainer
- context: `POST /api/pipeline/live-view` (`pkg/internal/apis/handlers/pipeline_live_view_handler.go`) asks the runner holding a running pipeline's device for a live-view URL via `POST {runner_url}/credimi/live-view` and returns it to the UI, which embeds it in a Sheet. Runner HTTP from credimi-2 normally goes through the `mobile-runner-http-request` Temporal activity.
- question: May the live-view API handler call the runner directly over HTTP instead of through the `mobile-runner-http-request` activity?
- options considered: (a) direct HTTP from the API handler via `mobilerunner.HTTPClient`, with the internal admin key and a 15s timeout (chosen); (b) start a workflow/activity and wait on it from the request.
- default risk: The call bypasses Temporal retries and history; a slow or offline runner surfaces as `503 device runner is offline` to the user instead of being retried. The runner contract is listed in `AGENTS.md` "External runner HTTP contract".
- decision: Approved exception — option (a), because the UI needs a synchronous answer (the URL) inside the click. Precedent: `checkMobileRunnerHealthHTTP` in `mobile_runners_handlers.go`, which also calls the runner directly.
- follow-up: None.

### 2026-09-30 - AGENTS.md names pipeline/runner surfaces that no longer exist

- status: open
- owner: human maintainer
- context: While writing the `credimi` skill, three `AGENTS.md` claims were checked against code and do not hold in this worktree: (1) "UI calls `POST /api/pipeline/start`" — no such route exists under `/api/pipeline` (`pkg/internal/apis/handlers/pipeline_handler.go`); the GUI (`webapp/src/lib/pipeline/queue.ts`) and the CLI both use `POST /api/pipeline/queue`, which starts device-less pipelines directly. (2) The external runner HTTP contract lists `POST {runner_url}/fetch-apk-and-action` and `/store-pipeline-result`; the code calls `/credimi/installer-action` and `/credimi/pipeline-result` (`pkg/workflowengine/pipeline/mobile_automation_hooks.go`). (3) `X-Api-Key` is documented for internal-admin routes; the middleware reads `Credimi-Api-Key` (`pkg/internal/middlewares/auth_api_key.go`, `pkg/internal/apis/handlers/api_key_service.go`); the same stale header appears in `docs/src/content/docs/software-architecture/scoreboard.md` and a comment in `pkg/conformancecatalog/store.go`.
- question: Update `AGENTS.md` (and the scoreboard doc/comment) to the current routes and header, or treat the documented names as the intended contract and change the code back?
- options considered: (a) fix the docs/comments to match code; (b) add compatibility aliases for the old runner endpoints and accept `X-Api-Key` as an alias; (c) add a `/api/pipeline/start` alias.
- default risk: Agents and docs keep describing endpoints that 404, and sibling `credimi-extra` implementers may build against the wrong runner paths.
- decision:
- follow-up: Confirm (a) and update `AGENTS.md` "Dynamic Pipeline Workflow", "External runner HTTP contract" and "Routes, DTOs, Auth, Errors" sections.

### 2026-10-01 - FCAF complete-validation exceeds Temporal 4 MB gRPC message limit

- status: resolved
- owner: human maintainer
- context: Run `Pipeline-fcaf-wallet-relying-party-complete-validation-66a4a11b-…/01a0f3a8-1bac-7eaf-81de-fa78ef684273` (namespace `fcaf-1`) is stuck retrying its final workflow task with `WORKFLOW_TASK_FAILED_CAUSE_GRPC_MESSAGE_TOO_LARGE` (7,083,396 vs 4,194,304). The `RespondWorkflowTaskCompleted` carries the `fcaf-validation` schedule command whose resolved `pipeline_outputs` is 7.07 MB. The frontend 4 MB inbound gRPC limit is not configurable (temporalio/temporal#2786); `limit.blobSize.error` is already raised to 100 MB in `deployment/dynamicconfig/development-sql.yaml`, which only moved the failure. The same ~7 MB `state.finalOutput` is copied again after validation: `PipelineReportGenerationInput` (`report_cleanup.go:57-60`, ~9 MB with the resolved definition), the workflow failure/result `Details.output` (`pipeline.go:262`, `:274`, `:289`), and the FCAF report `Evidence` map values (`engine/report.go:250`). The evidence-extraction setup activity input is already 1.96 MB. History was 18.6 MB at failure. Measured reductions: zlib 7.07 → 0.95 MB; test-referenced keys only 7.07 → 5.91 MB; `dcql_exchange` projected to `dcql_query`/`vp_token`/`error` 2.84 → 0.17 MB; ≥256 B strings total 4.48 MB vs 1.90 MB unique (capture-wallet session body repeats the exchange in raw/http/decrypted/observed forms).
- question: Which approach should keep FCAF evidence inside Temporal while staying under the 4 MB message limit, and in what order?
- options considered: (a) zlib `PayloadCodec` in `temporalcrypto.NewDataConverter` (SDK v1.36 `converter.NewZlibCodec`, `AlwaysEncode: false`); requires the runner worker (`credimi-runner`) to ship the codec first, the history API (`workflows_handlers.go:561-600` returns raw protojson payloads) to decode server-side or a codec endpoint shared by `@forkbombeu/temporal-ui` and `tui.credimi.io`; (b) evidence by reference: `fcaf-validation` and report generation read referenced step outputs from the workflow's own history instead of receiving copies; top-level runs stop returning `finalOutput` in result/failure details, kept only for child pipelines because `activities/workflow_result.go:74-78` and `pipeline.go:559,636` consume child `Details["output"]`; (c) narrow scenario evidence bindings in `cmd/fcaf-pipeline-gen` to validator-consumed fields (needs a validator required-fields contract; insufficient alone, ~3.2 MB); (d) shrink the capture-wallet session body (service contract owner unknown); (e) split validation into sequential activities (adds ~7 MB history, report merge in workflow code); (f) external storage claim-check (last resort, rejected by user unless nothing else works).
- default risk: Without a change every complete-validation run with this scenario set wedges at the final workflow task; fixing only the validation input moves the same failure to report generation or workflow completion. History grows linearly with scenarios toward the 50 MB hard limit.
- decision: Option (b), evidence by reference. `step_id` is injected into main-step input `config`; `fcaf-validation` and report generation read the run's own Temporal history; the `fcaf-validation` activity stores the full FCAF report JSON itself, and Temporal keeps a compact report with an evidence index and sha256 hashes; pipeline results return only caller-referenced step outputs; repeated shared config is trimmed from step activity inputs; a pre-schedule step input size guard is added; no payload compression.
- follow-up: Confirm (a) as short-term unblock plus (b) as structural fix; confirm `credimi-runner` codec ownership, server-side history decoding vs codec endpoint, and dropping `output` from top-level pipeline results.
- note (2026-10-01): User prefers (b) and is open to reworking pipeline/workflow logic. Step identity for history lookup must come from `step_id` injected into every step's activity/child-workflow `config`, not from deterministic Temporal activity IDs, because pipelines are hand-written. In the analysed run `step_id` is present in 442/458 mobile child-workflow configs and absent from all 875 regular step activity configs. Proposed direction: (b) as the fix plus a pre-schedule payload-size guard; (a) deferred and, if adopted, gated by a size threshold so small payloads stay plain JSON.
- note (2026-10-01): Report PDF and page read the stored `fcaf_report` JSON artifact (PocketBase), not Temporal history. The PDF never renders raw evidence values (`reportpdf/render.go:572-608` prints key/type/source/path/from and states raw values stay in the canonical JSON); the page uses `report.evidence` values only for legacy screenshot/deeplink fallback (`webapp/src/lib/fcaf/report.ts:220,273`). Constraint from user: PDF and page must stay fully readable as today, so the stored artifact keeps evidence values; only the Temporal-carried copy may become compact. Child pipelines return outputs because parent workflow code cannot read the child's history (`runChildPipeline`, `pipeline.go:559`). Proposed uniform rule instead of an exception: a pipeline returns only the outputs its caller references (top-level: none).

### 2026-10-01 - Deploy mid-run kills in-flight pipeline workflow (worker build switch)

- status: open
- owner: human maintainer
- context: `Pipeline-fcaf-wallet-relying-party-complete-validation-3974bad8-3180-4458-8f2d-d63abfe00611/01a0f7d4-349a-7809-9fc0-dcad7f540c1d` (namespace `fcaf-1`) ran 4h23m and failed on its last workflow task with `CRE220 Unexpected output from activity: missing or invalid runner_url for step onboard-reference-wallet`, even though that step's child workflow `…-onboard-reference-wallet` completed and the error payload belongs to another step (`{code, installer_path, version_id}` — a pre-#1446 runner `installer-action` response). The Temporal history shows every workflow task until 18:38 handled by build `f26f32311e0b0f070c6b5e7ecfed4a7e` (worker identities `101@7b62636fbd10@`, `101@112c82239652@`), and the final task at 18:41 by a different container `100@fae80867d058@` with build `fab9cef363262225eeb590e200cfb2e1` — the 1.301.1 deploy. The earlier run of the same pipeline `…-66a4a11b-ebd8-49f9-94d4-e2b76c1d4adc/01a0f3a8-1bac-7eaf-81de-fa78ef684273` died the same way (build `9e17517d…` for 2690 tasks, then `fab9cef…` for the last one, same CRE220). Replaying `01a0f7d4-…`'s recorded history with the current code fails locally with a determinism error (`lookup failed for scheduledEventID to activityID` at event 2873), so the newly deployed build cannot faithfully replay the history produced by the previous build. Workers are started with `worker.Options{}` (`pkg/workflowengine/hooks/hook.go`), i.e. no `BuildID`/`UseBuildIDForVersioning`, so every deployment joins one unversioned worker set and Temporal hands an in-flight workflow's next task to whichever container polls. The #1441 “evidence by reference” fix is otherwise proven on this run: it reached the last of ~1399 steps without the earlier 4 MB `WORKFLOW_TASK_FAILED_CAUSE_GRPC_MESSAGE_TOO_LARGE` wedge.
- question: How should deploys stop breaking in-flight pipeline runs?
- options considered: (a) re-run and accept the hazard; (b) worker build-ID versioning (`worker.Options{BuildID: <release>, UseBuildIDForVersioning: true}`) so in-flight runs stay pinned to their build, keeping the previous deployment alive until drained; (c) `workflow.GetVersion` gates on every replay-visible change, preserving the old branch as `DefaultVersion`; (d) rework the aggregate so the root workflow is a thin, stable step loop and all fragile work lives in per-step/per-scenario child workflows.
- default risk: (a) any deploy whose code changes workflow commands kills every run in flight — a 4h+ conformance run nearly always straddles a deploy. (b) without a drain step in-flight runs stall instead of failing, so it needs ops tooling. (c) only helps if the gate exists before the change and does nothing for histories already recorded by ungated builds. (d) large change to `cmd/fcaf-pipeline-gen` and the generated aggregate.
- decision: pending human.
- follow-up: Confirm whether credimi.io only deploys release builds; two different builds (`f26f3231`, `fab9cef`) served the same run.

### 2026-10-01 - `pipeline_results.maestro_screenshots` caps a run at 99 files

- status: resolved (agent default; human may revisit per-step storage)
- owner: human maintainer
- context: `pb_migrations/1783679853_updated_pipeline_results.js` sets `maxSelect: 99` on `pipeline_results.maestro_screenshots`. `storePipelineStepScreenshotFiles` (`pkg/internal/apis/handlers/pipeline_step_screenshots_handler.go:131`) rejects an upload once `len(existing)+len(new) > maxSelect`, so the runner's `POST {runner_url}/credimi/execution-screenshots` relays `400 {"error":"upstream","message":"maximum 99 files allowed","name":"bad_request"}` and the mobile-automation child fails with `CRE310 Unexpected HTTP status code: expected 200, got 400`, although the Maestro flow passed. In run `…-e1adb96d-…/01a0f969-…` this is 187 of 241 failed mobile steps (every sampled one carried that body). Each mobile step uploads 1-3 screenshots, so a complete validation (~200 mobile steps) stores a few hundred per run.
- question: Where should a run's step screenshots live, and what bounds them?
- options considered: (a) raise `maxSelect` in a migration; (b) store screenshots per step/child run instead of appending to the pipeline_results record; (c) keep the cap and upload only a bounded subset.
- default risk: (a) bigger run record payload (a few hundred names, not thousands); (b) data-model, handler, UI and retention change; (c) silently loses evidence.
- decision: (a). `pb_migrations/1790915954_updated_pipeline_results_maestro_screenshots.js` raises `maxSelect` to 2000; regression test `TestStorePipelineStepScreenshotsBeyondNinetyNinePerRun`. Migration only, no workflow code change, so deploying it does not change in-flight pipeline replay.

### 2026-10-02 - FCAF aggregate inserted PID issuance ignored the scenario issuer fixture

- status: resolved (agent)
- owner: agent
- context: `cmd/fcaf-pipeline-gen` resolves `${fixture.*}` tokens of copied scenario steps, but `pidIssuanceSteps` inserts steps after that rewrite and emitted a literal `${fixture.issuer_url}/sessions`. The aggregate has no `runtime.fixture`, so `pipeline.ApplyFixture` substituted `DefaultIssuerURL` (`https://issuer-backend.eudiw.dev`), which answers `401`; every scenario declares `issuer_url: https://capture-wallet.credimi.io` (`201`). Run `…-e1adb96d-…`: all 34 issuance sessions failed, and 20 of its 56 Maestro failures are in those same scenarios.
- decision: Resolve the base URL through `rewriteString` with the scenario fixture; regenerated complete (59 URLs) and happy-flow (17 URLs) aggregates; `TestGenerateCompleteFCAFPipeline` asserts no `${fixture.issuer_url}` survives.
- follow-up: The stored pipeline record on credimi.io must be updated from the regenerated aggregate (`fcaf sync`). The remaining Maestro failures are wallet UI assertions against `eudiw-beta-wallet/2026-09-42-demo` (`"1" is visible`, `"Share" is visible`, …) and need flow triage.

### 2026-10-01 - Unify cached fetch-load wrappers

- status: open (deferred)
- owner: human maintainer
- context: `createCachedFetchLoad` already owns cache policy (`get` / `getOrError` / `invalidateAll` + shared `requestFetchRef`). Call sites wrap it inconsistently: canonify uses `getOrError`; hub by-id/by-path use throwing `get`; device adds `getCachedDeviceRecords` + `findCachedDeviceByPath` because the cache key is the whole list. Each module still exports its own `invalidate*Cache` for tests.
- question: Should every cached loader expose the same public surface, or keep domain-named getters and only share the factory?
- options considered: (a) leave as-is (domain APIs + factory); (b) export the same trio (`get` / `getOrError` / `invalidateAll`) from every cached module; (c) a thin per-module `{ get, invalidateAll }` returning the factory, with extra helpers (e.g. find-by-path) beside it.
- default risk: Forcing the factory trio onto domain modules either leaks Effect-cache vocabulary into hub/canonify/device APIs, or paper-over device’s list-then-find shape.
- decision: Do not unify in the current pipeline-composer work. Revisit later.
- follow-up: When unifying, pick one error surface (throw vs return `Error`) and decide whether device list cache stays a special case.

### 2026-10-05 - Owner-only record fields: hidden fields, owner writes, installer downloads

- status: open (agent default applied; human may revisit)
- owner: human maintainer
- context: Finding `credimi/recordsecrets/hide-only-filter-and-file-leak`. `credentials.secrets` and `use_cases_verifications.secrets` (text) and `wallet_versions.android_installer`/`ios_installer` (file) were `hidden: false`, so non-owners could filter and sort on them (`secrets~'DUMMY%'`), and non-downloadable installers downloaded without a token. `Record.Hide` in the enrich hooks only runs after the query. `pb_migrations/1791213262_protect_owner_only_fields.js` makes the four fields `hidden` and the installers `protected`; the enrich hooks `Unhide` them for owners; `HandleInstallerDownload` (`OnFileDownloadRequest`) gates non-downloadable installers.
- question: (1) PocketBase drops hidden fields from non-superuser create/update bodies, including `@request.body` in rules. Is restoring them in `OnRecordCreateRequest`/`OnRecordUpdateRequest` (`pbutils.LoadHiddenRequestFields`, after the collection rule passed) and moving the `wallet_versions` "at least one installer" check from the create rule into `HandleWalletVersionCreate` acceptable? (2) Installer downloads of non-downloadable versions now require a superuser, the internal admin key, or a member of the owner org (file token, auth token or user-scoped `Credimi-Api-Key`). A runner that downloads with a user key of another organization, for a published wallet's non-downloadable version, now gets `404`.
- options considered: (a) hidden fields plus owner-checked hooks (applied); (b) move secrets to an owner-only collection; (c) keep fields visible and reject filters by string matching (misses relation and back-relation paths).
- default risk: (2) can fail mobile runs that previously worked through `get-installer-md5-or-etag`, which still authorizes any caller when the wallet is published, independent of `downloadable`.
- decision: (a). (2) confirmed by the user: runners using another organization's user key must not get non-downloadable installers; admin-managed runners using the internal admin key keep access.
- follow-up: Resolved: `get-installer-md5-or-etag` (`authorizeWalletInstallerAccess`) now allows other organizations only for `downloadable` versions of published wallets; the internal admin key and owner members keep access.

### 2026-10-05 - Server App URL reaches `InternalHTTPActivity` through a process-level source

- status: resolved (superseded)
- owner: human maintainer
- context: Finding `credimi/workflowengine/internal-app-url-from-user-config`. `InternalHTTPActivity` attaches `CREDIMI_INTERNAL_ADMIN_KEY` to a URL built from workflow config. The fix strips `app_url`/`internal_app_url` from user-controlled config (`MergeConfigs`, the pipeline YAML fill-in in `PipelineWorkflow.Start` and `StartQueuedPipelineActivity`, the rerun body copy) and makes the activity reject any destination whose origin is not `CREDIMI_INTERNAL_APP_URL` or the PocketBase App URL. The activity has no app handle: `NewInternalHTTPActivity()` takes no arguments and is built in ~30 places, including static worker lists and the step registry.
- question: Should the server App URL reach the activity by dependency injection instead of `workflowengine.SetServerAppURLSource`, which `hooks.WorkersHook` calls once per process?
- options considered: (a) process-level source registered at worker startup, read on each call so App URL edits apply without restart (chosen); (b) constructor injection through every `NewInternalHTTPActivity` call site, worker list and registry factory; (c) accept only `CREDIMI_INTERNAL_APP_URL`, which production Compose requires but other deployments may not set.
- default risk: (a) is a package-level registration, which `AGENTS.md` discourages; a worker process that never calls `WorkersHook` and has no `CREDIMI_INTERNAL_APP_URL` fails closed with `no Credimi base URL is configured`. `credimi-extra` runner workers do not register this activity.
- decision: Superseded on 2026-10-06 by the approved plan that removed worker-to-Credimi HTTP: `InternalHTTPActivity`, `SetServerAppURLSource`, `CREDIMI_INTERNAL_APP_URL` and `internal_app_url` no longer exist. Workers reach Credimi data through typed activities that receive `core.App` (`activities.CredimiActivities(app)`), and `app_url` is written only by `workflowengine.WithAppConfig` from PocketBase Settings.
- follow-up: None. No admin-key sender targets Credimi anymore; the internal admin key is only sent to runners through `mobile-runner-http-request`.
