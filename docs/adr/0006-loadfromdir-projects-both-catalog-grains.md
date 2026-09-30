# Filesystem walk projects both catalog grains

`LoadFromDir` is the suite indexer: one templates walk returns lean checks and
projected suite-grain rows. Suite display from `metadata.yaml` stays internal to
that walk and is applied during projection; callers (including `Rebuild`) do not
compose `projectSuites` themselves. We rejected leaving suite aggregation at the
ephemeral write step — that split hid display/join bugs across `LoadFromDir` and
`replaceEphemeralRows`.

## Considered Options

- **New named indexer type** — same depth, more rename churn; rejected.
- **Twin Check-loader merge only** — leaves suite projection outside the walk;
  rejected for this cut (deferred separately).
- **Walk projects both grains (chosen)** — one FS→both-grains interface; display
  map is an implementation detail.
