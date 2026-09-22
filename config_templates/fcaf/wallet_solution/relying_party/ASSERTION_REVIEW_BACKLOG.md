<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# FCAF assertion review backlog

This is the dedicated worklist for source tests that have no Credimi definition,
and for defined tests whose assertions remain pending or blocked. The tests that
already have a definition remain in the generated aggregate pipeline;
completing an item means reviewing its source scenario, pipeline evidence, and
assertions rather than removing it from execution.

Review baseline: the local source mirror under
`config_templates/fcaf_sources/wallet_solution/relying_party/`, cross-checked
against upstream `submitted` commit `2b223b56be0d0a073ee0cdc9db1d7fd31d9529a1`
(13/08/2026), and the Capture Wallet contract in
`pkg/fcaf/CAPTURE_WALLET_API.md`. The source mirror has 621 distinct
`WS_RP_*` files and Credimi has 595 matching test definitions. The remaining
source-test difference is listed below; it is deliberately separate from
assertion work that is incomplete or incorrect.

The lists below are the active review state; test-definition totals are kept in
the baseline above and validated by the FCAF catalog loader.

## Reclassified after Capture Wallet API refresh

The current Capture Wallet contract adds named PID fixtures, status-reference
fixtures, Request Object mutations, request-delivery behaviours, configurable
verifier responses, and DC API session creation. The following source tests are
**constructible but unimplemented**. Each remains pending until its exact source
scenario, delivered request, reference-Wallet behaviour, and protocol evidence
are verified on beta; none is a conformance pass merely because the API accepts
its setup.

### Constructible pending beta evidence

- **Implemented; beta evidence pending:** `WS_RP_MS_ProtocolMessages__003_UF`,
  `006`, `007`, `009`, `010`, `016`, `033`, `034`, `049`, `051`, and
  `WS_RP_SM_RpIntegrity__027`. The scenarios use `request_mutation` or
  `request_behavior.signature=corrupt` and bind the delivered JAR, outer
  request where applicable, retrieval, and Wallet outcome.
- **Implemented; beta evidence pending:** `WS_RP_MS_ProtocolMessages__046`,
  `048`, `WS_RP_MS_Metadata__139`, `140`, and `WS_RP_IA_Supportive__002`.
  The scenarios use `request_behavior.wallet_nonce` or
  `request_uri_response`, require POST Request URI retrieval, and bind the
  delivered request or response plus the Wallet outcome.
- **Implemented; beta evidence pending:**
  `WS_RP_IA_MainInteraction__053`, `055`, `056`,
  `WS_RP_IA_Metadata__010`, and `WS_RP_MS_ProtocolMessages__124`–`128`, `132`.
  `request_mutation` builds the missing, duplicated, and foreign-host Response
  URI requests and the unrecognized request parameter; `response_scenario`
  delivers the non-JSON, HTTP 400, and unrecognized-member verifier replies.
  Every case binds the captured Wallet HTTP request and the delivered verifier
  response. Open limitation: 125 and 126 can only evidence the Wallet's own
  error visually, because the Wallet has already submitted its Authorization
  Response when the malformed reply arrives.
- **Implemented; beta evidence pending:** `WS_RP_IA_MainInteraction__032`,
  `040`, `041`, `WS_RP_MS_CredentialFormats__033`, `044`, and
  `WS_RP_SH_Encoding_TextualEncoding_002`, `003`. Each case now issues the
  named fixture it needs and proves the fixture reached the Wallet before
  asserting the behaviour. 040 and 041 issue `pid_default` and `pid_person_b`
  and establish simultaneous possession through two value-constrained probes
  whose `document_number` values differ, so counting presentations can no
  longer be satisfied by one credential presented twice.
- **Implemented; beta evidence pending:** `WS_RP_MS_Metadata__081`–`090` and
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

### Available but deferred by selected scope

Capture now supports `dc_api` and `dc_api.jwt`, so the following are not
service-blocked: `WS_RP_IA_Engagement__002`,
`WS_RP_IA_ProtocolFlow__003a`, `003b_UF`, and
`WS_RP_SM_RpIntegrity__002`–`005`, `022`. They remain deferred because the
current FCAF work selection excludes Digital Credentials API cases. Do not
implement them without a scope change.

## Resolved assertion defects

Kept because the reasoning is the reviewable part: each entry records a
definition that ran in the aggregate and reported a pass for the wrong reason,
and what now decides it.

