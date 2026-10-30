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

- status: resolved (superseded)
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

- status: resolved
- owner: human maintainer
- context: While writing the `credimi` skill, three `AGENTS.md` claims were checked against code and do not hold in this worktree: (1) "UI calls `POST /api/pipeline/start`" — no such route exists under `/api/pipeline` (`pkg/internal/apis/handlers/pipeline_handler.go`); the GUI (`webapp/src/lib/pipeline/queue.ts`) and the CLI both use `POST /api/pipeline/queue`, which starts device-less pipelines directly. (2) The external runner HTTP contract lists `POST {runner_url}/fetch-apk-and-action` and `/store-pipeline-result`; the code calls `/credimi/installer-action` and `/credimi/pipeline-result` (`pkg/workflowengine/pipeline/mobile_automation_hooks.go`). (3) `X-Api-Key` is documented for internal-admin routes; the middleware reads `Credimi-Api-Key` (`pkg/internal/middlewares/auth_api_key.go`, `pkg/internal/apis/handlers/api_key_service.go`); the same stale header appears in `docs/src/content/docs/software-architecture/scoreboard.md` and a comment in `pkg/conformancecatalog/store.go`.
- question: Update `AGENTS.md` (and the scoreboard doc/comment) to the current routes and header, or treat the documented names as the intended contract and change the code back?
- options considered: (a) fix the docs/comments to match code; (b) add compatibility aliases for the old runner endpoints and accept `X-Api-Key` as an alias; (c) add a `/api/pipeline/start` alias.
- default risk: Agents and docs keep describing endpoints that 404, and sibling `credimi-extra` implementers may build against the wrong runner paths.
- decision: (1) resolved 2026-10-06 by the user: option (a), `AGENTS.md` "Dynamic Pipeline Workflow" now documents `POST /api/pipeline/queue` as the only run entrypoint. (3) applied: `AGENTS.md`, the scoreboard doc and the `pkg/conformancecatalog/store.go` comment now name `Credimi-Api-Key`. (2) resolved 2026-10-07 by the user: option (a), the "External runner HTTP contract" in `AGENTS.md` now documents `POST {runner_url}/credimi/installer-action` and `POST {runner_url}/credimi/pipeline-result` with `log_path` and `platform`, matching `mobile_automation_hooks.go` and `credimi-runner` (`pkg/design/runner_server.go`).
- follow-up: None.

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
- follow-up: None. No admin-key sender targets Credimi anymore; the internal admin key is sent to runners through `mobile-runner-http-request` and, as the approved exception (2026-09-25), by the direct `POST {runner_url}/credimi/live-view` call in `pkg/internal/apis/handlers/pipeline_live_view_handler.go`.

### 2026-10-08 - Changing API rules of a PocketBase system collection (`orgAuthorizations`)

- status: open (agent default applied; human may revisit)
- owner: human maintainer
- context: Finding `credimi/pb-rules/orgauthorizations-public-list`. `orgAuthorizations` had `listRule`/`viewRule` `""`, so anonymous callers and other tenants listed every organization's members. `orgAuthorizations` is `system: true` (from `migrations/pb_schema.json`), and PocketBase rejects API rule changes and `system` flag changes on system collections in validated saves (`validation_collection_system_rule_change`, `validation_collection_system_flag_change`). `pb_migrations/1791465595_restrict_org_authorizations_read.js` sets both rules to `user.id = @request.auth.id || (same-org member)` with `app.saveNoValidate`. The join page (`webapp/src/routes/my/organizations/join`) found joinable organizations through other tenants' memberships; it now excludes the caller's own organizations instead, so organizations without any membership are also listed.
- question: Is `app.saveNoValidate` in a migration the accepted way to change rules of the template's system collections (`orgAuthorizations`, `orgRoles`, `orgJoinRequests`, `org_invites`, ...)?
- options considered: (a) `saveNoValidate` on the rule fields only (applied); (b) raw SQL update of `_collections` (also skips validation, bypasses the collection model); (c) drop `system` from these collections (needs raw SQL too, and removes the PocketBase admin-UI guard).
- default risk: rule syntax is not validated on save; `TestOrgAuthorizationsReadRulesScopeToMembers` exercises the rules through the records API, so a broken rule fails the suite.
- decision: (a).
- follow-up: None unless the maintainer prefers another approach.

### 2026-10-08 - `z_test_collection` accepts anonymous writes

- status: open
- owner: human maintainer
- context: Finding `credimi/pb-rules/news-public-create`. `news` was fixed by `pb_migrations/1791464765_restrict_news_create.js` (`createRule` null). `z_test_collection` (`migrations/pb_schema.json`, id `ktjgpqf146ss2ia`) has the same root pattern: `createRule`, `updateRule` and `deleteRule` are `""`, so anyone can create, edit or delete its rows. It is a fixture collection used only by the unguarded webapp `/ui-tests/*` pages and `webapp/e2e/nru/collection-manager.spec.ts` (userA creates a record). It is `system: true`, so PocketBase rejects any rule change through `app.save` (`System collection API rule cannot be changed`).
- question: Restrict `z_test_collection` writes, and how?
- options considered: (a) superuser-only rules saved with `app.saveNoValidate` (new migration pattern; breaks the `/ui-tests` write pages and the e2e create); (b) logged-in create plus the owner/authorization expression of `listRule` for update/delete, saved with `app.saveNoValidate`; (c) remove the collection and `/ui-tests` from production builds; (d) leave as is.
- default risk: Anonymous writes stay open on a fixture collection that exists in every deployment; impact limited to junk test rows.

