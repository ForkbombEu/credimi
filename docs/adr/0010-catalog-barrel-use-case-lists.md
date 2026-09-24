# Catalog barrel exposes use-case lists, not grain list twins

The public `$lib/conformance` barrel exports hub Product-axis suite listing as
`listHubSuites` and nest helpers (`getNestStandards`, Store via
`Standards`). Grain adapters `listChecks` / `listSuites` stay package-internal
(SuiteBrowse and nest compose call them through `listHubSuites` / `listNest`;
FCAF deep-imports `listChecks`). We rejected keeping grain list names on the
barrel: callers had to choose Conformance check vs suite grain and Product vs
Filesystem axis, which invited dual-fetch and Store bypass (ADR-0002 / 0004 /
0007). Catalog surface remains required on nest and hub list options (ADR-0008).

## Considered Options

- **Keep `listChecks` / `listSuites` on the barrel** — facade as sugar only;
  rejected; the footgun remains.
- **Namespace or session object** — one import surface; rejected; only one HTTP
  adapter and extant call shapes are named functions.
- **Use-case barrel entrypoints (chosen)** — `listHubSuites` + nest helpers
  public; grain lists internal.