### `WS_RP_IA_MainInteraction__033`: DCQL value matching was never checked

**What the source asks.** The Wallet must hold eight credentials of the same
type and receive one DCQL query carrying several value constraints at once.
Exactly one credential satisfies all of them. Every other credential is a
deliberate near miss that fails exactly one constraint. The Wallet must release
the matching credential and must exclude all seven traps from the selection
prompt. This is a test of value matching, not of query parsing.

The Capture fixtures that realise those eight credentials are:

| Role | `fixture_id` | Differs along |
| --- | --- | --- |
| A, full match | `pid_default` | baseline: `family_name` `Rossi`, locality `Roma`, `age_over_18` `true`, `nationalities` `["IT"]`, `date_of_expiry` `2031-01-01` |
| B1, fails age | `pid_under_18` | `age_over_18: false` |
| B2, fails data type | *none* | no PID or degree claim is numeric; see the blocked list |
| B3, fails case | `pid_family_name_uppercase` | `family_name: "ROSSI"` |
| B4, fails whitespace | `pid_family_name_trailing_space` | `family_name: "Rossi "` |
| B5, fails encoding | `pid_locality_no_diacritics` | locality `Munchen` instead of `München` |
| B6, fails array | `pid_multiple_nationalities` | `nationalities: ["FR","DE"]` |
| B7, fails expiry bound | `pid_expiry_2032` | `date_of_expiry: 2032-01-01` |

**What the definition asserted.** One `dcql.response_satisfies_constraints` in
`credentials_match` mode over the shared `pipeline.dcql.main-interaction`
exchange, plus a screenshot check.

**Why that decided nothing.** `credentials_match` reads the captured
`dcql_query`, checks the credential queries are well formed, and then requires
`vp_token[<query id>]` to be non-empty. It never opens the returned
presentation. It therefore could not see which credential came back, and in
particular never compared a disclosed claim against the `values` restriction
that selected it. A Wallet that ignored value matching entirely and released
`pid_family_name_uppercase` produces a non-empty `vp_token` under the same
query id and passed. The assertion answered "did the Wallet answer at all",
while the source asks "did the Wallet answer with the one credential that
satisfies every constraint".

**What identifies the credential.** Not the set of disclosed claims: all eight
fixtures are complete PIDs, so a query for `family_name`, `address.locality`
and `nationalities` makes every one of them disclose exactly that same set. The
discriminator is the disclosed **value**. A presentation disclosing
`family_name: "Rossi"` is A and cannot be B3 (`"ROSSI"`) or B4 (`"Rossi "`);
one disclosing locality `Roma` cannot be B5. Asserting every constrained claim
carries a value from its `values` list is therefore both necessary and
sufficient to name the released credential.

**How it is decided now.** `fcaf-wallet-solution-relying-party-dcql-combined-value-constraints`
issues the eight fixtures, proves they are all held through an unconstrained
`multiple: true` inventory query (`oid4vp.distinct_presentations`, minimum 8),
and then sends one query restricting five independent axes at once:
`family_name = Rossi`, `age_over_18 = true`, `address.locality = Roma`,
`nationalities[0] = IT` and `date_of_expiry = 2031-01-01`. Only `pid_default`
satisfies all five and every trap fails exactly one, so no single constraint
can carry the result. The new
`oid4vp.dcql_value_constraints_satisfied` validator reads the restrictions from
the delivered query itself and requires every released presentation to disclose
each restricted claim with a value from that claim's list.

**Why the query sets `multiple: true`.** With `multiple` omitted the Wallet
returns one credential; releasing the match while also treating traps as
matches would stay invisible. `multiple: true` makes the Wallet return every
credential it considered a match, so a released trap cannot hide behind the
matching one. The validator's `require_multiple` param refuses evidence
gathered without it.

**Caveat if minimal disclosure is asserted too.** "The Wallet disclosed only
the requested claims" is a separate and worthwhile property, but `status` lives
in the SD-JWT payload outside the issuer disclosure frame, so it is always
present and must not be counted as an extra disclosure.

