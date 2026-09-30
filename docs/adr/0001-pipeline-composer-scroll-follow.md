# Pipeline Composer Scroll follow keeps viewport peer sync only

Scroll follow in Pipeline Composer peer-syncs the cards pane and YAML preview viewports. It must not select or highlight a step: hover and selected washes/rings are separate, user-driven state. Leadership uses an explicit scroll leader, last user-intent side, and driven-side suppression so residual peer scrolls and YAML regen cannot yank the opposite pane mid-gesture. Mid-gesture align is cards→YAML `start-band` and YAML→cards `center`; hard starts (edit focus, enable toggle, regen) may use start-align on YAML.

Rejected: proportional scroll linking, and driving wash/selection from the viewport active unit.
