# Shared product and filesystem axis wire names across catalog grains

Conformance check and conformance suite projections use the same JSON/SQL field
names for the dual browse axes: product `standard` / `component` / `version`,
filesystem `fs_standard` / `fs_version` (plus suite `path_prefix`). Callers do
not learn a second vocabulary per grain. Checks once used FS segments under
`standard` / `version` and product under `norm_*`; that collision forced every
filter and nest caller to know which grain they held. We renamed the check wire
to match the suite grain rather than renaming suites back to `norm_*` or keeping
dual vocabularies.

## Considered Options

- **Keep check `standard`=FS and `norm_*`=product** — rejected; silent mis-filters.
- **Rename suite product fields to `norm_*`** — rejected; suite table and ADR-0002
  already taught product names on the suite grain.
- **Shared axis names on both grains (chosen)** — one vocabulary; dual axes remain.
