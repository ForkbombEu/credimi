<!--
SPDX-FileCopyrightText: 2024-2026 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024-2026 The Forkbomb Company
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# In-card edit parks the cards pane; YAML Scroll follow stays live

Pipeline Composer **In-card edit** expands the Step or Follow-up card to host the edit form, fades sibling cards and non-active YAML lines, and start-aligns the card and its YAML range on open (ADR `0001-pipeline-composer-scroll-follow` hard start). After that enter start-align settles, the cards pane stays still until Save/dismiss exit completes (including shrink): Scroll follow peer sync is muted in both directions (the preference stays; the switch is inert). The YAML preview can still be scrolled by itself. Peer sync resumes when the pane is no longer still. Unlock is exit-complete, not when form mode goes idle while the held form is still mounted. Draft form values still commit to the pipeline/YAML only on Save.

Two clocks own the seams: park still-ness (`idle | aligning | still | exiting`) lives in `in-card-session` (twin-pane; `still` mutes peer sync until `noteExitComplete`); card layout enter/exit (`idle | waiting | entering | settled | exiting`) lives in `in-card-host-machine` (Runed FSM, driven by Host `layout.sync`). Face is `StepCardDisplay` (summary chrome shared by Hub and Composer); Host (`InCardHost`) owns Composer enter/exit motion and mounts the form over the Face. Scroll-follow leadership and mid-gesture align stay in ADR `0001`.

This amends the earlier “keep follow on cards for the whole session” rule: column-fill enter made YAML→cards follow hostile to a filled cards pane.

Rejected: freezing YAML; pinning via TanStack `rangeExtractor` as the primary fix; overflow-hiding the cards pane during the enter start-align scroll; unlocking when form mode goes idle while the held form is still mounted.
