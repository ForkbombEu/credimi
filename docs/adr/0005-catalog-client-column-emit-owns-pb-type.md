# Catalog Client-column emit owns Kind→PB field mapping

Go `columnSpec` (`schema.go`) remains the sole SoT for Client-visible catalog
columns. `go generate ./pkg/conformancecatalog` emits `columns.ts` specs that
include `pbType` (`text` / `number` / `json`) derived from Kind, plus Zod and PB
record bodies. Bun injectors (`generate.collections-models.ts`,
`generate.catalog-pb-types.ts`) only stitch those artifacts into PocketBase
codegen outputs; they do not re-map Kind. The catalog-pb-types inject upserts
Record bodies when the marker is already present so a Client column change does
not require a typegen wipe. Companion entrypoint: `make generate-catalog-wire`
(Go emit, then collections-models + types). Plain `make generate` stays Go-only;
`predev` / `generate:definitions` do not run `go generate`.

## Considered Options

- **Recipe-only chain** — one make target, leave Kind→PB in bun; rejected as
  ongoing hop tax on every Client column.
- **Go emits full merge fragments** — heavier Go surface; `data.db` / typegen
  still need a stitch step; rejected.
- **Spec-deepen + thin injectors (chosen)** — Kind→PB lives once in Go emit;
  bun stays stitch-only; respects typegen/`data.db` reality.