**Interaction with the status-reference probes, resolved.** Implementing this
case means issuing B3–B7, which are the same fixtures the
malformed-`status_reference` rejection scenario probes by `document_number`. A
fixture's `document_number` is a claim value rather than a per-issuance serial
and the aggregate never clears wallet state, so those probes could no longer
treat "a credential came back" as "the malformed token was kept". They now use
`multiple: true` and `oid4vp.malformed_status_credential_absent`, which
inspects the `status` claim of every returned credential and fails only on the
malformed shape the issuer was asked to emit. A validly issued duplicate of the
same fixture is therefore no longer mistaken for a retained rejected token.

## Still blocked

The beta contract supplies only the capabilities documented in
`CAPTURE_WALLET_API.md`. These items require an input, credential fixture,
transport capture, Wallet profile, or verifier behavior that remains absent.

### Capture Wallet certificate controls

- [ ] `WS_RP_MS_Metadata__125` and `WS_RP_MS_Metadata__127` (Capture Wallet
  rejects `client_id_scheme: x509_san_dns` because its supplied X.509 leaf
  certificate has no DNS SAN. Re-enable only after the service publishes a
  matching certificate and the Request Object can be captured.)

### Missing controls for source tests without a Credimi definition

- [ ] `WS_RP_IA_Engagement__001b` (the beta Capture service can generate an
  `eu-eaap://` deeplink, but the reference Android Wallet has no registered
  handler for that scheme, so invocation cannot be evidenced)

- [ ] `WS_RP_IA_ProtocolFlow__002d` (requires a reference Wallet profile that
  does not support `request_uri_method=post`; beta has only the post-supporting
  reference Wallet)
- [ ] `WS_RP_SM_RpIntegrity__013b_UF` (requires a Request Object with a
  controlled invalid JWS signature)
- [ ] `WS_RP_SM_RpIntegrity__013c_UF` (requires a Request Object signed with a
  controlled unacceptable algorithm)
- [ ] `WS_RP_SM_TrustMechanisms__101` (requires a Wallet Relying Party
  Registration Certificate fixture; beta's normal X.509 verifier certificate
  is not an exposed WRPAC fixture)
- [ ] `WS_RP_SM_TrustMechanisms__101b_UF` (requires an invalid-signature WRPAC
  fixture)
- [ ] `WS_RP_SM_TrustMechanisms__101c_UF` (requires two WRPAC fixtures with a
  controlled organization mismatch)

### Reclassified from pending

- [ ] `WS_RP_IA_MainInteraction__024` (requires an encrypted request that remains deliverable and user-confirmable; Capture does not publish encrypted Request Object delivery)
- [ ] `WS_RP_IA_Metadata__012` (requires Static Discovery with a controlled SIOPv2 `aud`; it is not a supported client identifier scheme)
- [ ] `WS_RP_IA_Metadata__013` (requires a controlled invalid Static Discovery `aud`; it is not a supported client identifier scheme)
- [ ] `WS_RP_IA_Supportive__006` (requires deliberately exhausting the Wallet device; this is neither a verifier nor an issuer capability)
- [ ] `WS_RP_MS_ProtocolMessages__095` (requires an ETSI trusted-list fixture and Wallet trust-list resolution)
- [ ] `WS_RP_MS_ProtocolMessages__151` (requires an unsupported mdoc format, while the reference Wallet supports `mso_mdoc`)
- [ ] `WS_RP_SM_IssuerIntegrity__012` (requires an mdoc revocation fixture; Capture publishes Token Status List allocation, not an ISO mdoc revocation fixture)

### Signature, trust, and Digital Credentials API controls

