# Catalog barrel omits nestSuites and fake-collection names

The public `$lib/conformance` barrel does not re-export `nestSuites` or the
fake-collection string consts (`conformance_checks` / `conformance_suites`).
Those stay package-internal next to the client adapter. Nest remains reachable
only via `Store.load` (client) or `getStandardsWithTestSuites` (SSR /
non-hydrating). We rejected keeping them on the barrel for discoverability:
exported collection names and a free nest builder invite bypassing ADR-0004’s
sole Store nest projection and inventing ad-hoc PocketBase filters.

## Considered Options

- **Keep barrel exports** — convenient for deep imports; rejected as a
  leak past the client seam and a dual-fetch / Store-bypass footgun.
- **Seal nest + collection names (chosen)** — display helpers stay public;
  adapter internals stay internal.
