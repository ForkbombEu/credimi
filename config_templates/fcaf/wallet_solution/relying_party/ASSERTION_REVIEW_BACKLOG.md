<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF assertion review backlog

Open work only. A completed item is removed from this file; its reasoning stays
in git history and in `pkg/fcaf/MEMORY.md`. Everything below is either **ready
to work** or **blocked**, and a blocked entry names the exact missing input.

A test that already has a Credimi definition keeps running in the generated
aggregate pipeline. Listing it here means its assertions or its evidence are not
yet trustworthy, not that it is absent from execution.

Review baseline: the local source mirror under
`config_templates/fcaf_sources/wallet_solution/relying_party/`, cross-checked
against upstream `submitted` commit `2b223b56be0d0a073ee0cdc9db1d7fd31d9529a1`
(13/08/2026), and the Capture Wallet contract in `pkg/fcaf/CAPTURE_WALLET_API.md`
as synced on 28/09/2026 (`1c8162c`). The mirror has 621 distinct `WS_RP_*` files
against 615 Credimi test definitions; the six without a definition are listed
under **Blocked**.

Since 30/09/2026 the scenarios target production `https://capture-wallet.credimi.io`.
Every verdict and probe dated earlier ran against `beta-capture-wallet.credimi.io`
and has not been repeated against production.

## Ready to work

### Implemented, awaiting an emulator run against Capture

These scenarios exist and their assertions are written. None has a reference
Wallet verdict yet: the setup is accepted by Capture, which is not evidence that
the Wallet received the property or answered as the source requires.

- **Implemented; Capture evidence pending:** `WS_RP_MS_ProtocolMessages__003_UF`,
  `006`, `007`, `009`, `010`, `016`, `033`, `034`, `049`, `051`, and
  `WS_RP_SM_RpIntegrity__027`. The scenarios use `request_mutation` or
  `request_behavior.signature=corrupt` and bind the delivered JAR, outer
  request where applicable, retrieval, and Wallet outcome.
- **Implemented; Capture evidence pending:** `WS_RP_MS_ProtocolMessages__046`,
  `048`, `WS_RP_MS_Metadata__139`, `140`, and `WS_RP_IA_Supportive__002`.
  The scenarios use `request_behavior.wallet_nonce` or
  `request_uri_response`, require POST Request URI retrieval, and bind the
  delivered request or response plus the Wallet outcome.
- **Implemented; Capture evidence pending:**
  `WS_RP_IA_MainInteraction__053`, `055`, `056`,
  `WS_RP_IA_Metadata__010`, and `WS_RP_MS_ProtocolMessages__124`–`128`, `132`.
  `request_mutation` builds the missing, duplicated, and foreign-host Response
  URI requests and the unrecognized request parameter; `response_scenario`
  delivers the non-JSON, HTTP 400, and unrecognized-member verifier replies.
  Every case binds the captured Wallet HTTP request and the delivered verifier
  response. Open limitation: 125 and 126 can only evidence the Wallet's own
  error visually, because the Wallet has already submitted its Authorization
  Response when the malformed reply arrives.
- **Implemented; Capture evidence pending:** `WS_RP_IA_MainInteraction__032`,
  `040`, `041`, `WS_RP_MS_CredentialFormats__033`, `044`, and
  `WS_RP_SH_Encoding_TextualEncoding_002`, `003`. Each case now issues the
  named fixture it needs and proves the fixture reached the Wallet before
  asserting the behaviour. 040 and 041 issue `pid_default` and `pid_person_b`
  and establish simultaneous possession through two value-constrained probes
  whose `document_number` values differ, so counting presentations can no
  longer be satisfied by one credential presented twice.
- **Implemented; Capture evidence pending:** `WS_RP_MS_Metadata__081`–`090` and
  `WS_RP_MS_CredentialFormats__030`, `031`. The six accept cases and the two
  storage cases read the existing `status_list_enabled` SD-JWT presentation;
  082 reads the statusless PID presentation. The five rejection cases each
  issue one malformed `status_reference` on a distinct claim-set fixture and
  then probe for that fixture's exact `document_number`, so a Wallet that
  stored the rejected token is detected rather than assumed absent. The
  malformed variants need `FCAF_SCENARIOS_ENABLED=true`.
  Open gap: 089 lists five malformed-URI shapes and `status_reference:
  malformed_uri` expresses only the unparseable one. The missing-scheme,
  unencoded-space, invalid-percent-encoding, illegal-character and empty-string
  shapes need additional issuer fixtures.