### 2026-10-14 - FCAF: an `EUDI_forbidden` source reports `not_applicable`, not a failure

- status: resolved (maintainer decision 14/10/2026); superseded 21/10/2026 by the entry "every `EUDI_forbidden` source reports only `not_applicable`, without a Wallet interaction" below
- owner: human maintainer
- context: Some FCAF wallet-solution/relying-party sources ask the Wallet to accept a feature the FCAF relevancy legend marks `EUDI_forbidden` ("Requirement is explicitly excluded or prohibited for usage with EUDI Wallets"): `WS_RP_MS_ProtocolMessages__038` and `WS_RP_SM_RpIntegrity__003` (a `redirect_uri:` Client Identifier Prefix; an unsigned request processed without signature validation) and `WS_RP_UC_Presentation__003` (a scope-based presentation request). A conformant EUDI Wallet must not support any of them, so the source's own expectation can never hold for the SUT, and reporting it as a pass or as a failure both misstate conformance. Before this decision there was no way to express "the source targets a different profile"; the only similar marker was `sdjwt.presentation_digest_algorithm_unsupported` reporting `not_applicable`.
- question: How should a source whose own relevancy is `EUDI_forbidden`, and whose expected result is acceptance of the forbidden feature, be reported against an EUDI Wallet?
- options considered: (a) keep asserting the acceptance and report `fail` (a permanent, meaningless failure); (b) mark the whole test `fcaf.skipped` (drops the request-shape evidence the source still exercises); (c) add a `fcaf.not_applicable` validator that keeps the source's request-shape assertions and replaces only the wallet-acceptance expectation with a reason, returning `StatusNotApplicable`.
- default risk: (c) could hide a real breach if it were applied to a source whose relevancy is not literally `EUDI_forbidden`, or used to excuse a missing capability instead of a deliberately excluded feature; the engine must also not let `not_applicable` mask a failed request-shape assertion.
- decision: (c). `fcaf.not_applicable` (`pkg/fcaf/validators/generic.go`, registered in `DefaultRegistry`) mirrors `fcaf.skipped`, requires a `reason` param and returns `StatusNotApplicable`; the engine folds `pass + not_applicable` into `not_applicable`, while any `fail`/`error`/`blocked`/`inconclusive` still dominates, and reports count it separately from `skipped`. Applied to `WS_RP_MS_ProtocolMessages__038`, `WS_RP_SM_RpIntegrity__003` and `WS_RP_UC_Presentation__003`, each with the reason "source relevancy is EUDI_forbidden: an EUDI wallet must not support <feature>". Their request-shape and visual-evidence assertions are kept.
- follow-up: Only sources whose FCAF relevancy literally reads `EUDI_forbidden` may use it; `EUDI_generic | EUDI_forbidden` alone is not enough. That lock governs this ground only - the other `not_applicable` ground (the SUT's own capabilities refute the source's precondition) is the 2026-10-14 entry below and is not restricted to `EUDI_forbidden` sources (e.g. `WS_RP_MS_ProtocolMessages__011`/`018`, `WS_RP_MS_Metadata__134`). The decision's "their request-shape and visual-evidence assertions are kept" holds for `WS_RP_MS_ProtocolMessages__038` and `WS_RP_SM_RpIntegrity__003`, but not literally for `WS_RP_UC_Presentation__003`, `020` and `112`: those sources' request shapes (a profile-defined scope value; an `openid_federation` `trust_chain`) have no fixture, so their unused evidence bindings were dropped (14/10/2026 self-review). Where the shape is deliverable it is asserted: `040` is rebound to the plain `redirect_uri:` Client Identifier and `060` to a session carrying the `redirect_uri` parameter. Locked by `TestFCAFNotApplicableValidator` (`pkg/fcaf/validators/registry_test.go`) and `TestEngineReportsForbiddenFeatureAsNotApplicableWithReason` (`pkg/fcaf/engine/executor_test.go`).

### 2026-10-14 - FCAF: `not_applicable` when the SUT cannot meet a source precondition, `fcaf.skipped` when the harness cannot build it

- status: resolved (maintainer decision 14/10/2026)
- owner: human maintainer
- context: Two different situations were being collapsed into one status. (1) The source's precondition does not hold for the wallet under test, so the test is not about this SUT: `WS_RP_MS_ProtocolMessages__151` requires a Wallet that does not support `vc+sd-jwt` and expects `vp_formats_not_supported`, but the reference wallet supports `mso_mdoc` (and answers the request), so the premise is factually false here. (2) The harness/Capture cannot construct the precondition at all, so the test cannot be run: `WS_RP_IA_MainInteraction__028` (Capture can issue no non-selectively-disclosable credential), `WS_RP_IA_MainInteraction__056` (the deployed verifier certificate has no `dNSName` SAN), `WS_RP_IA_Supportive__006` (a deliberately resource-exhausted wallet device), `WS_RP_SM_IssuerIntegrity__012` (a revoked mdoc MSO fixture), `WS_RP_UC_Presentation__004` (a second independently controllable device), and the transaction-data positive cases 028/135/154-157 (no supported transaction-data type fixture).
- question: Which status should each situation produce?
- options considered: (a) `fcaf.skipped` for both; (b) `not_applicable` for both; (c) `not_applicable` when the SUT's own capabilities refute the source's precondition, `fcaf.skipped` when the harness cannot build the precondition.
- default risk: `skipped` for situation (1) parks a runnable test as "not run" although its premise is provably false for this SUT; `not_applicable` for situation (2) publishes as "not applicable" a test nobody has executed, and misreports a missing fixture/permission as a profile exclusion.
- decision: (c). `WS_RP_MS_ProtocolMessages__151`'s `oid4vp.error_response_required` assertion carries `not_applicable_when_wallet_supports_format: mso_mdoc`, so it returns `not_applicable` once the wallet advertises the format the source assumed absent. Preconditions the harness cannot construct use `fcaf.skipped` with the missing fixture or permission as the reason.
- follow-up: A `fcaf.skipped` reason must name the unbuilt precondition, never the wallet's behaviour (see "Rule: encode the source, not what the reference wallet can pass" in `pkg/fcaf/MEMORY.md`).

