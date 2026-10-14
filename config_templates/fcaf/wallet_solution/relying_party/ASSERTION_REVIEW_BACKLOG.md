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
as synced on 30/09/2026 (`3d40a8d`). The mirror has 621 distinct `WS_RP_*` files
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
- **Implemented; Capture evidence pending:** `WS_RP_IA_Metadata__011`, `012`,
  and `013`. The interaction-metadata scenario keeps its default session,
  whose Request Object carries `aud: https://self-issued.me/v2`, for 012, and
  adds an aud-mismatch session that sets `/aud` through `request_mutation` for
  011 and 013. `oid4vp.request_audience` reads the Wallet's discovery mode from
  the POSTed `wallet_metadata` (`issuer` present means dynamic): 011 is
  `not_applicable` for a static-discovery Wallet such as 2026.09.42.
- **Implemented; Capture evidence pending:** `WS_RP_SM_RpIntegrity__024`,
  `030`, `033`, `034`, and `WS_RP_SM_RpIntegrity_CryptographicSignature_003`,
  `004`. The rp-integrity default session (ES256, `x5c`, POST retrieval) serves
  033, 034, 003 and 004, which assert the signature, `alg: ES256`, and a
  presentation; for `-9` the sources' expected results name JOSE `ES256`, and
  production refuses `ESP256`. A `decentralized_identifier` session signed with
  the DID key serves 024, which accepts an error or a discontinuation
  (`request_rejected`). The `dc-api-multisigned` scenario serves 030: production
  created `request_delivery: multisigned` sessions on 08/10/2026, with two
  `ES256` signatures for the `x509_hash` and `decentralized_identifier` Client
  Identifiers.
- **Implemented; Capture evidence pending:**
  `WS_RP_SH_Cryptography_CryptographicHash_008`. It reuses the
  `CryptographicHash_010` session: a SHA-384-digested `pid_person_b` requested
  under client metadata naming only SHA-256. A Wallet that presents it supports
  SHA-384 and is `not_applicable` (wallet 2026.09.42 did for 010); otherwise
  `sdjwt.presentation_digest_algorithm_unsupported` with `require_error` needs
  an error and no presentation.
- **Implemented; Capture evidence pending:** `WS_RP_SM_TrustMechanisms__002`,
  `003`, `004`, `008`, and `021`. The issuer chain needs no control: the Capture
  issuer certificate's AuthorityKeyIdentifier is
  `QlBQvhC4EPCdRFyNv6sQCO4n3Ek` (PID Issuer CA 02, probed 08/10/2026), which
  also covers the onboarding PID. The trust-mechanisms query carries it for
  002, 003 and 021 (`trusted_authorities_match`). A second session with an
  `aki` no certificate carries serves 004 and 008 (`trusted_authorities_no_match`,
  `fcaf-expect-no-matching-document`).
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
- **Implemented; Capture evidence pending:** the Wallet-profile cryptography
  cases. `WS_RP_SH_Cryptography_CryptographicHash_006` and `007` join
  `credential-digest-algorithm`. 006 reads the `wallet_metadata` POSTed during
  the 010 exchange and treats the SHA-384 presentation as proof that the Wallet
  supports another hash algorithm, so it is `not_applicable` when the Wallet
  withholds it. Because no metadata member is defined for hash algorithms, any
  IANA hash name counts. 007 issues `pid_expiry_2032` digested with SHA-384 and
  with SHA-256, requests it with the generated client metadata (which names no
  hash algorithm), and requires only SHA-256 presentations.
  `WS_RP_SH_Cryptography_Encryption_002` joins `response-encryption-a128gcm`;
  it is `not_applicable` unless the POSTed `wallet_metadata` lists A256GCM
  alone, and otherwise requires `access_denied`. Expected on wallet 2026.09.42:
  006 fails, because its metadata names no hash algorithm although it presented
  SHA-384; 007 depends on the default consent option; Encryption_002 is
  `not_applicable`.

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


### Reclassified from Blocked, definition pending

