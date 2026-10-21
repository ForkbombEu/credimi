<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF assertion review backlog

Open work only. A completed item is removed from this file; its reasoning stays
in git history and in `pkg/fcaf/MEMORY.md`. Everything below waits on a
maintainer decision, on the test environment, or on a missing Capture feature,
and each entry names what is missing.

A test that already has a Credimi definition keeps running in the generated
aggregate pipeline. Listing it here means its assertions or its evidence are not
yet trustworthy, not that it is absent from execution.

Review baseline: the source mirror under
`config_templates/fcaf_sources/wallet_solution/relying_party/` holds 621 distinct
`WS_RP_*` sources against 617 Credimi definitions. The four sources without a
definition are `WS_RP_SM_RpIntegrity__013c_UF`,
`WS_RP_SM_TrustMechanisms__101`, `101b_UF`, `101c_UF` (blocked on Capture).
Scenarios and probes target production `https://capture-wallet.credimi.io`;
facts undated below were probed 14/10/2026.

## Maintainer decision

- `WS_RP_SM_RpIntegrity__001`: Capture can deliver a `verifier_info`
  attestation with a corrupted signature (no signing key needed), so it is not
  Capture-blocked. The source applies only to a Wallet that uses `verifier_info`,
  and no evidence shows whether a Wallet does: a rejection passes, but a
  presentation cannot be told apart between "ignores the attestation"
  (`not_applicable`) and "accepts a bad one" (fail). Decide which verdict a
  presentation gets before implementing.
- `WS_RP_IA_MainInteraction__066`, `WS_RP_MS_ProtocolMessages__125`: their
  sources accept an error or a termination, and the DSL cannot express the
  termination branch, so the flows assert a named error screenshot and fail a
  Wallet that stays on its success screen while satisfying the source's
  alternative. Extend the accepted-outcome vocabulary or accept the strict
  failure.
- `mdoc_without_claim_disclosures` (`WS_RP_IA_MainInteraction__017`): the
  `credentials_match` family now consults `checks.dcql_query_matched`; add the
  same verifier-verdict check to this mode or leave it deliberately out.

## Not buildable by the harness (environment), maintainer decision

- `WS_RP_UC_Presentation__004`: needs an independently controllable second
  device (cross-device transfer); the pipeline drives one device.
- `WS_RP_IA_Supportive__006`: needs a deliberately resource-exhausted device
  (out of RAM and disk) answering `wallet_unavailable`.

## Blocked on Capture

Each entry names the input Capture does not supply; the test IDs listed with it
assert `fcaf.skipped` naming that input, unless stated otherwise.

- `verifier_attestation` and `x509_san_dns` sessions: production answers `500
  internal_error`, "there are no SAN-DNS names", for both prefixes (re-probed
  14/10/2026; the deployed verifier certificate carries
  `URI:https://capture-wallet.credimi.io/` and no `dNSName`). Needs a deployed
  certificate whose SAN matches the domain: `WS_RP_SM_RpIntegrity__008`, `009`,
  `011`, `012`, `WS_RP_MS_Metadata__116`–`124` and
  `WS_RP_IA_MainInteraction__056` (which is defined but currently bound to this
  unbuildable session). `WS_RP_MS_Metadata__125`, `127`, `128` additionally need
  a registry-issued `dNSName` certificate, which upstream states does not exist.
  `WS_RP_SM_RpIntegrity__010` has one further requirement that is not part of
  this blocker: an operator must configure the Wallet to trust the fixture
  attestation issuer at
  `/openid4vp/verifier-attestation-issuer/jwks.json`.
- `openid_federation`: Capture rejects the prefix with `400
  unsupported_client_id_scheme` and publishes no entity statements:
  `WS_RP_MS_Metadata__114`, `115`.
- RS384 or PS384 Request Object: Capture's KMS generates EC keys only, and the
  intended algorithm is an open upstream decision:
  `WS_RP_SM_RpIntegrity__032`,
  `WS_RP_SM_RpIntegrity_CryptographicSignature_002`.
