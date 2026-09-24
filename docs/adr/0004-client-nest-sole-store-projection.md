# Client Filesystem-axis nest has one Store projection

On the client, the Filesystem-axis nest (suite-grain rows grouped for pickers, path resolve, and scoreboard) is owned by `webapp/src/lib/conformance/store.svelte.ts`. Client callers — including TanStack queries — must `Store.load` rather than a parallel `listNest` / barrel-exported nest fetch. We rejected QueryClient-as-SoT (awkward for sync `resolveSuite` / scoreboard) and a third nest-browse module (only one real client adapter today).

SSR route loaders and other non-hydrating nest reads use `getNestStandards` as the sole one-shot helper (either Catalog surface). They do not hydrate Store (ADR-0002: load each projection only where needed). `listNest` stays package-internal compose behind Store and that helper; it is not part of the public barrel.

The legacy `$lib/standards` flat-options module is removed. Callers that need a flat standard/version list use the one-shot helper (or Store when they share the client nest) rather than a parallel nest adapter.

## Considered Options

- **QueryClient-as-SoT** — rejected; sync resolve and scoreboard need an in-memory nest.
- **Third nest-browse module** — rejected; only one real client adapter.
- **Multi-surface Store cache** — deferred; start-checks stays SSR `manual`, client Store stays `pipeline`.
- **Keep `$lib/standards` as a second client nest fetch** — rejected; old `$lib/conformance` leftover that defaulted `manual` and bypassed Store.
