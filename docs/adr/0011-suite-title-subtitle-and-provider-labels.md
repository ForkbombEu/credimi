# Suite title/subtitle and provider labels

Suite display splits into short `name` (suite title) and optional `subtitle` in
`metadata.yaml`, projected as `suite_name` / `suite_subtitle`. Provider facet
slugs stay the filter wire; short labels (`EWC`, `OpenID`, …) are authored once
in `config_templates/providers.yaml` and projected as `provider_label` on suite
rows. Suite title may be longer than the provider label (OpenID Foundation vs
OpenID) so compact selects and suite branding stay independent.

## Considered Options

- **Heuristic split of the old full `name`** — rejected; OpenID and EUDIW do not
  split cleanly.
- **Provider label repeated on every suite metadata** — rejected; drifts across
  suites of the same provider.
- **Title + subtitle on suite + central provider registry (chosen)** — one SoT
  per concern; filter values unchanged.