- [ ] `WS_RP_SM_RpIntegrity_CryptographicSignature_002` (requires an RS384-signed presentation request)
- [ ] `WS_RP_SM_RpIntegrity_CryptographicSignature_003` (requires a COSE `-7` signed presentation request)
- [ ] `WS_RP_SM_RpIntegrity_CryptographicSignature_004` (requires a COSE `-9` signed presentation request)
- [ ] `WS_RP_SM_RpIntegrity__001` (requires a controllable invalid `verifier_info` attestation)
- [ ] `WS_RP_SM_RpIntegrity__007` (requires a DID document whose verification method deliberately excludes the signing key)
- [ ] `WS_RP_SM_RpIntegrity__008` (requires a controlled verifier attestation `cnf` key)
- [ ] `WS_RP_SM_RpIntegrity__009` (requires a controlled verifier attestation `cnf` key)
- [ ] `WS_RP_SM_RpIntegrity__010` (requires a trusted verifier-attestation issuer)
- [ ] `WS_RP_SM_RpIntegrity__011` (requires an untrusted verifier-attestation issuer)
- [ ] `WS_RP_SM_RpIntegrity__012` (requires an invalid verifier-attestation signature)
- [ ] `WS_RP_SM_RpIntegrity__014` (requires a signed X.509 request without `x5c`)
- [ ] `WS_RP_SM_RpIntegrity__015` (requires an X.509 request signed by a key outside its `x5c` leaf)
- [ ] `WS_RP_SM_RpIntegrity__017` (requires an incomplete or untrusted X.509 chain)
- [ ] `WS_RP_SM_RpIntegrity__019` (requires a broken or untrusted `x509_hash` chain)
- [ ] `WS_RP_SM_RpIntegrity__021` (requires verifier metadata outside `client_metadata` to be delivered in the Request Object)
- [ ] `WS_RP_SM_RpIntegrity__024` (requires a Wallet profile that rejects every non-`x509_hash` client identifier, contrary to the supported DID and SAN schemes)
- [ ] `WS_RP_SM_RpIntegrity__025` (requires injecting a trust-anchor certificate into `x5c`)
- [ ] `WS_RP_SM_RpIntegrity__026` (requires a self-signed request certificate)
- [ ] `WS_RP_SM_RpIntegrity__030` (requires a multi-signed Request Object)
- [ ] `WS_RP_SM_RpIntegrity__032` (requires an RS384-signed Request Object)
- [ ] `WS_RP_SM_RpIntegrity__033` (requires a COSE `-7` signed Request Object)
- [ ] `WS_RP_SM_RpIntegrity__034` (requires a COSE `-9` signed Request Object)

### Nonce, key-fixture, and trust-list controls

- [ ] `WS_RP_SM_TrustMechanisms__002` (requires an AKI-backed issuer certificate fixture)
- [ ] `WS_RP_SM_TrustMechanisms__003` (requires an AKI-backed matching credential fixture)
- [ ] `WS_RP_SM_TrustMechanisms__004` (requires an AKI-backed non-matching credential fixture)
- [ ] `WS_RP_SM_TrustMechanisms__005` (requires a match in an end-entity issuer certificate)
- [ ] `WS_RP_SM_TrustMechanisms__006` (requires a match in a sub-CA certificate)
- [ ] `WS_RP_SM_TrustMechanisms__007` (requires a match in a CA certificate)
- [ ] `WS_RP_SM_TrustMechanisms__008` (requires an AKI mismatch credential fixture)
- [ ] `WS_RP_SM_TrustMechanisms__009` (requires an ETSI trusted-list match fixture)
- [ ] `WS_RP_SM_TrustMechanisms__010` (requires an ETSI trusted-list no-match fixture)
- [ ] `WS_RP_SM_TrustMechanisms__011` (requires an invalid ETSI trusted-list fixture)
- [ ] `WS_RP_SM_TrustMechanisms__012` (requires an ETSI Trusted List identifier fixture)
- [ ] `WS_RP_SM_TrustMechanisms__013` (requires List of Trusted Lists navigation and TSP certificate evidence)
- [ ] `WS_RP_SM_TrustMechanisms__015` (requires a failed ETSI trust-chain fixture)
- [ ] `WS_RP_SM_TrustMechanisms__021` (requires multiple X.509 trust-mechanism fixtures)
- [ ] `WS_RP_UC_Presentation__003` (requires a defined scope-to-DCQL mapping)
- [ ] `WS_RP_UC_Presentation__004` (requires an independently controlled second-device invocation path)
- [ ] `WS_RP_IA_MainInteraction__046` (requires a credential/presentation fixture exceeding user-agent URL limits)

- [ ] `WS_RP_MS_ProtocolMessages__011` (needs a wallet configuration that does not support POST `request_uri` retrieval)
- [ ] `WS_RP_MS_ProtocolMessages__013` (needs two available credentials that satisfy one DCQL credential query)
- [ ] `WS_RP_MS_ProtocolMessages__017` (needs a wallet configuration without `transaction_data` support and a verifier callback capture)
- [ ] `WS_RP_MS_ProtocolMessages__018` (needs a verifier-supported valid `transaction_data` type and matching wallet capability)
- [ ] `WS_RP_MS_ProtocolMessages__020` (needs a verifier-supported scope value with a defined DCQL mapping)
- [ ] `WS_RP_MS_ProtocolMessages__030` (needs a scope-only request with a controlled unknown scope value)
- [ ] `WS_RP_MS_ProtocolMessages__039` (needs an unsigned request with a malformed or non-HTTPS `redirect_uri:` client identifier)
- [ ] `WS_RP_MS_ProtocolMessages__040` (needs a `redirect_uri:` client identifier request without `redirect_uri`)
- [ ] `WS_RP_MS_ProtocolMessages__041` (needs a direct_post.jwt request with `redirect_uri:` client identifier and no `response_uri`)
- [ ] `WS_RP_MS_ProtocolMessages__043` (needs a controllable HTTP request_uri endpoint)

