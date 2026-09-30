# Conformance catalog and the Hub

Verified against source.

## Catalog model

- Durable source of truth is the **filesystem** under `config_templates/`; the served catalog is a process-private `:memory:` SQLite projection rebuilt on boot (`pkg/conformancecatalog/catalog.go:5-11`).
- Layout: classic `standard/version/suite/<check>.yaml`; FCAF `fcaf/<version>/<suite>/tests/<id>.yaml`. `fcaf_sources` is excluded from the walk.
- Discovery gates: a version directory needs `version.yaml`; a suite directory needs `metadata.yaml`; `standard.yaml` is optional (UID falls back to the directory name). Unparsable authored YAML fails bootstrap closed.
- Classic checks = every non-directory file at the suite root except `metadata.yaml`; the check stem is the filename without extension. FCAF checks = `tests/*.yaml`, the test id being the `id:` key or the filename stem.
- Durable `path` = `<fsStandard>/<fsVersion>/<suite>/<stem>`; the record id is the first 15 hex chars of `sha256("credimi.conformance_check:" + path)` (`PathID`).

**Facets** (`protocol`, `sut`, `role`, `provider`) resolve most-specific-first: check file → suite `metadata.yaml` → suite UID (FCAF defaults `provider: fcaf`).

**Product axes** (`standard`, `component`, `version`) are normalized projections, e.g. `openid4vp_wallet` → openid4vp/wallet, FCAF `wallet_solution/relying_party` → openid4vp/wallet (with an empty normalized version); unknown prefixes pass through.

**Surface** = `visible_in` (`manual` | `pipeline`), authored on the suite and inherited by its checks; empty means both. A suite row's `visible_in` is the union over its checks.

Grain columns: checks carry `id, path, title, fs_standard, fs_version, suite, file, visible_in, protocol, sut, role, provider, standard, component, version`; suites add the display fields plus `check_count, members, path_prefix, fs_standard, fs_version`, and sort wallet → issuer → verifier, then standard, suite, version.

### `metadata.yaml` (suite)

Keys: `uid, name, subtitle, homepage, repository, help, description, logo, visible_in[], protocol, sut, role, provider`. Projection: `name→suite_name`, `subtitle→suite_subtitle`, `homepage→suite_homepage`, `repository→suite_repository`, `help→suite_help`, `description→suite_description`, `logo→suite_logo` (all trimmed).

`config_templates/providers.yaml` maps a provider slug to the short label used in compact UI (`provider_label`); `standard.yaml`/`version.yaml` are read only for their `uid`.

The fields a **check file** may declare for facet/title metadata: `title`, `name`, `protocol`, `sut`, `role`, `provider` (`title` wins over `name`, which wins over the filename stem).

## Served URL contract

Read-only PocketBase-shaped routes (public, `AuthenticationRequired: false`):

- `GET /api/collections/conformance_checks/records` and `/records/{id}`
- `GET /api/collections/conformance_suites/records` and `/records/{id}`

Writes on those paths are rejected with HTTP 400 and `conformance_checks is a read-only catalog projection of config_templates`. List/get run through `pocketbase/tools/search`, so `filter`, `sort`, `page`, `perPage`, `skipTotal` and the PocketBase ListResult shape apply.

Refresh after editing templates: restart, or `POST /api/conformance-catalog/rebuild` → `{ok, count, source}`. That route is gated by the **internal admin API key in the `Credimi-Api-Key` header** (`key_type=internal_admin`, backed by the `api_keys` collection). A code comment in `pkg/conformancecatalog/store.go:127-129` still says `X-Api-Key: $CREDIMI_INTERNAL_ADMIN_KEY` — stale, ignore it. Boot rebuild skips a missing templates directory and fails bootstrap on any other error.

## Manual check files → GUI form

A check YAML declares its inputs with the Go-template helper `credimi`:

```yaml
session_id: >-
    {{ credimi `{
        "credimi_id": "…_session_id",
        "field_id": "session_id",
        "field_label": "i18n_session_id",
        "field_description": "i18n_session_id_description",
        "field_default_value": "",
        "field_type": "string",
        "field_options": []
    }` uuidv4 }}
```