- **Implemented; Capture evidence pending:** the ISO mdoc counterparts
  `WS_RP_MS_Metadata__091`–`100`, `103` and `WS_RP_MS_CredentialFormats__029`.
  091, 093, 095, 098 and 103 read the `status_list_enabled` mdoc presentation
  of `credential-status-list`; 029 reads its SD-JWT VC, whose issuer-signed JWT
  holds the `status` claim. 092, 094, 096, 097, 099 and 100 own
  `mdoc-status-reference-rejection`, which issues each malformed
  `status_reference` as an mdoc on the same claim-set fixtures the SD-JWT
  rejection uses and probes with a pinned `mso_mdoc` query, so SD-JWT copies
  cannot answer it. 092 and 094 share the empty Status map; 094 asserts no
  outcome, because its source leaves it to local policy.
- **Implemented; Capture evidence pending:** `WS_RP_MS_CredentialFormats__041`
  (`mdoc-multiple-device-responses`: three mdoc PIDs, three queries each
  pinning one `document_number`) and `WS_RP_MS_ProtocolMessages__013`
  (`pid-candidate-selection`: two SD-JWT PIDs matching one query). The 013
  flow was checked on wallet 2026.09.42 up to the selection: the consent screen
  lists the candidates as `Option N of M`, and exposes only the selected
  option's text to accessibility.

`request_behavior.signing_key: "unrelated"` signs the Request Object with a key
that is not the one bound to the advertised client identifier, leaving `x5c` and
the DID document untouched:

- **Implemented; Capture evidence pending:** `WS_RP_SM_RpIntegrity__015` and
  `WS_RP_MS_Metadata__132` share
  `fcaf-wallet-solution-relying-party-rp-integrity-unrelated-signing-key`.
  015 requires the delivered JAR to carry an `x5c` chain and its signature to
  fail against that leaf key, which is the RFC 7515 statement of its source.
  132 additionally requires the `x509_hash` Client Identifier to equal the
  SHA-256 of the delivered leaf, so the signing key is provably the only
  defect; that assertion is what separates it from `WS_RP_MS_Metadata__130`,
  whose Client Identifier deliberately mismatches the leaf instead. Both require
  the Wallet to answer `invalid_request` without a presentation, as their
  sources state.

`request_behavior.certificate_chain` replaces `x5c` with a generated chain that
is self-signed, rooted in an untrusted generated root, or missing its issuer,
signs with that chain's leaf key, and recomputes the `x509_hash` Client
Identifier so the chain is the only defect:

- **Implemented; Capture evidence pending:** `WS_RP_SM_RpIntegrity__017`, `019`
  and `026` share
  `fcaf-wallet-solution-relying-party-rp-integrity-certificate-chain`, one
  session per behaviour. The new `oid4vp.request_certificate_chain` validator
  decodes the delivered `x5c` and requires the exact defect: `incomplete` for
  017 (the chain links but its top-most certificate is not self-signed, so its
  issuer is absent), `untrusted_root` for 019 (the chain links and does
  terminate in a self-signed root that the Wallet has no reason to trust), and
  `self_signed_leaf` for 026 (a single certificate signing itself). Each case
  also requires the signature to verify against the delivered leaf, so the
  chain is provably the only defect, and 019 additionally pins the recomputed
  `x509_hash` Client Identifier.

  Outcome assertions follow the sources exactly: 017 and 019 require
  `invalid_request` without a presentation, while 026 accepts an error or a
  discontinuation through `request_rejected`, because its source lists
  `invalid_client`, an unspecified error, and discontinuation as equally
  acceptable. The caveat that the delivered Client Identifier changes with the
  leaf is therefore satisfied: none of the three reads a captured presentation,
  and the audience mismatch cannot mask a wrong verdict.

`dcql_query: null` combined with `scopes` delivers a Section 5.1 scope-only
Authorization Request:

- **Implemented; Capture evidence pending:** `WS_RP_MS_ProtocolMessages__030` owns
  `fcaf-wallet-solution-relying-party-unknown-scope`. `dcql_query: null`
  removes the query from the delivered Request Object while the Verifier keeps
  one for its own verification, and `scopes` is joined into the delivered
  `scope` claim. The assertions pin both halves of that precondition, the
  Wallet's retrieval of the request, and `invalid_scope` without a
  presentation. Nothing reads the query the Verifier retained; asserting on it
  would describe a message the Wallet never saw.

  `WS_RP_MS_ProtocolMessages__020`, `WS_RP_MS_ProtocolMessages__141` and
  `WS_RP_UC_Presentation__003` stay blocked: they need a scope the Wallet
  resolves to a DCQL query, and the service deliberately defines no scope
  values.

Transaction data gained object encoding per Section 5.1 and a real binding
check, `checks.transaction_data_verified`, which is `true` only when the
Wallet returned a matching `transaction_data_hashes` entry. That removes the
evidence gap but not the blocker: `WS_RP_MS_ProtocolMessages__017`, `018`,
`135` and `154`–`159` still need a transaction-data type the reference Wallet
supports.

### Digital Credentials API batch, awaiting a verdict

`WS_RP_IA_Engagement__002`, `WS_RP_IA_ProtocolFlow__003a`, `003b_UF` and
`WS_RP_SM_RpIntegrity__002`-`005`, `022` are implemented across the
`dc-api-signed-encrypted`, `dc-api-unencrypted`, `dc-api-unsigned` and
`dc-api-invalid-signature` scenarios. Two facts shaped their assertions and must
hold when they run: a DC API request carries no `response_uri`, `redirect_uri`,
`state` or `aud`, so `WS_RP_IA_Engagement__002` asserts `expected_origins` is
present and the redirect parameters are absent; and the Key Binding JWT audience
becomes `origin:<browser origin>`, which is why `WS_RP_SM_RpIntegrity__022` uses
`sdjwt.kb_jwt_claim_string_prefix`.

The `fcaf-dc-api-present` action drives the full browser path on wallet
2026.09.42, so these are now runnable.


### Known gaps inside implemented coverage

- `WS_RP_MS_ProtocolMessages__125` and `126` can only evidence the Wallet's own
  error visually: the Wallet has already submitted its Authorization Response
  when the malformed verifier reply arrives.
- `WS_RP_IA_MainInteraction__033` covers its numeric data-type axis on a
  separate credential type. The source's B2 trap is a PID-shaped credential
  holding a float, which no PID or degree fixture provides, and OpenID4VP 6.4
  restricts DCQL `values` to strings, integers and booleans, so a float
  constraint cannot appear in the PID query at all: the verifier refuses it with
  `Invalid integer: Received 70.5`. The axis therefore runs as its own pair of
  requests against `urn:credimi:numeric-claims:1`. Revisit if Capture ever
  issues a PID carrying a numeric claim.
- `WS_RP_SH_Cryptography_CryptographicHash_010` is implemented and reports
  `not_applicable` against wallet 2026.09.42, which stored and presented a
  SHA-384 digested PID. Its source only requires withholding for a Wallet that
  does not support the hash function, so this is a profile mismatch, not a
  Wallet defect. Revisit if a Wallet under test advertises SHA-256 only.
- The 15 malformed-DCQL tests migrated on 25/09/2026 assert the delivered
  request, a screenshot count, and `oid4vp.no_presentation_returned`. The
  reference Wallet answers several of those requests instead of rejecting them,
  so they are expected to fail until the Wallet is fixed; see
  `pkg/fcaf/MEMORY.md` for the per-variant table to report upstream.
- The request and metadata controls implemented on 28/09/2026 have
  reference-Wallet verdicts that are not passes:
  `WS_RP_MS_ProtocolMessages__002` fails because wallet 2026.09.42 rejects
  every plain request (`UnsupportedClientIdPrefix`: it supports only
  `x509_san_dns` and `x509_hash`, and a plain request needs the unsigned
  `redirect_uri:` prefix), a HAIP-profile mismatch rather than a defect;
  `WS_RP_MS_Metadata__106` fails on the protocol assertion because the Wallet
  rejects the non-object `client_metadata` without posting `invalid_request`
  (`REFERENCE-WALLET-ISSUES.md` RI-WALLET-003); `WS_RP_MS_Metadata__135` is
  `not_applicable` because the Wallet posts no `jwks`, so it never requires an
  encrypted Request Object.
