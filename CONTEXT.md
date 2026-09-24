<!--
SPDX-FileCopyrightText: 2024-2026 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024-2026 The Forkbomb Company
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# Credimi

Domain language for Credimi product concepts. Implementation details do not belong here.

## Pipeline editor (surface)

**Pipeline Composer**:
The product surface for creating and editing a pipeline (cards, YAML preview, and manual YAML edit).
_Avoid_: Steps builder (as user-facing name), pipeline form (as product name)

**Steps builder**:
Implementation name for the three-pane control inside Pipeline Composer. Not used in product UI copy.
_Avoid_: Using Steps builder in user-facing strings

**Scroll follow**:
In Pipeline Composer, the optional behaviour that keeps the cards pane and the YAML preview viewport peer-synced while scrolling. It does not select or highlight a step.
_Avoid_: Scroll sync (as the product name), proportional scroll, peer sync (as user-facing copy), active unit

## Pipeline editor (wallet version)

**Change wallet version**:
A bulk convenience action that sets the same wallet version on every matching mobile-automation step in the pipeline. It is not a persisted pipeline-level entity.
_Avoid_: Pipeline wallet version, global version, sync version (unless referring to this action)

**Step version**:
The wallet version chosen on a single mobile-automation step (including “install from external source”).
_Avoid_: Calling step version “change wallet version”

**Matching mobile steps**:
Mobile-automation steps that share the same wallet and the same serialized version id, which is the precondition for Change wallet version.
_Avoid_: All steps, every step (unless they match)

## Pipeline editor (follow-ups)

**Follow-up**:
An optional after-run action in the pipeline editor (email or HTTP only). In YAML it lives under `finally`. Product UI says Follow-up, not finally.
_Avoid_: Final step, finally step, cleanup step (cleanup is a different engine concept), after-run step (UI jargon)

**Follow-ups**:
The editor section below the main steps separator that lists Follow-ups. Maps to the pipeline `finally` object (`always` / `on_success` / `on_failure`).
_Avoid_: Finally section, final steps, When the run ends (rejected button-era wording)

**Follow-up condition**:
When a Follow-up runs relative to the pipeline outcome: Always, On success, or On failure. Always also covers canceled runs; cancel is not a separate UI control.
_Avoid_: Both checkboxes, on cancel (as its own control)

## Pipeline executions (list)

**Child workflow**:
A Temporal workflow started by a pipeline run (for example a step workflow). On the list card, “N children” means the count of these direct child workflows only.
_Avoid_: Child pipeline (unless a nested pipeline step), grandchild, step (as the count label), children as a PocketBase relation

**Pipeline run**:
One execution of a pipeline workflow (or a queued ticket awaiting start), shown as a parent row in the list SmallTable.
_Avoid_: Calling a child workflow a pipeline run on the list card

## Hub (listing)

**Hub item**:
A discoverable Hub row for one entity (for example a credential issuer or a verifier).
_Avoid_: Issuance (as the Hub tab or row name), listing card (as the domain name)

**Nested Hub item**:
A Hub item that belongs under another Hub item in the Issuers or Verifiers tabs (a credential under a credential issuer, or a use case verification under a verifier). Not a Temporal child workflow.
_Avoid_: Children (in Hub copy), child (in Hub copy), issuance credential (as the relation name)

**Credential issuer**:
The Hub parent for credentials on the Issuers / Credentials tab.
_Avoid_: Issuance, issuer-only wording that drops credentials when both are in scope

**Use case verification**:
The Hub nested item under a verifier on the Verifiers / Use case verifications tab.
_Avoid_: Use case (alone when the Hub entity is meant), verification use case (inverted label)

## Conformance catalog

**Conformance check**:
One runnable catalog entry identified by a durable filesystem-shaped path (filesystem-axis standard/version/suite/stem). It carries title, browse/filter facets, and a product-axis projection; it does not own suite display metadata.
_Avoid_: Blueprint row, template file (unless referring to the on-disk YAML), test (unless FCAF test id)

**Conformance suite**:
A suite under a product-axis standard×component×version, with authored display metadata and its member checks. The hub suite table browses product axes; nested pickers and hub detail group on filesystem axes so URLs stay path-stable. Checks are the leaves. Nest standard/version nodes are filesystem-axis labels (uid + display name), not carriers of authored standard.yaml / version.yaml metadata.
_Avoid_: Denormalizing suite display onto every check, blueprints nest, empty suite without checks, dual-fetching both projections on every hub layout, treating nest Standard/Version as authored catalog metadata

**Product axis**:
The normalized standard × component × version used to browse and filter the hub suite table and projected onto checks for facet filters. Same meaning on both catalog grains.
_Avoid_: Filesystem path segment, nest uid (when meaning product); dual-fetching Product-axis suite table and Filesystem-axis nest on every hub layout

**Filesystem axis**:
The durable path segments (filesystem standard / filesystem version / suite, and suite path prefix) that keep hub URLs and nest trees path-stable. Same meaning on both catalog grains.
_Avoid_: Product standard or version when meaning a path segment; unqualified “standard” for an FS dir

**Catalog surface**:
Which product UI may browse a Conformance suite: `manual` (start-checks) or `pipeline` (hub and pipeline pickers). Authored per suite; omit to show in both. It is a browse filter, not a runtime execution mode.
_Avoid_: Pipeline editor manual mode, show_in_pipeline_gui, treating surface as a Temporal or workflow setting

## FCAF (wallet-solution relying-party)

**FCAF assessment report**:
The persisted assessment artifact for a pipeline run (the `fcaf_report` JSON). It records suite selection, executed tests, evidence, and summary counts.
_Avoid_: FCAF result, conformance JSON, engine report (as the product name), validation output

**FCAF presentation**:
The display projection nested inside an FCAF assessment report (deeplink, screenshot index with test links, summary filters aligned to executed status). Callers render from it; they do not scrape raw evidence.
_Avoid_: Report view model, display helpers, UI report shape

**FCAF taxonomy**:
The stable grouping of tests derived from the test identifier prefix (category and subgroup). It is the source of truth for area ordering in the sheet and PDF.
_Avoid_: suite.section, YAML section labels
