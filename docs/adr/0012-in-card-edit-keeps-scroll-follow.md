<!--
SPDX-FileCopyrightText: 2024-2026 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024-2026 The Forkbomb Company
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# In-card edit keeps Scroll follow; top-align is enter-only

Pipeline Composer **In-card edit** expands the Step or Follow-up card to host the edit form, fades sibling cards and non-active YAML lines, and start-aligns the card and its YAML range on open. Scroll follow stays enabled for the session: the top-align is an enter-only hard start (as ADR `0001-pipeline-composer-scroll-follow` already allows), not a sticky pin and not a suspension of peer sync. Draft form values still commit to the pipeline/YAML only on Save.

Rejected: freezing Scroll follow for the whole edit session; keeping the editing card sticky-top for the whole session.