- `WS_RP_MS_Metadata__107` reads "no `client_metadata`" as the outer
  Authorization Request of a by-reference POST retrieval, which never carries
  it. Upstream's `client_metadata: null` construction is limited to unencrypted
  `direct_post`, which the HAIP reference Wallet refuses right after retrieving
  the Request Object, so it would fail for a reason unrelated to the source.
- The Client Identifier binding controls implemented on 28/09/2026 all reach
  the reference Wallet with the source-defined defect, but only
  `WS_RP_SM_RpIntegrity__013b_UF` and `WS_RP_MS_ProtocolMessages__043` pass.
  `WS_RP_MS_Metadata__126`, `130`, `WS_RP_SM_RpIntegrity__007`, `014` and
  `WS_RP_MS_ProtocolMessages__039` fail on the protocol assertion: the Wallet
  rejects each request without posting `invalid_request` (RI-WALLET-002).
  `WS_RP_MS_ProtocolMessages__041` fails because the Wallet does not support
  the `redirect_uri:` prefix at all. `007` is weaker than it looks: the Wallet
  rejects `decentralized_identifier:` as unsupported before any key check, and
  beta signs even its normal DID requests with the X.509 key
  (MOCK-VERIFIER-002).
- `WS_RP_IA_MainInteraction__064` and `066` share
  `verifier-response-controls` with 125 and 126. 064 fails on wallet
  2026.09.42 (30/09/2026): the Wallet posted a verified presentation, beta
  answered HTTP 400 with `{ "redirect_uri": "..." }`, and the Wallet showed
  `Verifier rejected the response` without opening the URI. That run used a
  `request_mutation` to strip the `redirect_uri` beta then signed into the
  request (MOCK-VERIFIER-003, since fixed); the mutation is gone and the
  unmutated session has not been rerun. 066 has the same limit as 125 and 126:
  its only evidence of the Wallet's own error is visual, so
  `evidence.non_empty` would also pass for a Wallet that shows success. Its
  reference-Wallet run is still pending.
- `WS_RP_IA_MainInteraction__057`, `061`, `067`, `WS_RP_IA_Supportive__001`,
  and `WS_RP_MS_ProtocolMessages__127`, `128` configure a `redirect_uri` that
  beta signed into the request until MOCK-VERIFIER-003 was fixed on
  30/09/2026, so a conformant Wallet rejected them. They need a
  reference-Wallet run after the fix.

### Constructible from the current contract, definition pending

Reclassified on 28/09/2026 against the Capture Wallet contract at `1c8162c`
and upstream `FCAF_FIXTURES.md` and `REMAINING_WORK.md` at the same commit.
Beta accepted every control below on that date and the delivered request
carried the property; see the dated probe in `CAPTURE_WALLET_API.md`. None of
the remaining entries has a scenario yet. Most currently bind the positive
`pipeline.dcql.metadata` exchange with a placeholder `credentials_match`
assertion, which proves nothing about the source; each needs its own scenario,
exact evidence bindings, and a reference-Wallet run.

Wallet-profile-dependent cryptography:

- [ ] `WS_RP_SH_Cryptography_CryptographicHash_006` (captured `wallet_metadata`
  from a POST retrieval; the verdict depends on whether the Wallet profile
  allows other hash algorithms)
- [ ] `WS_RP_SH_Cryptography_CryptographicHash_007` (generated client metadata
  advertises SHA-256 only; each decoded presentation records
  `digest_algorithm`)
- [ ] `WS_RP_SH_Cryptography_Encryption_002` (`client_metadata` narrowed to
  `encrypted_response_enc_values_supported` without `A256GCM`; the reference
  Wallet supports more than A256GCM, so expect `not_applicable`)

## Blocked

The Capture contract supplies only the capabilities documented in
`CAPTURE_WALLET_API.md`. Each entry names the input, credential fixture,
transport capture, Wallet profile, or verifier behaviour that is absent, with
the upstream `REMAINING_WORK.md` owner where one exists.

### Blocked on the Capture deployment certificate