- Multi-element `x5c` for `WS_RP_SM_RpIntegrity__016`, `018`, `020`: Capture
  serves the deployed certificate as a single-element `x5c` (leaf only,
  re-checked 14/10/2026) and `request_behavior.certificate_chain` only generates
  synthetic untrusted chains. Needs a control that serves the deployed chain
  (leaf plus `PID Issuer CA 02`).
- Trust anchor inside `x5c` for `WS_RP_SM_RpIntegrity__025`: needs a verifier
  control that serves the Wallet's trust anchor, `PID Issuer CA 02`, in the
  chain; a scenario cannot build it from the leaf's AIA and would break when the
  anchor changes.
- Verifier certificates issued by a trust registry: a WRPAC/WRPRC access or
  registration certificate for `WS_RP_SM_TrustMechanisms__101`, `101b_UF`,
  `101c_UF`, and a Wallet-trusted certificate for an EdDSA signing key for
  `WS_RP_SM_RpIntegrity__013c_UF` (without it the Wallet rejects on trust, not on
  the algorithm). No Credimi definitions yet.
- Credential without cryptographic holder binding: Credo refuses
  `require_cryptographic_holder_binding: false` on issuance
  (credo-ts#2936, open 13/10/2026): `WS_RP_IA_MainInteraction__006`, `008`,
  `010`, `WS_RP_MS_CredentialFormats__046`.
- Credential without selective disclosure, so an empty `claims` object means
  full disclosure: `WS_RP_IA_MainInteraction__028`.
- Credential whose data the Wallet cannot parse ("Encrypted data" profile):
  `WS_RP_IA_MainInteraction__024`.
- Credential large enough to exceed user-agent URL limits:
  `WS_RP_IA_MainInteraction__046`.
- Revoked ISO mdoc (Capture publishes a Token Status List allocation only):
  `WS_RP_SM_IssuerIntegrity__012`.
- COSE CWT Referenced Token fixture: `WS_RP_MS_Metadata__101`, `102`,
  `WS_RP_MS_CredentialFormats__032`.
- `status_reference` malformed-URI shapes beyond the unparseable one
  (missing scheme, unencoded space, invalid percent-encoding, illegal character,
  empty string): `WS_RP_MS_Metadata__089`, `099`, which currently assert only
  the unparseable shape.
- SD-JWT VC JSON serialization: Capture's issuer and verifier support compact
  serialization only, so the credential cannot be delivered in JSON form.
  `WS_RP_MS_CredentialFormats__048` is also profile-conditional (a Wallet profile
  that requires JSON serialization), so a Wallet whose profile does not require
  it reports `not_applicable` rather than `skipped`.
- Issuer chain of depth 3 or more with a controlled Authority Key Identifier in
  the leaf, a sub-CA or the CA (Capture's chain is the issuer leaf plus `PID
  Issuer CA 02`): `WS_RP_SM_TrustMechanisms__005`–`007`.
- ETSI Trust List fixture (Capture publishes no Trust List):
  `WS_RP_SM_TrustMechanisms__009`–`013`, `015`,
  `WS_RP_MS_ProtocolMessages__095`.
- Signing of a profile-conformant `verifier_info` attestation: Capture delivers
  it verbatim and signs none, and the EUDI Relying Party Registration
  Certificate has no TS5 attachment structure, so no profile format exists for
  the caller to sign: `WS_RP_SM_DeviceBinding__002`–`006`.
- `request_uri` target recording: `raw.request_uri_http` carries only `method`,
  `headers` and `body` (re-checked 14/10/2026), so ER3.3 of
  `WS_RP_IA_ProtocolFlow__002b` and `002d` (the requested path, without query or
  fragment) cannot be asserted. Capture must record the requested target.
- `WS_RP_MS_Metadata__134` for a Wallet publishing `jwks`: encryption is
  deployed (probed 14/10/2026: POST Request URI retrieval returns the Nested JWT
  in `raw.authorization_request_jwe`), but Capture negotiates the RFC 8414
  `request_object_encryption_*` members while the source names
  `authorization_encryption_*`, so a Wallet advertising only the source's
  members is answered `400`. Upstream decision.
