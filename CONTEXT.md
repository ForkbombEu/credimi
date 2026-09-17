<!--
SPDX-FileCopyrightText: 2024-2026 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024-2026 The Forkbomb Company
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# Credimi

Domain language for Credimi product concepts. Implementation details do not belong here.

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
