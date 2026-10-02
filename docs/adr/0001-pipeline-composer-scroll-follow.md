# Pipeline Composer Scroll follow keeps viewport peer sync only

Scroll follow in Pipeline Composer peer-syncs the cards pane and YAML preview viewports. It must not select or highlight a step: hover and selected washes/rings are separate, user-driven state. Leadership uses an explicit scroll leader, last user-intent side, and driven-side suppression so residual peer scrolls and YAML regen cannot yank the opposite pane mid-gesture. Mid-gesture align is cards→YAML `start-band` and YAML→cards `center`; hard starts (edit focus, enable toggle, regen) may use start-align on YAML.

Continuous peer target: cards→YAML uses the **topmost** card whose top edge is inside the cards viewport (skips a top-clipped sliver so a barely-peeking card with long YAML does not keep owning follow); YAML→cards remains center (+ hysteresis). Selection / UnitHighlight stays user-driven and is not retargeted from `activeUnit`.

Discrete reorder framing (paired step shift): after a successful adjacent swap, frame the **swapped pair** (both indices) in cards and YAML independently — when the pair fits, minimal scroll so both units are fully visible (no scroll if both already are); when the pair is taller than the viewport, no scroll if both already intersect, else center the gap between them (top/bottom may clip). Cards frame first; YAML may lag slightly to help sync. Continuous topmost/center follow is unchanged after a brief mute around the reorder. This is not the old single-unit nearest reveal path.

Twin-pane session (`createTwinPaneSession`): one module owns both composer virtualizers, PeerScrollFollow, UnitHighlight (still separate from `activeUnit`), multi-list FLIP layout, and paired shift (`runPairedShift`, including `ensureSwapIndicesMounted`). The Steps builder view binds DOM roots and forwards UI events; it does not orchestrate mute→mount→FLIP→frame. `layout-swap` stays FLIP-only (no TanStack merge).

Rejected: proportional scroll linking, and driving wash/selection from the viewport active unit.
