# Ephemeral query cache is the sole post-rebuild catalog projection

Filesystem `config_templates` remains the durable source of truth. After `Rebuild`,
the only live catalog projection is the process-private `:memory:` SQLite query
cache that backs the fake PocketBase collection URLs. We rejected keeping a
second in-memory `[]Check` snapshot written beside that cache: production
list/get already used SQLite, so dual-write failed the depth deletion test and
risked divergence. The thin `Snapshot` read-through (`SELECT` → `[]Check`) was
removed: tests assert via production list/get or `countEphemeralChecks` (also
used by rebuild HTTP), not a parallel typed dump of the cache.

## Considered Options

- **Dual-write Snapshot + ephemeral SQLite** — convenient for tests/counts;
  rejected as a consistency footgun.
- **Keep Snapshot as SELECT read-through for tests** — deferred then removed;
  it was a non-production assert path with no callers outside tests.
- **Ephemeral SQLite sole projection (chosen)** — one place owns “what the
  catalog currently is.”