Reviewed on 30/09/2026 against each source, upstream `FCAF_CAPABILITY_MATRIX.csv`
(`de49603`), and session-creation probes on production
`capture-wallet.credimi.io`. Each construction below was accepted and the
delivered request carried the property; no Wallet has run them yet.

Constructible from existing controls:

- [x] `WS_RP_MS_Metadata__119` (`request_mutation` sets `client_id` to
  `verifier_attestation:capture-wallet.credimi.io` on the default session; the
  delivered header has no `jwt`, which is the defect. No attestation session is
  needed). Resolved 14/10/2026: the source precondition is unbuildable, so the
  test asserts `fcaf.skipped` naming the missing certificate (see the
  certificate entry below).
- [x] `WS_RP_MS_Metadata__113` (`request_mutation` sets an `openid_federation:`
  `client_id` and a malformed `trust_chain` JOSE header; the source expects
  rejection). Resolved 14/10/2026: Capture rejects `openid_federation` with
  `400 unsupported_client_id_scheme` (probed 14/10/2026), so the session cannot
  be created; the test asserts `fcaf.skipped` with that reason.
- [x] `WS_RP_MS_ProtocolMessages__040` (plain request, `client_id`
  `redirect_uri:<URI>` with `response_mode=fragment` and `response_uri`
  removed through `outer_request` mutation. `<URI>` is a second session's
  capture redirect page with its `response_code`, so a recorded visit on that
  session proves the Wallet derived the target from the Client Identifier.
  Upstream rates it category A. Wallet 2026.09.42 rejects the `redirect_uri:`
  prefix, so it is expected to fail). Resolved 14/10/2026: the source relevancy
  is `EUDI_forbidden`, so the acceptance clause is `fcaf.not_applicable` and the
  request-shape assertions are kept.
- [x] `WS_RP_MS_ProtocolMessages__017` (`transaction_data` of an unknown type;
  `invalid_transaction_data` passes, a presentation carrying
  `transaction_data_hashes` makes it `not_applicable` because the Wallet then
  supports the parameter). Resolved 14/10/2026: implemented as
  `jwt.payload_field_presence` + `dcql.response_satisfies_constraints{mode:
  transaction_data_error_required}` on the delivered request and the Wallet's
  error response.

Profile-conditional, decided by evidence like `WS_RP_MS_Metadata__135` and
`WS_RP_SH_Cryptography_Encryption_002`: each applies only to a Wallet lacking a
capability, and the verdict is `not_applicable` when the Wallet's own metadata
or behaviour shows it has it.

- [x] `WS_RP_MS_ProtocolMessages__011` (Wallet without POST Request URI
  retrieval; a POST retrieval shows support). Resolved 14/10/2026: the
  reference Wallet retrieves the Request Object with HTTP POST, so the source's
  GET-fallback premise is false for this SUT and the acceptance clause is
  `fcaf.not_applicable` (profile-conditional); the retrieval and request-shape
  assertions are kept.
- [ ] `WS_RP_IA_ProtocolFlow__002d` (same condition; no Credimi definition yet.
  Its example invocation is `eu-eaap://`, which the reference Wallet does not
  handle, so use the default scheme and record the deviation)
- [ ] `WS_RP_MS_ProtocolMessages__151` (Wallet without `mso_mdoc`;
  `vp_formats_supported` in the POSTed `wallet_metadata` shows support)
- [x] `WS_RP_MS_Metadata__134` (Wallet requiring encrypted Request Objects;
  applies only when `wallet_metadata` publishes `jwks`, which wallet 2026.09.42
  does not. For a Wallet that does, see the Request Object encryption entry
  below). Resolved 14/10/2026: the reference Wallet publishes no `jwks`, so the
  premise does not hold and the test asserts `fcaf.not_applicable`.

Published in the Capture contract at `3d40a8d`, not yet deployed: on
30/09/2026 production and beta both refused `request_behavior:
{"request_object_encryption": "none"}` as `invalid_request_behavior`, and served
a plain JWS to a POST retrieval whose `wallet_metadata` carried `jwks`. Re-probe
after deployment. (`request_delivery: "multisigned"`, refused on the same date,
is deployed on production since 08/10/2026; `WS_RP_SM_RpIntegrity__030` uses
it.)