### Remaining issuer status-list controls

`status_reference` now provides the SD-JWT status shapes moved to the
constructible list above. Capture still cannot create the COSE/ISO mdoc shapes,
custom CBOR labels, or OpenID Federation and verifier-attestation fixtures
required by the following source tests.
- [ ] `WS_RP_MS_Metadata__094` (needs a COSE referenced token without status_list)
- [ ] `WS_RP_MS_Metadata__095` (needs a COSE token with controlled unsigned status_list.idx encoding)
- [ ] `WS_RP_MS_Metadata__096` (needs malformed COSE status_list.idx encodings)
- [ ] `WS_RP_MS_Metadata__097` (needs a COSE referenced token with status_list.idx omitted)
- [ ] `WS_RP_MS_Metadata__098` (needs a COSE token with controlled status_list.uri encoding)
- [ ] `WS_RP_MS_Metadata__099` (needs malformed status_list.uri variants in an issued COSE token)
- [ ] `WS_RP_MS_Metadata__100` (needs a COSE referenced token with status_list.uri omitted)
- [ ] `WS_RP_MS_Metadata__101` (needs a COSE referenced token with a status claim at label 65535)
- [ ] `WS_RP_MS_Metadata__102` (needs a COSE referenced token with label 65535 omitted)
- [ ] `WS_RP_MS_Metadata__103` (needs issuer control of StatusListInfo CBOR map shape and labels)
- [ ] `WS_RP_MS_Metadata__110` (needs a signed redirect_uri-prefixed Request Object)
- [ ] `WS_RP_MS_Metadata__112` (needs a valid OpenID Federation trust chain)
- [ ] `WS_RP_MS_Metadata__113` (needs malformed or untrusted OpenID Federation chains)
- [ ] `WS_RP_MS_Metadata__114` (needs OpenID Federation metadata resolution with conflicting client_metadata)
- [ ] `WS_RP_MS_Metadata__115` (needs OpenID Federation metadata resolution without client_metadata)
- [ ] `WS_RP_MS_Metadata__116` (needs a valid verifier attestation JWT and matching client identifier)
- [ ] `WS_RP_MS_Metadata__117` (needs a verifier attestation JWT with a mismatched subject)
- [ ] `WS_RP_MS_Metadata__118` (needs a verifier attestation JWT in the Request Object JOSE header)
- [ ] `WS_RP_MS_Metadata__119` (needs a verifier_attestation request missing the jwt header)
- [ ] `WS_RP_MS_Metadata__120` (needs a controllable verifier attestation redirect_uris claim)
- [ ] `WS_RP_MS_Metadata__121` (needs a verifier attestation redirect_uris mismatch)
- [ ] `WS_RP_MS_Metadata__122` (needs a verifier attestation without redirect_uris)
- [ ] `WS_RP_MS_Metadata__123` (needs a verifier attestation request with external non-key metadata)
- [ ] `WS_RP_MS_Metadata__124` (needs a verifier attestation request with client_metadata-only non-key metadata)
- [ ] `WS_RP_MS_Metadata__126` (needs an x509_san_dns request with SAN mismatch)
- [ ] `WS_RP_MS_Metadata__128` (needs an x509_san_dns request with redirect URI hostname mismatch)
- [ ] `WS_RP_MS_Metadata__130` (needs an x509_hash request with leaf certificate hash mismatch)
- [ ] `WS_RP_MS_Metadata__132` (needs an x509_hash request signed by a different key)
- [ ] `WS_RP_MS_Metadata__133` (needs an origin-prefixed request outside the Digital Credentials API)
- [ ] `WS_RP_MS_Metadata__134` (needs verifier-issued encrypted Request Objects from POST wallet metadata)
- [ ] `WS_RP_MS_Metadata__135` (needs a verifier to return an unencrypted Request Object after encryption negotiation)
- [ ] `WS_RP_MS_Metadata__136` (needs POST wallet metadata for a verifier signing-capable client identifier prefix)
- [ ] `WS_RP_MS_Metadata__137` (needs POST wallet metadata for a redirect_uri-prefixed request)

