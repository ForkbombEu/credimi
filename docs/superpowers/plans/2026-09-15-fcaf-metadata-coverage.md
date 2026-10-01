# FCAF plain redirect-URI and metadata coverage

## Scope

Correct source-test ownership for the six supported cases:

- `WS_RP_MS_ProtocolMessages__038`
- `WS_RP_MS_Metadata__111`
- `WS_RP_MS_Metadata__129`
- `WS_RP_MS_Metadata__131`
- `WS_RP_IA_Metadata__015`
- `WS_RP_IA_Metadata__016`

`WS_RP_MS_Metadata__125` and `WS_RP_MS_Metadata__127` remain blocked: on 15/09/2026, Capture Wallet rejected `x509_san_dns` session creation because its supplied leaf certificate has no DNS SAN.

## Evidence design

1. Create a plain redirect-URI scenario that preserves the unsigned Authorization Request and its session result. Bind case 038 to the exact plain session and successful Wallet response.
2. Create an X.509-hash scenario that captures the compact signed Request Object and session result. Validate its JWS signature and leaf-certificate hash binding for cases 111, 129, and 131.
3. Create two decentralized-identifier scenarios: one with non-key verifier metadata and one with an empty replacement `client_metadata`. Capture the service DID document, the signed Request Object, the session response, and UI evidence. The positive case validates the DID signing key and successful Wallet response; the negative case validates the empty request metadata and rejection without a presentation.
4. Remove the six test IDs from generic scenarios, regenerate the aggregate pipeline, and update generated integration counts.

## Verification

Run focused validator, catalog, generator, and pipeline tests; then `make test`. Run the repository Go formatter and lint entrypoints where private dependency access permits.