### 2026-10-14 - FCAF: embedding the Debian iso-codes ISO 3166-2 list (LGPL-2.1-or-later) in an AGPL repository

- status: resolved (maintainer decision 14/10/2026)
- owner: human maintainer
- context: The passed-test audit of run `4t9hxzgxlf` found that `sdjwt.claim_country_subdivision` and `mdoc.element_country_subdivision` only checked the shape `^[A-Z]{2}-[A-Z0-9]{1,3}$` and that the prefix equalled the issuing country, so a well-formed but non-existent subdivision (e.g. `IT-ZZ`) passed, although the sources (`config_templates/fcaf_sources/.../CredentialMetadata/WS_RP_DM_Credentialmetadata_Issuingjurisdiction_PID_IETF-sd-jwt-vc_003.md:37` and `_PID_ISO-mdoc_003.md:37`) require the code to be "listed in ISO 3166-2:2020, 6.1 (according to the corresponding country code)". Closing the gap needs the subdivision code list. `pkg/fcaf/validators/iso3166_2.go` now embeds the code elements only (5046 codes, 200 countries) derived from Debian's `iso-codes` package (`data/iso_3166-2.json`, 2024-01-14), with SPDX attribution to the iso-codes authors and Forkbomb BV, and `LICENSES/LGPL-2.1-or-later.txt` was added. The repository already embeds the far smaller ISO 3166-1 alpha-2 table inline in `pkg/fcaf/validators/value.go` with no separate attribution, but this is a third-party dataset under LGPL-2.1-or-later carried into an AGPL-3.0-or-later repository; `REUSE.toml` has no entry for `pkg/fcaf/validators`, and this backlog had no policy on ISO data before.
- question: Is embedding the ISO 3166-2 code list derived from Debian iso-codes under SPDX `LGPL-2.1-or-later` (`LICENSES/LGPL-2.1-or-later.txt`) acceptable in this AGPL repository, or should the subdivision check use another source, or fall back to the shape-only check?
- options considered: (a) keep the derived iso-codes dataset in its own file under LGPL-2.1-or-later with its own SPDX header and license text (currently in the working tree); (b) derive the list from a permissively licensed source (e.g. a government-published ISO 3166-2 dataset or a permissive package) to avoid a second license in the tree; (c) ship only the shape check and record the un-encoded expected result in `ASSERTION_REVIEW_BACKLOG.md` and the two inventory rows.
- default risk: (a) mixes an LGPL-2.1-or-later data file into an AGPL-3.0-or-later repository; the LGPL terms attach to the file and its distributed form, and a reviewer must confirm that the codes-only derivation and the attribution satisfy them. (b) risks a less accurate or less maintained list. (c) leaves a source requirement unasserted, so an invalid subdivision still passes.
- decision: option (a). LGPL-2.1-or-later is compatible with AGPL-3.0-or-later (LGPL 2.1 §3 allows relicensing under GPL-2.0-or-later, hence GPL-3.0, which combines with AGPL-3.0). Keep the codes-only file with its SPDX attribution and `LICENSES/LGPL-2.1-or-later.txt`; `reuse lint` passes.

### 2026-10-15 - Run page: `403` for another organization's pipeline run discloses that the run exists

