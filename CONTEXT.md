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

## Pipeline executions (list)

**Child workflow**:
A Temporal workflow started by a pipeline run (for example a step workflow). On the list card, “N children” means the count of these direct child workflows only.
_Avoid_: Child pipeline (unless a nested pipeline step), grandchild, step (as the count label), children as a PocketBase relation

**Pipeline run**:
One execution of a pipeline workflow (or a queued ticket awaiting start), shown as a parent row in the list SmallTable.
_Avoid_: Calling a child workflow a pipeline run on the list card

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