- JSON keys: `credimi_id, field_id, field_label, field_description, field_default_value, field_type, field_options`; `field_type` ∈ `string | object | options`; the optional trailing function name is the only recognised one, `uuidv4` (`pkg/templateengine/templates.go:24-42`, `207-223`).
- Declared fields are collected at template preprocess into `normalized_fields` / `specific_fields[file].fields`.
- The webapp fetches them with `POST /api/template/placeholders` `{test_id, filenames[]}` (auth required; the filename must be path-shaped). The Hub check page passes `standard/version` as `test_id`.
- Submitting the form posts to `POST /api/compliance/{protocol}/{version}/save-variables-and-start`; each `{credimi_id, value, field_name}` is persisted into the `config_values` collection, the template is rendered, and the rendered YAML is handed to the conformance workflow.
- Pipeline-executed checks extract the same declarations independently with `extractCredimiJSON` in `pkg/workflowengine/pipeline/conformance_checks_hook.go:322-433`, applying `uuidv4` and coercing `string`/`object` types.

`field_type: "options"` is declared but nothing consumes `field_options` beyond passthrough.

## Hub entity model

- `hub_items` is a PocketBase **view** (formerly `marketplace_entries` → `marketplace_items`) unioning `wallets`, `credential_issuers`, `credentials`, `custom_checks`, `verifiers`, `use_cases_verifications`, `pipelines`.
- Every branch filters `name IS NOT NULL AND published = 1`; children additionally require their parent (`credential.credential_issuer` issuer published, `use_case_verification.verifier` verifier published). Discoverability therefore = published **and** named, with a published parent.
- Row columns: `id, type, name, description, updated, avatar_file, avatar_url, organization_id, children, children_count, children_search, canonified_name, organization_name, organization_canonified_name, path`, where `path = type/organization_canonified_name/base_path`. Reads are public; writes are not possible.
- `hub_organizations` (formerly `marketplace_organizations`) lists orgs owning at least one published hub item.
- Public URLs: `/hub`, `/hub?tab=…`, `/hub/{type}/{org}/{name}` (e.g. `/hub/credentials/{org}/{issuer}/{credential}`); item detail resolves by exact `path`. Legacy `/marketplace*` routes redirect to `/hub*`.
- Canonified names are derived by hooks on create/update; the union of canonify path templates is: `credential_issuers` → organization, `credentials` → credential issuer, `use_cases_verifications` → verifier, `wallets`/`verifiers`/`pipelines`/`custom_checks` → organization, `wallet_versions` → `canonified_tag`.

### Wallets and versions on the Hub

A wallet needs `published=true` and a `name`. The wallet page expands its actions and versions. Wallet metadata: `name, description, logo, playstore_url, appstore_url, repository, home_url`. Versions carry `tag`, `android_installer`, `ios_installer`, `downloadable`: when `downloadable` is false, installers are hidden from everyone except members of the owning organization.

## StepCI integrations (issuers, credentials, verifiers)

- The StepCI YAML for a credential lives on the `credentials` record (`yaml` + `secrets`); for a verifier flow on its **use case verification** record (`use_cases_verifications.yaml` + `.secrets`). Verifiers themselves have no StepCI YAML.
- Static vs dynamic: a record with `yaml` is dynamic (re-executed for each offer/deeplink); a record with only `deeplink` is static.
- Offer resolution: `GET /api/credential/get-credential-offer?credential_identifier=<path>` returns `{credential_offer, dynamic, code, secrets}` — dynamic records return the YAML to run, static ones their stored deeplink, and a record with neither gets a synthesised `openid-credential-offer://?credential_offer=<...>`.
- Execution reads the StepCI capture named **`deeplink`** (`captures.deeplink`), which is the convention across credential offers, use case verifications, and conformance checks.
- Preview: `POST /api/get-deeplink` `{yaml, secrets?}` runs the YAML and returns `{deeplink, steps, output}` with the full StepCI report.
- CI runs rewrite a StepCI YAML's top-level `env.host` to the temporary record host; records without `env.host` are skipped. No `env.body` convention exists in code or templates.

## Deep links

Public, no auth:

- `POST /api/get-deeplink` — preview a YAML (above).
- `GET /api/credential/deeplink?id=<canonified path or id>` — expects a `credentials` record.
- `GET /api/verification/deeplink?id=<…>` — expects a `use_cases_verifications` record.

`?redirect=true` answers 301 with `Location: <deeplink>`, otherwise plain text. A `?id` that resolves to the wrong collection is a 400 `invalid record type`. Authenticated extra: `GET /api/compliance/deeplink/{workflowId}/{runId}`.

## Notes and gaps

- `services` no longer exists (renamed to `organization_info`); `standards` was deleted. Do not build on them.
- The vLEI suite directory contains only `metadata.yaml`, so the classic loader projects **zero** checks for it; where its manual check definition lives is unresolved.
- `migrations/pb_schema.json` is a stale snapshot; do not treat it as the current schema.