The contract publishes `client_id_scheme: "verifier_attestation"`, the
`verifier_attestation` object (`subject`, `issuer`, `redirect_uris`, extra
`claims`, `signature: "corrupt"`), and `x509_san_dns`, so the entries below are
constructible from the contract. Beta refuses the session with `500
internal_error`, `there are no SAN-DNS names`, on 25/09/2026 and again on
28/09/2026, and production `capture-wallet.credimi.io` answers the same on
30/09/2026. Re-check after Capture publishes a SAN-matching certificate.

- [ ] `WS_RP_SM_RpIntegrity__001` (also needs a caller-signed `verifier_info`;
  see the verifier info entry below)
- [ ] `WS_RP_SM_RpIntegrity__008` (default verifier attestation, `cnf` holding
  the signing key)
- [ ] `WS_RP_SM_RpIntegrity__009` (`request_behavior.signing_key: unrelated`
  under verifier attestation)
- [ ] `WS_RP_SM_RpIntegrity__010` (also needs an operator to configure the
  fixture attestation issuer at
  `/openid4vp/verifier-attestation-issuer/jwks.json` as trusted in the Wallet)
- [ ] `WS_RP_SM_RpIntegrity__011` (`verifier_attestation.issuer`)
- [ ] `WS_RP_SM_RpIntegrity__012` (`verifier_attestation.signature: corrupt`)
- [ ] `WS_RP_MS_Metadata__116`, `118`, `122`, `124` (default verifier
  attestation)
- [ ] `WS_RP_MS_Metadata__117` (`verifier_attestation.subject`)
- [ ] `WS_RP_MS_Metadata__119` (`request_mutation.request_object_header.unset`
  `/jwt`)
- [ ] `WS_RP_MS_Metadata__120`, `121` (`verifier_attestation.redirect_uris`)
- [ ] `WS_RP_MS_Metadata__123` (verifier attestation plus non-key metadata
  outside `client_metadata`)
- [ ] `WS_RP_MS_Metadata__125`, `127`, `128` (`x509_san_dns` happy path and
  redirect-URI host mismatch. Upstream: the EUDI service-provider registry
  issues no certificate with a `dNSName` SAN, so a registry-trusted request
  cannot use this prefix even after Capture changes its certificate)

### Blocked on a missing Capture Wallet capability

- [ ] `WS_RP_IA_MainInteraction__024` and `WS_RP_MS_Metadata__134` (encrypted
  Request Object delivery is not implemented; upstream 5.4. Upstream lists
  `134` as needing only the captured `wallet_metadata`, but its source also
  requires the Wallet to decrypt an encrypted Request Object)
- [ ] `WS_RP_IA_Metadata__011`, `012`, `013` (dynamic and static discovery,
  including controlled SIOPv2 `aud` values; Credo-TS signs only DID,
  `x509_hash`, `x509_san_dns`, and unsigned `redirect_uri` requests; upstream
  5.9)
- [ ] `WS_RP_SM_RpIntegrity__013c_UF` (no Credimi definition; a Request Object
  signed with a controlled unacceptable algorithm; upstream 5.3)
- [ ] `WS_RP_SM_RpIntegrity__030` (a multi-signed Request Object; upstream 5.3)
- [ ] `WS_RP_SM_RpIntegrity__032` and
  `WS_RP_SM_RpIntegrity_CryptographicSignature_002` (RS384 or PS384; the KMS
  generates EC keys only and the intended algorithm is an open upstream
  decision)
- [ ] `WS_RP_SM_RpIntegrity__033`, `034`,
  `WS_RP_SM_RpIntegrity_CryptographicSignature_003`, `004` (COSE `-7` versus
  `-9`; both are ECDSA P-256 with SHA-256, expressible in a JAR only as
  `ES256`; upstream expects a source reclassification)
- [ ] `WS_RP_SM_RpIntegrity__025` (the Wallet's configured trust anchor inside
  `x5c`; the service does not hold it)
- [ ] `WS_RP_MS_ProtocolMessages__040` and `WS_RP_IA_MainInteraction__060`
  (a response mode other than `direct_post`; Capture supports `direct_post`,
  `direct_post.jwt`, `dc_api`, and `dc_api.jwt` only)