- [ ] `WS_RP_MS_Metadata__134` for a Wallet publishing `jwks` (POST retrieval
  returns a Nested JWT, `alg: ECDH-ES`, `cty: JWT`, kept in
  `raw.authorization_request_jwe`; `raw.authorization_request_jwt` stays the
  enclosed signed JWT). Capture negotiates the RFC 8414
  `request_object_encryption_*` members, while the source names
  `authorization_encryption_*`; upstream records that as an open decision
- [ ] Fix `WS_RP_MS_Metadata__135` once encryption is deployed: its scenario
  must set `request_behavior.request_object_encryption: none`, and
  `oid4vp.unencrypted_request_object_rejected` must also require
  `raw.authorization_request_jwe` to be absent. Otherwise a `jwks`-publishing
  Wallet receives an encrypted object while the validator, reading
  `authorization_request_jwt`, treats the delivery as unencrypted

Found during the same review: MOCK-VERIFIER-002 does not reproduce on
production, which signs `decentralized_identifier` requests with the DID key.
`WS_RP_IA_Metadata__015`, `016` and `WS_RP_SM_RpIntegrity__006`, previously
unable to pass on beta, need a production run.

### Known gaps inside implemented coverage

- `WS_RP_MS_ProtocolMessages__125`, `126` and `WS_RP_IA_MainInteraction__066`
  can only evidence the Wallet's own error visually: the Wallet has already
  submitted its Authorization Response when the malformed verifier reply
  arrives. Updated 14/10/2026: the three steps now store a named error
  screenshot (`non-json-wallet-error` / `http-error-wallet-error` /
  `no-redirect-wallet-error`) and assert `evidence.screenshot_named`, so a
  Wallet that stays on its success screen fails. On wallet 2026.09.42 the
  success screen is what 066 and 125 show (expected FAIL,
  `REFERENCE-WALLET-ISSUES.md` RI-WALLET-025); 126 shows
  `Oups! ... Verifier rejected the response` (pass).
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
  unmutated session has not been rerun. 066 kept the same limit as 125 and 126
  until 14/10/2026, when its step was inlined like theirs and it moved from
  `evidence.non_empty` to `evidence.screenshot_named{name:
  no-redirect-wallet-error}`. The emulator run of 14/10/2026 leaves the
  reference Wallet on its success screen when the reply carries no
  `redirect_uri`, so 066 is expected to FAIL (the source's "raises an
  invalid_request error or terminates transaction" is a judgment:
  `REFERENCE-WALLET-ISSUES.md` RI-WALLET-025).
- `WS_RP_IA_MainInteraction__057`, `061`, `067`, and
  `WS_RP_MS_ProtocolMessages__127`, `128` configure a `redirect_uri` that
  beta signed into the request until MOCK-VERIFIER-003 was fixed on
  30/09/2026, so a conformant Wallet rejected them. They need a
  reference-Wallet run after the fix (`WS_RP_IA_Supportive__001` passed on
  13/10/2026).

### Unresolved after the 14/10/2026 passed-test audit

The 439 passed tests of `fcaf_assessment_4t9hxzgxlf.json` were re-audited (353
genuine passes, 80 false positives, 3 skips, 2 unclear; see
`pkg/fcaf/MEMORY.md`). Open items it left behind:

- [ ] `WS_RP_MS_ProtocolMessages__006` and `007` keep the `unclear` verdict: the
  source demands an `invalid_request` error and the run's evidence cannot show
  whether one was posted. Both now assert `invalid_request_required`; a fresh
  reference-Wallet run must confirm the verdict.
- [ ] `WS_RP_MS_CredentialFormats__034` still infers its `values` filter: no
  validator pins an mdoc element to a value and no second non-matching mdoc
  fixture is wired to the credential-formats scenario, so the assertion passes
  on either credential. Needs a new `mdoc.element_equals{namespace, element,
  value}` validator plus a second mdoc issuance (`fixture_id: pid_person_b`).
