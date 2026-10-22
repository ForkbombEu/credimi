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
- `WS_RP_SM_IssuerIntegrity__012`: needs the Wallet to hold a revoked mdoc
  past the status list's max-age. Capture revokes a stored mdoc (`POST
  /sessions/{id}/status`, probed 22/10/2026), but the list is served with
  `Cache-Control: max-age=43200`, so a Wallet that fetched it at issuance may
  rely on the valid copy for 12 hours. The pipeline has no wait step, and each
  scenario resets the Wallet, so the wait cannot overlap other scenarios.

## Blocked on Capture

Each entry names the input Capture does not supply; the test IDs listed with it
assert `fcaf.skipped` naming that input, unless stated otherwise.

- `x509_san_dns` sessions: production answers `500 internal_error`, "there are
  no SAN-DNS names" (re-probed 22/10/2026; the deployed verifier certificate
  carries `URI:https://capture-wallet.credimi.io/` and no `dNSName`).
  `WS_RP_MS_Metadata__125`, `127`, `128` additionally need a registry-issued
  `dNSName` certificate, which upstream states does not exist.
- Wallet-trusted Verifier attestation issuer: `WS_RP_SM_RpIntegrity__010`
  needs an operator to configure the Wallet to trust the fixture attestation
  issuer at `/openid4vp/verifier-attestation-issuer/jwks.json`.
  `verifier_attestation` sessions themselves are deployed (probed 22/10/2026).
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
  (leaf plus `PID Issuer CA 02`). Note, unreviewed against these sources:
  `certificate_chain: "with_trust_anchor"` (built for 025) serves exactly
  [leaf, `PID Issuer CA 02`] since 22/10/2026.
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
- COSE CWT Referenced Token fixture: `WS_RP_MS_Metadata__101`, `102`,
  `WS_RP_MS_CredentialFormats__032`.
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