- [ ] `WS_RP_IA_MainInteraction__006`   (credential with no hb)
- [ ] `WS_RP_IA_MainInteraction__008`   (credential with no hb)
- [ ] `WS_RP_IA_MainInteraction__010`   (credential with no hb)
- [ ] `WS_RP_IA_MainInteraction__060`   (IMPOSSIBLE, credo does not support fragment/query)
- [ ] `WS_RP_IA_MainInteraction__064`
- [ ] `WS_RP_IA_MainInteraction__065`
- [ ] `WS_RP_IA_MainInteraction__066`
- [ ] `WS_RP_IA_Metadata__011`          (dynamic discovery)
- [ ] `WS_RP_IA_Metadata__014`          (support openid_federation prefix for client_id)
- [ ] `WS_RP_MS_CredentialFormats__029` (issue jwt with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__032` (issue cwt with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__041` (multiple mdoc credentials with different values)
- [ ] `WS_RP_MS_CredentialFormats__046` (requires credential with no key-binding)
- [ ] `WS_RP_MS_CredentialFormats__048` (strange json encoding vc, not to be done)
- [ ] `WS_RP_MS_Metadata__105`
- [ ] `WS_RP_MS_Metadata__106`
- [ ] `WS_RP_MS_Metadata__107`
- [ ] `WS_RP_MS_Metadata__109`
- [ ] `WS_RP_MS_ProtocolMessages__002`
- [ ] `WS_RP_MS_ProtocolMessages__135` (wallet shoudl support transaction_data)
- [ ] `WS_RP_MS_ProtocolMessages__141` (wallet does not have scope to dcql mapping)
- [ ] `WS_RP_MS_ProtocolMessages__143` (client_id prefix unsupported, at the moment not settable)
- [ ] `WS_RP_MS_ProtocolMessages__144` (client_id prefix HTTPS does not exists!)
- [ ] `WS_RP_MS_ProtocolMessages__145` (locally stored verifier metadata???)
- [ ] `WS_RP_MS_ProtocolMessages__146` (client_id resolves to a trusted registry?)
- [ ] `WS_RP_MS_ProtocolMessages__154` (non implementable due to the lack of know transaction type supported by the wallet)
- [ ] `WS_RP_MS_ProtocolMessages__155` (non implementable due to the lack of know transaction type supported by the wallet)
- [ ] `WS_RP_MS_ProtocolMessages__156` (non implementable due to the lack of know transaction type supported by the wallet)
- [ ] `WS_RP_MS_ProtocolMessages__157` (non implementable due to the lack of know transaction type supported by the wallet)
- [ ] `WS_RP_MS_ProtocolMessages__158` (non implementable due to the lack of know transaction type supported by the wallet)
- [ ] `WS_RP_MS_ProtocolMessages__159` (non implementable due to the lack of know transaction type supported by the wallet)
- [ ] `WS_RP_SM_DeviceBinding__002`
- [ ] `WS_RP_SM_DeviceBinding__003`
- [ ] `WS_RP_SM_DeviceBinding__004`
- [ ] `WS_RP_SM_DeviceBinding__005`
- [ ] `WS_RP_SM_DeviceBinding__006`
- [ ] `WS_RP_SH_Cryptography_Encryption_002` (requires a Wallet that support only A256GCM and verifier info that not include it)
- [ ] `WS_RP_SH_Cryptography_CryptographicHash_006` (requires a Wallet Metadata retrieval mechanism and a Wallet profile with another supported hash algorithm)
- [ ] `WS_RP_SH_Cryptography_CryptographicHash_007` (requires a Wallet profile with another supported hash algorithm and a defined client-metadata representation)
- [ ] `WS_RP_SH_Cryptography_CryptographicHash_008` (requires a verifier to use and expose a non-SHA-256 hashing function)
- [ ] `WS_RP_SH_Cryptography_CryptographicHash_010` (requires an issuer fixture using a non-SHA-256 credential digest algorithm)