- [ ] `WS_RP_IA_MainInteraction__026` is expected to FAIL on the reference
  Wallet: it now asserts `access_denied_required` for source ER3, and the Wallet
  shows its local no-match notice and posts nothing (RI-WALLET-001/010 family,
  mirroring `WS_RP_MS_ProtocolMessages__106`). Confirm on a device.
- [ ] `WS_RP_IA_MainInteraction__013` expects a per-claim deselection the
  reference Wallet does not offer for DCQL requests (RI-WALLET-024). Decide
  between the strict failure and `fcaf.not_applicable` per the `.agents/HITL.md`
  entry of 2026-10-14.
- [ ] `WS_RP_IA_MainInteraction__066` and `WS_RP_MS_ProtocolMessages__125`
  leave the Wallet on its success screen when the response-endpoint reply is
  missing a `redirect_uri` / not JSON (RI-WALLET-025); 066's disposition is a
  judgment on the source's "terminates transaction". Confirm on a device.
- [ ] `oid4vp.request_uri_retrieval` cannot assert that the Wallet requested the
  `request_uri` path without query or fragment (source
  `WS_RP_IA_ProtocolFlow__002b` ER3.3): Capture records only
  `{body, headers, method}` in `raw.request_uri_http`. Capture must record the
  requested target before the clause is assertable.
- [ ] Mixed3's `VR-3` was truncated in the audit report: `vp_token_json_object`'s
  new array clause was reconstructed from `WS_RP_MS_ProtocolMessages__127a`'s
  source and the per-test pointer. Confirm against the untruncated report.
- [ ] The `credentials_match` family now consults Capture's `presentation_valid`
  / `dcql_query_matched`; `mdoc_without_claim_disclosures` was deliberately left
  out of that change (`WS_RP_IA_MainInteraction__017`). Add the verifier-verdict
  check there if the maintainers want it.
- [ ] Three optional validators were deliberately not added:
  `oid4vp.transaction_data_support`, `oid4vp.request_uri_method_fallback` and
  `mdoc.mso_status_member_absent` (two are conditional on a future Wallet run,
  one belongs to the mdoc owner).

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
30/09/2026 and again on 08/10/2026. Re-check after Capture publishes a
SAN-matching certificate. Until then 008–012 report `skipped` through
`fcaf.skipped`, as do `WS_RP_SM_RpIntegrity__001`, `025`, `032` and
`CryptographicSignature_002` below.

- [ ] `WS_RP_SM_RpIntegrity__008` (default verifier attestation, `cnf` holding
  the signing key)
- [ ] `WS_RP_SM_RpIntegrity__009` (`request_behavior.signing_key: unrelated`
  under verifier attestation)
- [ ] `WS_RP_SM_RpIntegrity__010` (also needs an operator to configure the
  fixture attestation issuer at
  `/openid4vp/verifier-attestation-issuer/jwks.json` as trusted in the Wallet)
- [ ] `WS_RP_SM_RpIntegrity__011` (`verifier_attestation.issuer`)
- [ ] `WS_RP_SM_RpIntegrity__012` (`verifier_attestation.signature: corrupt`)
- [x] `WS_RP_MS_Metadata__116`, `118`, `122`, `124` (default verifier
  attestation). Resolved 14/10/2026: expressed as `fcaf.skipped` naming the
  missing SAN-DNS certificate.
- [x] `WS_RP_MS_Metadata__117` (`verifier_attestation.subject`). Resolved
  14/10/2026: expressed as `fcaf.skipped` naming the missing SAN-DNS
  certificate.
- [x] `WS_RP_MS_Metadata__120`, `121` (`verifier_attestation.redirect_uris`).
  Resolved 14/10/2026: expressed as `fcaf.skipped` naming the missing SAN-DNS
  certificate.
- [x] `WS_RP_MS_Metadata__123` (verifier attestation plus non-key metadata
  outside `client_metadata`). Resolved 14/10/2026: expressed as `fcaf.skipped`
  naming the missing SAN-DNS certificate.
- [x] `WS_RP_MS_Metadata__125`, `127`, `128` (`x509_san_dns` happy path and
  redirect-URI host mismatch. Upstream: the EUDI service-provider registry
  issues no certificate with a `dNSName` SAN, so a registry-trusted request
  cannot use this prefix even after Capture changes its certificate). Resolved
  14/10/2026: expressed as `fcaf.skipped` naming the missing `dNSName` SAN.