- [ ] `WS_RP_IA_MainInteraction__006`, `008`, `010` and
  `WS_RP_MS_CredentialFormats__046` (a credential without cryptographic holder
  binding; Credo refuses `require_cryptographic_holder_binding: false` during
  issuance until
  [credo-ts#2936](https://github.com/openwallet-foundation/credo-ts/pull/2936))
- [ ] `WS_RP_MS_CredentialFormats__048` (SD-JWT VC JSON serialization; issuer
  and verifier support compact serialization only)
- [ ] `WS_RP_MS_Metadata__089` gap and `WS_RP_MS_Metadata__099` (missing-scheme,
  unencoded-space, invalid-percent-encoding, illegal-character and empty URI
  shapes; `status_reference: malformed_uri` expresses only the unparseable one)
- [ ] `WS_RP_SM_IssuerIntegrity__012` (an ISO mdoc revocation fixture; Capture
  publishes Token Status List allocation only)
- [ ] `WS_RP_MS_Metadata__101`, `102` and `WS_RP_MS_CredentialFormats__032`
  (a CWT Referenced Token with the Status claim at CBOR label 65535, and for
  102 one without it. Capture issues only SD-JWT VC and ISO mdoc; the mdoc
  Mobile Security Object keys its status by the text `"status"`, not label
  65535. `cose.cwt_status_claim` reports the mdoc evidence as blocked. Upstream
  `FCAF_FIXTURES.md` maps these tests to the mdoc configurations)

### Blocked on Credimi-signed or profile-defined attestations

- [ ] `WS_RP_SM_DeviceBinding__002`–`006` (Capture delivers `verifier_info`
  verbatim and signs none: the caller must sign the attestation and its proof
  over the chosen `nonce` and the Client Identifier. The EUDI concrete type,
  the Relying Party Registration Certificate, has no TS5 attachment structure,
  so there is no profile format for the reference Wallet to validate, and the
  harness has no attestation-signing step)

### Blocked on external trust infrastructure

Upstream leaves the scope of 5.6–5.8 as an open decision.

- [ ] `WS_RP_SM_TrustMechanisms__101`, `101b_UF`, `101c_UF` (no Credimi
  definitions; WRPAC certificate fixtures; upstream 5.6)
- [ ] `WS_RP_IA_Metadata__014`, `WS_RP_MS_Metadata__112`, `113`, `114`, `115`
  (OpenID Federation entity statements, trust chain, and authority; upstream
  5.7)
- [ ] `WS_RP_SM_TrustMechanisms__002`–`013`, `015`, `021`,
  `WS_RP_MS_ProtocolMessages__095`, and `WS_RP_IA_MainInteraction__065`
  (issuer chains with controlled Authority Key Identifiers and ETSI trusted-list
  fixtures; upstream 5.8)
- [ ] `WS_RP_MS_ProtocolMessages__145`, `146` (Wallet-local or trusted-registry
  verifier metadata and the required `invalid_client` conflict)

### Blocked on the reference Wallet profile or environment

- [ ] `WS_RP_IA_Engagement__001b` (no Credimi definition; the reference Android
  Wallet registers no `eu-eaap://` handler)
- [ ] `WS_RP_IA_ProtocolFlow__002d` (no Credimi definition) and
  `WS_RP_MS_ProtocolMessages__011` (a Wallet that does not support
  `request_uri_method=post`)
- [ ] `WS_RP_SM_RpIntegrity__024` (a Wallet profile rejecting every
  non-`x509_hash` Client Identifier)
- [ ] `WS_RP_UC_Presentation__004` (an independently controllable second-device
  invocation path)
- [ ] `WS_RP_IA_Supportive__006` (deliberate Wallet device resource exhaustion)
- [ ] `WS_RP_IA_MainInteraction__046` (a credential large enough to exceed URL
  or transport limits)
- [ ] `WS_RP_MS_ProtocolMessages__151` (an unsupported mdoc format variant; the
  reference Wallet supports `mso_mdoc`)

### Blocked on scope and transaction-data support

- [ ] `WS_RP_MS_ProtocolMessages__020`, `141` and `WS_RP_UC_Presentation__003`
  (a scope the Wallet resolves to a DCQL query; the service defines no scope
  values)
- [ ] `WS_RP_MS_ProtocolMessages__017`, `018`, `135`, `154`–`159` and
  `WS_RP_SH_Cryptography_CryptographicHash_008` (a transaction-data type the
  reference Wallet supports; `008` declares `transaction_data_hashes_alg:
  ["sha-512"]` on such an entry)
