# FCAF Positive Adapt-Manual Plan

## Goal

Promote the positive `adapt manual` FCAF tests into repository-backed Credimi
definitions, grouped by equivalent protocol request and evidence shape. Keep
negative and malformed scenarios in separately named pipelines.

## Current baseline

- 74 tests are covered by the existing SD-JWT/mDoc happy-flow batch.
- `WS_RP_IA_MainInteraction__008` is implemented and reuses the holder-binding pipeline.
- `WS_RP_MS_ProtocolMessages__013` is implemented and reuses the multi-query pipeline.
- The remaining positive candidates require additional request shapes.

## Batch 1: Existing reusable evidence

Status: in progress

- `WS_RP_IA_MainInteraction__008`
  - Pipeline: `dcql-holder-binding-type-boundary`
  - Evidence: DCQL exchange, presentation, Maestro screenshots
- `WS_RP_MS_ProtocolMessages__013`
  - Pipeline: `dcql-same-credential-multiple-queries`
  - Evidence: multiple credential-query entries, VP response, screenshots

## Batch 2: DCQL request variants

Status: planned

- Claims-only request (`given_name` only)
- Request without `claims`
- Required and optional credential sets
- Multiple matching credentials
- Transaction data with valid `credential_ids`

Each pipeline must define the exact DCQL body, use the Capture Wallet
`/openid4vp/sessions` API, run the corresponding Maestro flow through the
registered runner, fetch the session evidence, and validate the captured
request/response rather than relying on screenshots alone.

## Batch 3: Verifier and request-object positive flows

Status: planned

- `verifier_info` without `credential_ids`
- `client_id` and `iss` handling
- Request URI POST transport and content headers
- Request-object response content type and encoding
- `wallet_nonce` round-trip
- Request-object/query parameter precedence
- Redirect URI client identifier handling

These require dedicated Capture Wallet request/session support and must not be
attached to the ordinary presentation happy flow.

## Batch 4: Credential formats and security positives

Status: planned

- Credential format handling
- RP integrity positive cases
- Session encryption positive cases
- Interaction metadata and supportive interaction cases

## Per-test artifact contract

Every implemented test must include:

1. Source-backed YAML test definition.
2. A reusable precondition reference.
3. A repository-backed pipeline YAML.
4. A Maestro action written and verified with Maestro MCP on the real wallet.
5. Capture Wallet protocol evidence and visual evidence separately.
6. Validator definitions and unit tests where a new assertion mode is needed.
7. An updated `implementation-inventory.csv` row.
8. A batch manifest entry with an independent run command.

## Validation gates

```sh
go test ./pkg/fcaf/... ./cmd/cli
git diff --check
make fcaf-sync FCAF_RUNNER_ID=forkbomb-bv-andrea/usb CREDIMI_API_KEY="$CREDIMI_API_KEY"
```

Only tests whose pipeline, evidence mapping, and validators pass these gates
are added to the runnable batch manifest.
