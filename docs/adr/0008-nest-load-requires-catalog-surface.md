# Nest load requires Catalog surface

Nest-load APIs (`listNest`, `getNestStandards`, `Store.load`, and
path resolve that shares that options shape) require an explicit Catalog
surface (`manual` | `pipeline`). There is no silent default. We rejected
defaulting to `manual`: the live client Store boots as `pipeline`, so an
omitted surface projected the wrong Catalog surface without a type error.
Hub list intents (`SuiteListIntent` / `CheckListIntent` → `listHubSuites` /
`listChecks`) also require Catalog surface; compile always emits the
`visible_in` filter. Multi-surface Store cache remains deferred (ADR-0004);
requiring the knob does not need that cache.

## Considered Options

- **Default `surface = 'manual'`** — fewer call-site args; rejected as a
  silent wrong-projection footgun when client nest is `pipeline`.
- **Required Catalog surface (chosen)** — call site names the surface;
  wrong-surface bugs become type errors.
