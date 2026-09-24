# Catalog barrel exposes use-case lists, not grain list twins

The public `$lib/conformance` barrel exports hub Product-axis suite listing as
`listHubSuites`, FCAF check-grain listing as `listFcafTests` / `getFcafTests`,
and nest helpers (`getNestStandards`, Store via `Standards`). Grain adapters
`listChecks` / `listSuites` stay package-internal (SuiteBrowse and nest compose
call them through `listHubSuites` / `listNest`; `listFcafTests` compiles
`fs_standard: 'fcaf'` and maps path → test id/title). We rejected keeping grain
list names on the barrel: callers had to choose Conformance check vs suite grain
and Product vs Filesystem axis, which invited dual-fetch and Store bypass
(ADR-0002 / 0004 / 0007). Catalog surface remains required on nest and hub /
FCAF list options (ADR-0008). FCAF taxonomy grouping stays on `$lib/fcaf`.

## Considered Options

- **Keep `listChecks` / `listSuites` on the barrel** — facade as sugar only;
  rejected; the footgun remains.
- **Namespace or session object** — one import surface; rejected; only one HTTP
  adapter and extant call shapes are named functions.
- **Use-case barrel entrypoints (chosen)** — `listHubSuites` + `listFcafTests` +
  nest helpers public; grain lists internal.