### Blocked on a missing Capture Wallet capability

- [ ] `WS_RP_SM_RpIntegrity__013c_UF` (no Credimi definition; a Request Object
  signed with an unacceptable algorithm such as EdDSA under a Wallet-trusted
  certificate. Upstream: Credo signs EdDSA, but no trusted certificate exists
  for such a key, so the Wallet would reject on trust rather than on the
  algorithm)
- [ ] `WS_RP_SM_RpIntegrity__032` and
  `WS_RP_SM_RpIntegrity_CryptographicSignature_002` (RS384 or PS384; the KMS
  generates EC keys only and the intended algorithm is an open upstream
  decision)
- [ ] `WS_RP_SM_RpIntegrity__025` (the Wallet's trust anchor inside `x5c`,
  awaiting a verifier control that serves it. The anchor, `PID Issuer CA 02`,
  is public at `https://preprod.pki.eudiw.dev/aia/PIDIssuerCA02-EU.cacert.pem`,
  and on 30/09/2026 a `request_object_header` mutation appending it to `x5c`
  was delivered. A scenario would have to hard-code that certificate, because
  no pipeline step can read the leaf's AIA and convert PEM to `x5c`, so it
  would break when the anchor changes. The verifier will serve the anchor
  itself)
- [x] `WS_RP_IA_MainInteraction__006`, `008`, `010` and
  `WS_RP_MS_CredentialFormats__046` (a credential without cryptographic holder
  binding; Credo refuses `require_cryptographic_holder_binding: false` during
  issuance until
  [credo-ts#2936](https://github.com/openwallet-foundation/credo-ts/pull/2936),
  still open on 13/10/2026). Resolved 14/10/2026: 006/008/010 and 046 all
  report `skipped` through `fcaf.skipped`; 006/010 name the unbound and
  mixed-binding preconditions respectively.
- [ ] `WS_RP_MS_CredentialFormats__048` (SD-JWT VC JSON serialization; issuer
  and verifier support compact serialization only). It reports `skipped`
  through `fcaf.skipped`
- [ ] `WS_RP_MS_Metadata__089` gap and `WS_RP_MS_Metadata__099` (missing-scheme,
  unencoded-space, invalid-percent-encoding, illegal-character and empty URI
  shapes; `status_reference: malformed_uri` expresses only the unparseable one)
- [ ] `WS_RP_SM_IssuerIntegrity__012` (an ISO mdoc revocation fixture; Capture
  publishes Token Status List allocation only)
- [x] `WS_RP_MS_Metadata__101`, `102` and `WS_RP_MS_CredentialFormats__032`
  (resolved 14/10/2026: Token Status List §6.3 places the mdoc Status CBOR
  structure under the MSO text label `status`; 032 and 101 assert it with the
  `mdoc.mso_status_*` validators, 102 issues an mdoc without status and probes
  its rejection like the 092 family; `cose.cwt_status_claim` was removed)

### Blocked on Credimi-signed or profile-defined attestations

- [ ] `WS_RP_SM_RpIntegrity__001` (delivering a `verifier_info` attestation
  with a broken signature needs no signing, but the source applies only when
  the Wallet chooses to use `verifier_info`, and nothing shows whether the
  reference Wallet does. A presentation would be indistinguishable between
  ignoring the attestation and accepting a bad one)
- [x] `WS_RP_SM_DeviceBinding__002`–`006` (Capture delivers `verifier_info`
  verbatim and signs none: the caller must sign the attestation and its proof
  over the chosen `nonce` and the Client Identifier. The EUDI concrete type,
  the Relying Party Registration Certificate, has no TS5 attachment structure,
  so there is no profile format for the reference Wallet to validate, and the
  harness has no attestation-signing step). Resolved 14/10/2026: all five report
  `skipped` through `fcaf.skipped` with the missing attestation-signing step as
  the reason.

### Blocked on external trust infrastructure

Upstream leaves the scope of 5.6–5.8 as an open decision.

- [ ] `WS_RP_SM_TrustMechanisms__101`, `101b_UF`, `101c_UF` (no Credimi
  definitions; WRPAC certificate fixtures; upstream 5.6)
- [x] `WS_RP_IA_Metadata__014`, `WS_RP_MS_Metadata__112`, `114`, `115`
  (OpenID Federation entity statements, trust chain, and authority; upstream
  5.7). Resolved 14/10/2026: `112` asserts `fcaf.not_applicable` (the
  `openid_federation` Client Identifier Prefix is `EUDI_forbidden`); `114` and
  `115` report `skipped` through `fcaf.skipped` because Capture rejects
  `openid_federation` with `400 unsupported_client_id_scheme`; `014` was
  already `skipped`.
- [ ] `WS_RP_SM_TrustMechanisms__005`–`007` (an issuer chain of depth 3 or
  more with a controlled Authority Key Identifier in the leaf, a sub-CA or the
  CA; Capture's chain is the issuer leaf plus PID Issuer CA 02), `009`–`013`,
  `015` (a mock ETSI Trusted List) (upstream 5.8). The TrustMechanisms tests
  report `skipped` through `fcaf.skipped`. Note
  14/10/2026: `WS_RP_MS_ProtocolMessages__095` now reports `fcaf.skipped` (no
  ETSI Trust List fixture), and `WS_RP_IA_MainInteraction__065` is implemented
  (`dcql.response_satisfies_constraints{mode: trusted_authorities_match}` on
  `pipeline.dcql.trust-mechanisms`) instead of blocked.
- [ ] `WS_RP_MS_ProtocolMessages__145`, `146` (Wallet-local or trusted-registry
  verifier metadata and the required `invalid_client` conflict)

### Blocked on the reference Wallet profile or environment

- [ ] `WS_RP_IA_Engagement__001b` (no Credimi definition; the reference Android
  Wallet registers no `eu-eaap://` handler)
- [x] `WS_RP_IA_MainInteraction__024` (the Wallet holds a matching credential
  whose data is encrypted or locked, so it cannot check a `values` restriction.
  This is Wallet state, not Request Object encryption, which it was previously
  filed under. Upstream rates it category A since `9091cc2`, reading it as
  Request Object encryption; that reading does not match the source). Resolved
  14/10/2026: the harness cannot deliver the premise (Capture's PID is plain and
  readable), so the test asserts `fcaf.skipped` with that reason.
- [x] `WS_RP_IA_MainInteraction__060` (upstream category F: its source asks for
  the Authorization Response appended to `redirect_uri` under
  `response_mode=direct_post`, which contradicts OpenID4VP Section 8.2. Report
  upstream rather than implement). Resolved 14/10/2026: the source relevancy is
  `EUDI_forbidden` (a Wallet must not support `redirect_uri` in the initial
  Authorization Request), so the acceptance clause is `fcaf.not_applicable`.
- [ ] `WS_RP_UC_Presentation__004` (an independently controllable second-device
  invocation path)
- [ ] `WS_RP_IA_Supportive__006` (deliberate Wallet device resource exhaustion)
- [ ] `WS_RP_IA_MainInteraction__046` (a credential large enough to exceed URL
  or transport limits; a missing issuer fixture)

### Blocked on scope and transaction-data support

- [x] `WS_RP_MS_ProtocolMessages__020`, `141` and `WS_RP_UC_Presentation__003`
  (a scope the Wallet resolves to a DCQL query; the service defines no scope
  values). Resolved 14/10/2026 for `020`: the source relevancy is
  `EUDI_forbidden` (scope-borne DCQL), so the acceptance clause is
  `fcaf.not_applicable`; `003` was already `not_applicable`.
- [x] `WS_RP_MS_ProtocolMessages__018`, `135`, `154`–`159` (a transaction-data
  type the reference Wallet supports; `017` was reclassified above and
  `WS_RP_SH_Cryptography_CryptographicHash_008` is implemented). Resolved
  14/10/2026 for `018`: the reference Wallet declares no supported
  transaction-data type, so the premise is factually false for this SUT and the
  test asserts `fcaf.not_applicable`.