- status: open (agent default applied; human may revisit)
- owner: human maintainer
- context: Opening `/my/tests/runs/<workflow_id>/<run_id>` for a run of another organization (e.g. from a push notification delivered to a browser now signed in as another user) made `GET /api/my/workflows/{workflowId}/runs/{runId}` answer `404` (Temporal `NotFound` in the caller's namespace) and the run page turned every fetch error into a `500`. The user asked to be told that the run belongs to another account instead.
- question: May the endpoint tell a caller that a workflow/run ID pair belongs to another organization?
- options considered: (a) on Temporal `NotFound`, look the pair up in `pipeline_results` and answer `403` `workflow belongs to another organization` when its owner is another organization (applied; `pipelineRunOwnedByAnotherOrganization` in `pkg/internal/apis/handlers/workflows_handlers.go`); (b) keep `404` and have the page only suggest that the run may belong to another account; (c) carry the recipient user in the push payload/URL and compare it client-side.
- default risk: (a) confirms to any authenticated user that a given `(workflow_id, run_id)` exists in some other organization. It discloses no owner, name or content, and run IDs are random UUIDs, so guessing is impractical; non-pipeline runs (no `pipeline_results` row) still answer `404`.
- decision: (a), pending maintainer confirmation.
- follow-up: If disclosure is not acceptable, drop the lookup and switch the run page message to option (b).
### 2026-10-20 - Animated SVG illustrations on marketing pages

- status: open
- owner: human maintainer
- context: `webapp/static/pages/why-credimi/*.svg` are illustrations for the competitor comparison page. The user asked for animation. AGENTS.md says motion is functional, not decorative, and forbids autoplay and animations over 250ms in product UI.
- question: Do static marketing pages under `webapp/static/pages/` get an exception to the product-UI motion rules?
- options considered: keep illustrations static; play the entrance once with no loop; subtle looping animation on marketing pages only.
- default risk: if the exception is not documented, agents will copy looping animation into product UI.
- decision: on 20/10/2026 the user chose subtle looping animation for this page: CSS and SMIL inside the SVG, low contrast, and switched off under `prefers-reduced-motion`.
- follow-up: confirm the exception and move it into the Brand And Design section of AGENTS.md, or reject it and make the SVGs static.

### 2026-10-21 - FCAF: every `EUDI_forbidden` source reports only `not_applicable`, without a Wallet interaction

- status: resolved (maintainer decision 21/10/2026)
- owner: human maintainer
- context: The 14/10/2026 decision applied `fcaf.not_applicable` only to sources that expect the Wallet to accept the forbidden feature, and kept their request-shape and visual-evidence assertions, so those tests still drove the Wallet. Of the eleven wallet-solution/relying-party sources whose relevancy reads `EUDI_generic | EUDI_forbidden`, `WS_RP_IA_Metadata__014` and `WS_RP_MS_Metadata__113` stayed `fcaf.skipped`, and `WS_RP_MS_Metadata__110` and `WS_RP_SM_RpIntegrity__002` asserted a rejection, although `110`, `113` and `002` expect the Wallet to reject, which is what an EUDI Wallet without the feature does.
- question: Should every `EUDI_forbidden` source report `not_applicable`, and should those tests still exercise the Wallet?
- decision: every source whose relevancy contains `EUDI_forbidden` reports only `not_applicable`, whatever its expected result: `WS_RP_IA_MainInteraction__060`, `WS_RP_IA_Metadata__014`, `WS_RP_MS_Metadata__110`, `112`, `113`, `WS_RP_MS_ProtocolMessages__020`, `038`, `040`, `WS_RP_SM_RpIntegrity__002`, `003`, `WS_RP_UC_Presentation__003`. Each definition carries one `fcaf.not_applicable` assertion and no other. They are owned by `scenarios/fcaf-wallet-solution-relying-party-eudi-forbidden.yaml`, which has only the `fcaf-validation` step: its single evidence source `pipeline.eudi-forbidden` is the literal output `relevancy: EUDI_forbidden`, so neither the Verifier nor the Wallet is contacted for these tests.
- follow-up: The Wallet steps these tests used stay, because other tests own them. A new `EUDI_forbidden` source joins the `eudi-forbidden` scenario with the same single assertion. The second `not_applicable` ground (the SUT's capabilities refute the source's precondition) is unchanged.

### 2026-10-22 - FCAF attestation provider: attributing the Authorization Request from Capture's global request ledger

- status: resolved (upstream credimi-capture-wallet `759bab5`, 26/10/2026)
- owner: human maintainer
- context: `WS_AP_IA_MainInteraction_010a`, `_010b` and `WS_AP_MS_ProtocolMessages_039a` check the Wallet's Authorization Request (`GET /issuers/{id}/authorize`). The Wallet sends it from Chrome and Capture records it only in the global `GET /oid4vci/requests` ledger with `session_id: null`; the issuance session holds the PAR, callback, token and Credential Requests. The ledger has no filter and keeps the last 1000 requests of every caller.
- question: May an FCAF verdict rest on a request attributed to the session from the global ledger, and may the stored FCAF report copy that whole ledger?
- options considered: (a) attribute it in the validator: same issuer configuration, no other session, between the session's PAR and its next request, narrowed by the PAR `state`, inconclusive when ambiguous (implemented in `oid4vci.wallet_request` `ledger_fallback`); (b) mark the three tests `fcaf.skipped` until Capture ties `/authorize` to the session; (c) change Capture to resolve the `request_uri` and record the request on the session.
- default risk: (a) concurrent traffic on the same issuer configuration within seconds can make the attribution ambiguous, and enough later traffic can evict the PAR's time from the ledger; both report inconclusive, never pass. The report also stores other callers' redacted requests and forwarded IP addresses. (b) leaves three runnable tests unexecuted.
- decision: (a) as an interim, pending the maintainer's confirmation; (c) is the durable fix and needs a credimi-capture-wallet change.
- resolution: (c) landed upstream: Capture attributes `/authorize` by the PAR `request_uri`. On 26/10/2026 `ledger_fallback` and the `request_ledger` evidence were removed; the tests read the session's own requests.

### 2026-10-23 - Server rendering for public pages, public origin and AI crawler policy

- status: open
- owner: human maintainer
- context: The SEO and AI-discovery work found every public page outside `/pages` served as an empty client-side shell (root layout `ssr = false`), so crawlers that do not run JavaScript, including most AI crawlers, saw no title or content. Requests reach the webapp through the PocketBase proxy with `Host` set to the internal UI address and no forwarded host or protocol, so request-derived URLs point at the internal host.
- question: (1) May the `(public)` route group render on the server? (2) Is a build-time `PUBLIC_SITE_URL`, defaulting to `https://credimi.io`, the right source for canonical and sitemap URLs? (3) Should AI training crawlers (GPTBot, ClaudeBot, Google-Extended, CCBot, …) be allowed, not only AI search agents?
- options considered: (1) SSR for `(public)` only (applied), SSR app-wide, or prerendering only; (2) `PUBLIC_SITE_URL` (applied), PocketBase Settings Application URL (admin-only, not readable by anonymous SSR), or forwarding `X-Forwarded-Host`/`-Proto` from the Go proxy and configuring the adapter's host and protocol headers; (3) allow all AI crawlers (applied, the user asked for content to reach AI models) or allow search and user agents only.
- default risk: (1) a public component that touches browser-only APIs while rendering returns 500, or crashes the server process when a promise rejects unhandled (the realtime subscription did, fixed); (2) a self-hosted build that does not set `PUBLIC_SITE_URL` canonicalizes to credimi.io; (3) content is used for model training.
- decision: applied as above on the user's request "all my content suggested by AI harnesses and models".
- follow-up: confirm, then keep the "Public Pages, SEO And AI Discovery" section of AGENTS.md as the canonical rule.

### 2026-10-26 - FCAF attestation provider: Capture deployment the ws_ap scenarios target

- status: resolved (user decision 26/10/2026)
- owner: human maintainer
- context: The ws_ap task asked for runs against `beta-capture-wallet.credimi.io`, but on 26/10/2026 beta served none of the `759bab5` issuance contract (`evidence_mode`, `offer_scheme`, `offer_mutation`, `issuance_response_scenario`, `GET /sessions/{id}/requests`), while production `capture-wallet.credimi.io` served all of it.
- question: Which host do the ws_ap scenarios and emulator runs use?
- options considered: production; wait for beta.
- default risk: on beta every ws_ap scenario is refused or silently loses its evidence.
- decision: production `capture-wallet.credimi.io`, the existing scenario's host.

### 2026-10-26 - FCAF attestation provider: offer-by-value tests and the eu-eaa-offer scheme

- status: resolved (user decision 26/10/2026)
- owner: human maintainer
- context: `WS_AP_MS_ProtocolMessages_033a` (valid Credential Offer by value), `WS_AP_MS_ProtocolMessages_033b_UF` (offer without `credential_issuer`) and `WS_AP_MS_ProtocolMessages_033c_UF` (offer without `credential_configuration_ids`) deliver their offer with `eu-eaa-offer://`, as their Technology sections and the task specify. The reference Wallet 2026.09.42 registers `haip-vci` and `openid-credential-offer` but not `eu-eaa-offer`, so it is never invoked and the three tests report blocked (`WS_AP_IA_Engagement_007b` reports the missing scheme as a failure). The two UF sources' precondition only says "a custom URL scheme"; with `haip-vci://` the reference Wallet shows its error screen and sends no request for both malformed offers (probed 26/10/2026).
- question: Should `WS_AP_MS_ProtocolMessages_033b_UF` and `WS_AP_MS_ProtocolMessages_033c_UF` (and possibly `WS_AP_MS_ProtocolMessages_033a`) use a scheme the Wallet under test registers, so the offer-handling behaviour is evaluated even when `eu-eaa-offer://` is missing?
- options considered: (a) keep `eu-eaa-offer://` (implemented; blocked on the reference Wallet); (b) use `haip-vci://` for the UF tests; (c) add a second session per test with a fallback scheme.
- default risk: (a) leaves the three tests unevaluated on any Wallet without `eu-eaa-offer://`; (b) and (c) evaluate the offer under a scheme the sources' Technology does not name.
- decision: (a), keep `eu-eaa-offer://`; a Wallet without the scheme leaves the three tests blocked.

### 2026-10-26 - FCAF attestation provider: tests after an issuer rejection report blocked

- status: resolved (user decision 26/10/2026)
- owner: human maintainer
- context: Credo answers a PAR carrying a JAR `request` with `400`, and a `jwt` proof with only a key attestation with `400 invalid_proof`, so a conforming Wallet's later requests are never sent. The ws_ap validators report a missing request, Credential Request or issued credential as blocked when an earlier request of the session got an error status from the issuer, naming that request and error, and as failed otherwise. For a JAR Wallet this turns every test after the PAR into blocked, for example `WS_AP_IA_MainInteraction_010a`, `WS_AP_IA_MainInteraction_010b`, `WS_AP_IA_MainInteraction_010c`, `WS_AP_IA_Supportive_011a`, `WS_AP_MS_ProtocolMessages_039a`, `WS_AP_MS_ProtocolMessages_039b`, `WS_AP_IA_MainInteraction_030a`, `WS_AP_IA_MainInteraction_030b`, `WS_AP_MS_ProtocolMessages_050a` to `WS_AP_MS_ProtocolMessages_050e` and `WS_AP_IA_MainInteraction_031a`.
- question: Is "blocked after any issuer error status" the right rule, given that it also turns a Wallet-caused `4xx` (for example a malformed token request) into blocked for the later tests?
- options considered: (a) any issuer `4xx`/`5xx` before the missing step (implemented); (b) only known Capture/Credo limitations (JAR, key-attestation-only proof); (c) always fail.
- default risk: (a) can hide a Wallet defect behind blocked, though the message names the rejected request; (b) needs a list kept in step with Credo; (c) reports Credo limitations as Wallet failures.
- decision: (a), any issuer error status before the missing step reports blocked, naming the rejected request.

### 2026-10-26 - FCAF attestation provider: reading unattributed ledger requests as the Wallet's

- status: resolved (user decision 27/10/2026)
- owner: human maintainer
- context: `WS_AP_MS_ProtocolMessages_033b_UF` (no metadata request) and `WS_AP_IA_Supportive_011b_UF` (no Authorization Request without `request_uri`) can only see the forbidden requests among `GET /sessions/{id}/requests` `unattributed_requests`, which hold every caller's requests in the window. The validators filter by issuer host and path and treat a match as the Wallet's.
- question: Is the "otherwise idle deployment" assumption acceptable for these two verdicts?
- options considered: (a) assume an idle deployment and say so in the result (former implementation); (b) report inconclusive when a forbidden path matches an unattributed request; (c) a dedicated Capture deployment for these runs.
- default risk: (a) another client's metadata fetch or `/authorize` within the window fails a conforming Wallet; (b) a Wallet that does send the forbidden request is reported inconclusive, never failed, by these two checks; (c) needs a separate deployment.
- decision: (b). A forbidden request attributed to the session still fails; a match among unattributed requests reports inconclusive, and no match passes (`unattributedMatch` in `pkg/fcaf/validators/oid4vci_ws_ap.go`).

### 2026-10-27 - Action author model provider for local first tests

- status: resolved (user decision 27/10/2026)
- owner: human maintainer
- context: phase 0 still needs a real-model authoring run, but no separate Anthropic API key is available; the user has a ChatGPT/Codex subscription and approved local-only testing.
- question: May the action-author worker use OpenAI's documented ChatGPT-plan OAuth and Responses API for local tests?
- options considered: (a) worker-side ChatGPT OAuth provider using the existing harness Responses activities; (b) a Codex CLI subprocess; (c) reuse Codex's cached credentials; (d) defer until an API key is available.
- default risk: (b) replaces the harness-owned model/tool loop; (c) mixes client registrations and token refresh ownership; (d) leaves phase 0 unproven. The selected flow is restricted to local/open-source use and requires streamed Responses; tools use `additional_tools` developer inputs to preserve the harness's bare-name tool identities.
- decision: (a), local only. No production/hosted multi-user authorization is implied; usage and cost evaluation remain deferred until the implementation is stronger.

### 2026-10-28 - Action author phase 1: authenticated organization binding

- status: resolved (user selected authenticated Credimi start on 28/10/2026)
- owner: human maintainer
- context: phase 0 authoring/replay has passed and the harness is upgraded to 0.8.0. The user requested warning cleanup followed by phase 1. The original proposal specifies Credimi wallet-version/APK selection and approval-gated, owner-scoped Go persistence; it does not define how the local harness session acquires a trusted organization. The current local worker uses the `default` namespace.
- question: Should phase 1 add an authenticated Credimi session-start endpoint and use the caller's authorized organization namespace, retaining the harness UI for review and approval, or remain a local single-user API-key bridge?
- options considered: (a) authenticated Credimi start endpoint, organization namespace, Go activities holding `core.App` for owner-scoped resolution and persistence (recommended; requires a new API contract and namespace integration); (b) local Python activities calling the existing PocketBase API with a worker-held user API key (no new API/namespace change, but departs from the proposed Go persistence boundary and is not a hosted multi-user design).
- default risk: accepting a model- or client-supplied owner in the unauthenticated local harness could authorize writes to the wrong organization. Neither an internal-admin key nor a model-selected owner is a safe substitute for caller authorization.
- phase 1 invariants: selected existing wallet version and APK provenance; literal supplied test PIN in accepted YAML; two successful clean-state replays before saving; human approval of the exact accepted YAML; unpublished creation; explicit approval before replacement. Runner reservation/execution/final cleanup remain phase 2, and embedded Credimi chat remains phase 3.
- decision: (a). Authenticated `POST /api/action-author/sessions` derives the actor and authorized organization from Credimi; Python workers and the review UI use that organization namespace. Go `core.App` activities resolve the existing wallet version and save only the human-confirmed, twice-replayed candidate. The local harness rejects direct session creation. Short-lived, build-bound installer grants avoid giving the Python worker a Credimi API key.
- persistence: new actions are unpublished; replacing an existing action requires its record ID and current code SHA256. Hidden provenance binds the build, APK, candidate and workflow/run; native code edits invalidate it. PocketBase strips non-superuser writes to hidden fields before request hooks, so a native update can answer 200 without changing provenance.
- verification: isolated Credimi/Temporal/UI smoke exercised a real APK download and metadata inspection, simulated installation and two replays, then human-only saving and browser reload. Physical-device execution was not repeated. Runner lifecycle and cleanup are intentionally unchanged.

### 2026-10-28 - Action author phase 1 validation: existing x/net vulnerability

- status: resolved (user approved the security upgrade on 28/10/2026; `make lint` passes)
- owner: human maintainer
- context: phase 1's full Go tests pass, but `make lint` stops at govulncheck on GO-2026-6860 in the unchanged `golang.org/x/net v0.60.0`; the reported call path is `activities.extractOutputRules` → `html.Parse`. The advisory identifies `v0.62.0` as fixed.
- question: Should the public dependency upgrade be a separate maintenance change or explicitly included in the action-author branch?
- options considered: (a) upgrade and validate x/net separately; (b) extend this branch with the upgrade. Do not suppress the advisory or change the HTTP activity merely to bypass the check.
- default risk: the existing HTML parser has an availability vulnerability; phase 1 must not be described as having a green aggregate lint gate until it is resolved.
- decision: (b). Upgrade `golang.org/x/net` to `v0.62.0` and the related x/* versions required by its module graph. Preserve Forkbomb dependencies. Module hygiene, module verification, vet, reachable-code vulnerability scanning and golangci-lint all pass after the upgrade.

### 2026-10-29 - Action author code review: open conventions and deferred findings

- status: resolved (user decisions 29/10/2026)
- owner: human maintainer
- context: a full review of `agents/action-author` and the branch's Go code fixed the bugs listed in the review report. The items below are conventions without a documented rule, or findings deliberately left unfixed.
- questions:
  1. Temporal-client failures in handlers: `POST /api/action-author/sessions` answers `503 temporal_unavailable`, `conformance_check_handlers` answers `500`. Which is the repository convention?
  2. Binary responses in route definitions: the installer route is the first user of `ResponseContentType` with an empty `ResponseSchema`. Is that the pattern for future binary endpoints?
  3. Installer grants are signed with a key derived from the `users` auth-token secret, so rotating it invalidates outstanding (10-minute) grants, and a grant may be used any number of times within its lifetime. Is that acceptable, or should grants be single-use (server-side state)?
  4. The model's requested action name is kept for the session until another `set_target` names a new one. Should a target change without a name reset it to `Onboarding`?
  5. `maestro_inspect_screen` can run before `set_target`, although device operations are otherwise refused until a target is chosen. Enforcing it needs a new workflow patch id.
  6. The harness publishes tool inputs (`set_target` parameters, `fill` text) unmasked to the event stream; masking them needs an upstream harness change.
  7. Anthropic replies truncated by `max_tokens`/`pause_turn` raise before usage reaches the per-message token budget, so those tokens are not counted.
  8. Should the README's historical prose (version-upgrade notes, dated acceptance narratives) move to a changelog?
- options considered: keep current behaviour (default) or change per item; none is a regression introduced by the review.
- default risk: items 3 and 6 are the only security-relevant ones; both are limited to the local loopback authoring setup and a dedicated test PIN.
- decisions:
  1. `503`, domain `temporal`, reason `temporal_unavailable`, is the convention whenever a Credimi route-group handler cannot reach Temporal. Existing endpoints moved to the shared `temporalUnavailableError()` in #1551; the action-author session endpoint uses it too, and AGENTS.md documents the rule.
  2. Keep `ResponseContentType` with an empty `ResponseSchema` as the pattern for binary responses.
  3. Keep reusable 10-minute grants signed from the `users` token secret; revisit single-use grants when runners download from machines outside local control.
  4. Keep the requested name for the whole session.
  5. Fixed: `maestro_inspect_screen` refuses to run before `set_target`, behind patch id `action-author-inspect-requires-target-v1`.
  6. Accepted: Credimi test PINs (typically `123456`) are not secrets, so unmasked tool inputs in the harness stream are acceptable. Masking was removed entirely.
  7. Fixed: a Claude reply truncated by `max_tokens`/`pause_turn` becomes a normal reply without its unfinished tool call, so its tokens count toward the budget and the turn's completed work is retained; `Continue` resumes.
  8. Done: historical README prose moved to `agents/action-author/CHANGELOG.md`.

### 2026-10-29 - Action author on iOS simulators (local Mac mode)

- status: closed (30/10/2026): the local mode is deleted (plan step 13); its iOS rules, adapted to runner devices, live in `credimi-runner/agent/README.md`, section "iOS simulator". Items 4 and 5 are superseded there: the device type decides the platform, and the reset's app poll checks the clone.
- owner: human maintainer
- context: the local action author drove only adb phones. The user asked for the same agent on iOS simulators, before the runner plan (`ACTION_AUTHOR_ON_CREDIMI_RUNNERS_PLAN.md`), keeping agent-device for exploration. A spike on a clone of the Mac's `credimi` simulator showed agent-device (0.21.32) and Maestro (2.11/2.12) driving one simulator at once, `simctl clone`+boot in about 11 s, the first Maestro run on a fresh clone in about 37 s, and that Maestro's `back` is a no-op on iOS.
- decisions:
  1. `ios_installer` is a zipped simulator `.app` (user). The worker extracts it with `ditto` and accepts exactly one `.app`, at the top level or under `Payload/`, like the runner's `unzipIOSAppIfNeeded`.
  2. Reset by simulator clone, as pipelines do (user): no new `credimi-extra` activity. The worker clones the base simulator into a session template with the app installed, and every reset or replay boots a fresh clone of the template.
  3. Provenance and the build contract name the installer (user): `apk_sha256` became `installer_sha256`, plus `platform`. No migration: `authoring_provenance` is JSON and the branch is unmerged.
  4. Agent decision: `POST /api/action-author/sessions` requires `platform` (`android`|`ios`), chosen on the start page; the worker refuses a session for the other platform before resolving the build. Alternative: infer the platform from the worker, which Credimi cannot see at start.
  5. Agent decision: on iOS the build check binds the template to the bundle ID and zip digest the worker installed and checks the clone still has the bundle; an installed `.app` cannot be hashed back to the zip, unlike Android's APK SHA256.
- default risk: agent-device has no iOS device allowlist, so iOS pinning relies only on the wrapper overwriting `udid`. Clones of a killed worker are removed at the next worker start, not before.
- verification (29/10/2026): worker on the Mac (iOS 26.5 clone of `credimi`) against local Credimi and Temporal through an SSH tunnel, version `forkbomb-bv-andrea/eudiw-beta-wallet/2026-10-43-demo`, Claude `claude-sonnet-5-5`. agent-device explored the onboarding, the first submission passed both replays (48.2 s, 39.3 s), and Accept saved unpublished action `7ouelbtgpwgd5kn` with provenance `platform: ios`. The worker deleted its template and clone on stop.
- follow-up: `ACTION_AUTHOR_ON_CREDIMI_RUNNERS_PLAN.md` now covers both platforms on runners (device type decides the platform; the run's clone becomes the template; agent source embedded in the runner binary for native macOS).

### 2026-10-30 - Action author on runners: plan details resolved from code

- status: open (agent defaults applied; human may revisit)
- owner: human maintainer
- context: steps 2-6 of `ACTION_AUTHOR_ON_CREDIMI_RUNNERS_PLAN.md`, done while PR #1553 is still open (plan contingency: no `WaitTimeoutSeconds`, no early device tracking). Some plan details disagree with the code they reference.
- questions and options considered:
  1. The plan has `processStep` call `PrepareMobileDevice`, which fails without an installed package (`missing installed package for <version>`). Pipelines with `version_id: installed_from_external_source` install nothing, so that check would fail them, and `processStep` publishes `setted_devices` to run data between the device setup and the install, which decides whether a failed install is cleaned. Options: (a) `processStep` and `PrepareMobileDevice` share an unexported core with a tracking callback, and only `PrepareMobileDevice` adds the package and iOS installer-path checks; (b) apply the checks to pipelines too.
  2. The plan names the session results `"completed"`/`"failed"`; pipelines report `success`/`failed`/`canceled` (`pipeline.go`). Options: reuse the pipeline values, or add new ones the semaphore never saw.
  3. The plan says to map queue-limit errors of `POST /api/action-author/sessions` "the way `enqueuePipelineRun` does (`isQueueLimitExceeded`, `429`)", but `enqueuePipelineRun` answers `409`, domain `queue_limit`, reason `queue limit exceeded`. Options: `409` like the queue endpoint, or `429` as written.
  4. The plan does not say how other enqueue failures answer. Options: keep the endpoint's existing `503 start_failed` (`cannot start authoring session`) and cancel the ticket best-effort, as `enqueuePipelineRun` rolls back ambiguous failures; or a new reason.
  5. The `mobile_devices.type` select does not list `ios_phone`, which the plan rejects with `422 device_type_not_supported`. Options: keep the check for records written without validation or a future type value, or drop it.
- default risk: (1b) and new result values would change pipeline behavior or semaphore history; a different queue-limit status splits one semaphore condition across two codes for the same clients.
- decisions (agent defaults):
  1. (a): pipeline behavior and history are unchanged; the session path keeps the plan's checks.
  2. Pipeline values: `success` when the agent ended in any way, `failed` when setup failed, `canceled` when the session was canceled.
  3. `409` with the queue endpoint's exact error, so one semaphore condition has one answer.
  4. `503 start_failed` plus a best-effort `CancelRun` of the ticket, so an enqueue that timed out does not hold the device for a session nobody watches.
  5. Keep the check (tested with a record saved without validation); `GET /api/action-author/devices` also skips unsupported types.
- follow-up: when #1553 merges, move its early tracking into the shared core (plan contingency). The installer grant is deleted (plan step 6), so decision 3 of the 29/10/2026 code-review entry no longer applies; `RouteDefinition.ResponseContentType` stays without a user, per decision 2 of that entry.

### 2026-10-30 - Action author review fixes: save caller contract and start-endpoint answers

- status: open (agent defaults applied; human may revisit)
- owner: human maintainer
- context: fixes of the security, Credimi and runner reviews of the action author on runners, before end-to-end testing.
- questions and options considered:
  1. How the `Save authored wallet action` activity trusts its caller. Options: trust the payload's actor and build (any workflow of the namespace could save as anyone); or accept only the `ActionAuthor` child (`ActionAuthorSession-<ticket>-author`) of a running `Action author session` of the same namespace and take actor, version and platform from the session's start input.
  2. How a save answers an existing action with the same name. Options: the explicit replacement precondition (`replace_action_id`, `expected_existing_sha256`), which the harness callback path never used; or a conflict that names neither the action id nor its code SHA256.
  3. How `POST /api/action-author/sessions` answers a device whose runner is `disabled`. Options: a new `403 runner_disabled`; or the existing offline answer (`503`, domain `device_id`, reason `device runner is offline`).
  4. How it answers a device of an organization the caller does not own. Options: `403 runner_not_owned` after the type check (reveals existence and type); or `404 device_not_found` before any device detail.
- default risk: (1) costs two `DescribeWorkflowExecution` calls and one history read per save; a Temporal lookup failure is retried, a non-matching caller is a non-retryable `CRE234`. (3) a disabled runner looks unreachable rather than administratively disabled.
- decisions (agent defaults):
  1. The session contract; a payload value that differs from the session's is `CRE234`, an omitted one is filled from the session.
  2. Conflict without id or hash; actions are never replaced by the authoring path, and a retry of the same workflow run still returns its own record.
  3. The offline `503`: the device list hides disabled runners, `mobilerunner.RecentlyAlive` and the worker start already treat a disabled runner as offline, so one condition keeps one answer.
  4. `404 device_not_found`, the same answer as an unknown device; superusers see every runner and still get `403 runner_not_owned` when the runner and the wallet belong to different organizations.
- follow-up: the iOS `release_device` activity has a 30 s schedule-to-start timeout, so a runner without the agent worker no longer blocks cleanup and RunDone.
