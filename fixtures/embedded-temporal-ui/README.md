# Embedded Temporal UI API fixtures

Harvested from a production run-page HAR (`credimi.io`, 2026-10-01). These are
the JSON bodies the Temporal UI iframe fetches under `/temporal-ui/api/v1/...`
through Credimi's org-scoped proxy.

## Enable locally

Point the API process at this directory (absolute path or relative to CWD):

```sh
export CREDIMI_TEMPORAL_UI_API_FIXTURES="$PWD/fixtures/embedded-temporal-ui"
```

Then restart the API and open:

```
http://localhost:<UI_PORT>/my/tests/runs/Pipeline-fcaf-wallet-relying-party-data-model-sd-jwt-validation-764025c1-299a-43d1-8a59-6131c8451f76/01a0f3a9-a91c-7a83-8cda-f063f4695d0c
```

That env also serves Credimi `GET /api/my/workflows/.../runs/...` from
`bodies/credimi-my-workflow-run.json` (page shell). HTML/JS for the iframe still
come from the live `temporal_ui_embedded` service; Temporal UI API JSON is
replayed. Auth, method, and namespace scoping still apply. The recorded namespace
`fcaf-1` is rewritten to the caller's organization namespace in Temporal UI
response bodies.

Leave the env unset in production. Large history bodies may be stored as `.gz`;
the fixture loader decompresses them.

## Refresh from a HAR

```sh
python3 - <<'PY'
# extract /temporal-ui/api entries into fixtures/embedded-temporal-ui/
# (see pkg/internal/temporalui fixtures loader + this manifest shape)
PY
```

Or re-run the extractor used to build `manifest.json` / `bodies/`.
