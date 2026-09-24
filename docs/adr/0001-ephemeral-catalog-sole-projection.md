# Ephemeral query cache is the sole post-rebuild catalog projection

Filesystem `config_templates` remains the durable source of truth. After `Rebuild`,
the only live catalog projection is a process-private `:memory:` SQLite query
cache that backs the fake PocketBase collection URLs. We rejected keeping a
second in-memory `[]Check` snapshot written beside that cache: production
list/get already used SQLite, so dual-write failed the depth deletion test and
risked divergence. The thin `Snapshot` read-through (`SELECT` → `[]Check`) was
removed: tests assert via production list/get or `countEphemeralChecks` (also
used by rebuild HTTP), not a parallel typed dump of the cache.

The cache is an owned handle (`catalogCache`), not a bare process-global DB.
Production uses one default handle behind the `Rebuild` / `catalogDB` facades
(fixed shared-memory URI). Package tests that need isolation open a private
handle with a distinct URI via `openCatalogCache`; HTTP handlers keep the
default facade so route wiring stays thin. We rejected injectable-URI-only
package functions (ownership stays implicit) and injecting raw `*dbx.DB` into
handlers (schema/lifetime would leak to callers).

## Considered Options

- **Dual-write Snapshot + ephemeral SQLite** — convenient for tests/counts;
  rejected as a consistency footgun.
- **Keep Snapshot as SELECT read-through for tests** — deferred then removed;
  it was a non-production assert path with no callers outside tests.
- **Ephemeral SQLite sole projection (chosen)** — one place owns “what the
  catalog currently is.”
- **Injectable URI / raw DB injection without a named handle** — rejected;
  lifetime and schema stay on the cache handle; facades keep boot/HTTP churn
  small.
