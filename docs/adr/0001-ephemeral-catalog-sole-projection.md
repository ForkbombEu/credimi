# Ephemeral query cache is the sole post-rebuild catalog projection

Filesystem `config_templates` remains the durable source of truth. After `Rebuild`, the only live catalog projection is the process-private `:memory:` SQLite query cache that backs the fake PocketBase collection URLs. We rejected keeping a second in-memory `[]Check` snapshot written beside that cache: production list/get already used SQLite, so dual-write failed the depth deletion test and risked divergence. `Catalog.Snapshot()` may remain only as a thin read-through (`SELECT` → `[]Check`) for tests; it must not be a second write target.

## Considered Options

- **Dual-write Snapshot + ephemeral SQLite** — convenient for tests/counts; rejected as a consistency footgun.
- **Delete Snapshot entirely** — viable; deferred while tests still assert on `[]Check` via read-through.
- **Ephemeral SQLite sole projection (chosen)** — one place owns “what the catalog currently is.”
