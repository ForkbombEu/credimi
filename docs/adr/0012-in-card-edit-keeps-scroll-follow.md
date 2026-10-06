<!--
SPDX-FileCopyrightText: 2024-2026 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024-2026 The Forkbomb Company
SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# In-card edit parks the cards pane; YAML Scroll follow stays live

While In-card edit is open, the cards pane stays parked until Save or dismiss
exit completes. Scroll follow stays on as a preference, but pane sync is muted
both ways so the cards pane does not move with the YAML preview. The YAML
preview can still be scrolled by itself. Unlock is exit-complete, not when the
form looks idle while it is still mounted. Draft form values still commit to the
pipeline/YAML only on Save.

Park mute and layout motion are separate clocks.

This amends the earlier “keep follow on cards for the whole session” rule:
column-fill enter made YAML→cards follow hostile to a filled cards pane.

Rejected: freezing YAML; pinning via TanStack `rangeExtractor` as the primary
fix; unlocking when form mode goes idle while the held form is still mounted.
