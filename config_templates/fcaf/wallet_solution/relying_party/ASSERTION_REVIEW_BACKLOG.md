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
`WS_RP_*` files and Credimi has 584 matching test definitions. The remaining
source-test difference
difference is listed below; it is deliberately separate from assertion work
that is incomplete or incorrect.

The lists below are the active review state; test-definition totals are kept in
the baseline above and validated by the FCAF catalog loader.

## Blocked

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
- [ ] `WS_RP_IA_MainInteraction__012c_UF`,
  `WS_RP_IA_MainInteraction__012d_UF`, and
  `WS_RP_IA_MainInteraction__034a` through `034i_UF` (the public beta
  service has no documented, deterministic `default_credential_A` fixture
  with the required absent claims and selectively disclosable array/object
  members; generic PID presentations cannot prove these exact selections or
  non-selections)
- [ ] `WS_RP_MS_CredentialFormats__029a` through `029g` and
  `WS_RP_MS_CredentialFormats__033a` through `033h` (beta accepts
  `status_list_enabled: true`, but does not expose an issued status-bearing
  SD-JWT or mdoc MSO as a reusable Capture artifact. A test would require an
  unverified Wallet issuance-and-presentation fixture rather than direct
  protocol evidence)

- [ ] `WS_RP_SM_SessionEncryption__001f` (requires the decrypted
  Authorization Response as the exact unsigned JWT; beta exposes the encrypted
  JWE and verification outcome but not that plaintext JWT artifact)

- [ ] `WS_RP_IA_ProtocolFlow__002d` (requires a reference Wallet profile that
  does not support `request_uri_method=post`; beta has only the post-supporting
  reference Wallet)
- [ ] `WS_RP_IA_ProtocolFlow__003a` (Digital Credentials API flow; beta
  Capture exposes only redirect-based OpenID4VP delivery)
- [ ] `WS_RP_IA_ProtocolFlow__003b_UF` (Digital Credentials API flow; beta
  Capture exposes only redirect-based OpenID4VP delivery)
- [ ] `WS_RP_MS_ProtocolMessages__003_UF` (requires a signed Request Object
  whose JOSE `typ` is controllably invalid; beta only produces its fixed signed
  request type)
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

- [ ] `WS_RP_SH_Encoding_TextualEncoding_008` and
  `WS_RP_SH_Encoding_TextualEncoding_011` (both are implemented against
  `urn:credimi:degree:1`, which carries the source-required heterogeneous
  `degrees` and `academic_programmes` arrays; the credential's disclosure
  frame is top-level only, so a conforming Wallet discloses each array whole
  and per-element removal stays unobservable. Unblocking requires
  `credimi-capture-wallet` to publish per-element disclosure frames for those
  two claims; the Credimi definitions then pass unchanged)
- [ ] `WS_RP_SH_Encoding_TextualEncoding_017` and
  `WS_RP_SH_Encoding_TextualEncoding_021` (the available Capture PID mdoc
  fixture uses `eu.europa.ec.eudi.pid.1`; it cannot produce the source-required
  positive `org.iso.18013.5.1.first_name` element, so an absent-namespace
  request would test rejection rather than the required successful selection)
- [ ] `WS_RP_SM_IssuerIntegrity__014` (Capture exposes the presented SD-JWT
  `x5c` header but no independently published trust-anchor certificate or
  fingerprint; chain position or self-signing cannot prove that the trust
  anchor was excluded)

- [ ] `WS_RP_IA_MainInteraction__024` (requires an encrypted request that remains deliverable and user-confirmable; Capture does not publish encrypted Request Object delivery)
- [ ] `WS_RP_IA_MainInteraction__040` (requires two same-type credentials with distinct values; no such issuer fixture is published)
- [ ] `WS_RP_IA_MainInteraction__041` (requires two same-type credentials with distinct values; no such issuer fixture is published)
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
- [ ] `WS_RP_SM_RpIntegrity__002` (Digital Credentials API request; outside the selected FCAF scope)
- [ ] `WS_RP_SM_RpIntegrity__003` (Digital Credentials API request; outside the selected FCAF scope)
- [ ] `WS_RP_SM_RpIntegrity__004` (Digital Credentials API request; outside the selected FCAF scope)
- [ ] `WS_RP_SM_RpIntegrity__005` (Digital Credentials API request; outside the selected FCAF scope)
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
- [ ] `WS_RP_SM_RpIntegrity__022` (Digital Credentials API request; outside the selected FCAF scope)
- [ ] `WS_RP_SM_RpIntegrity__024` (requires a Wallet profile that rejects every non-`x509_hash` client identifier, contrary to the supported DID and SAN schemes)
- [ ] `WS_RP_SM_RpIntegrity__025` (requires injecting a trust-anchor certificate into `x5c`)
- [ ] `WS_RP_SM_RpIntegrity__026` (requires a self-signed request certificate)
- [ ] `WS_RP_SM_RpIntegrity__027` (requires an invalid Request Object signature)
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

- [ ] `WS_RP_MS_ProtocolMessages__006` (needs a retrievable Request Object signed without a `typ` JOSE header)
- [ ] `WS_RP_MS_ProtocolMessages__007` (needs a retrievable Request Object signed with an invalid `typ` JOSE header)
- [ ] `WS_RP_MS_ProtocolMessages__009` (needs a signed Request Object with independently controlled `client_id` and `iss` claims)
- [ ] `WS_RP_MS_ProtocolMessages__010` (needs a signed Request Object with the required `client_id` claim omitted)
- [ ] `WS_RP_MS_ProtocolMessages__011` (needs a wallet configuration that does not support POST `request_uri` retrieval)
- [ ] `WS_RP_MS_ProtocolMessages__013` (needs two available credentials that satisfy one DCQL credential query)
- [ ] `WS_RP_MS_ProtocolMessages__016` (needs a signed or referenced Authorization Request with a controllable unknown top-level parameter)
- [ ] `WS_RP_MS_ProtocolMessages__017` (needs a wallet configuration without `transaction_data` support and a verifier callback capture)
- [ ] `WS_RP_MS_ProtocolMessages__018` (needs a verifier-supported valid `transaction_data` type and matching wallet capability)
- [ ] `WS_RP_MS_ProtocolMessages__020` (needs a verifier-supported scope value with a defined DCQL mapping)
- [ ] `WS_RP_MS_ProtocolMessages__030` (needs a scope-only request with a controlled unknown scope value)
- [ ] `WS_RP_MS_ProtocolMessages__033` (needs a signed or referenced request with a controllable invalid state value)
- [ ] `WS_RP_MS_ProtocolMessages__034` (needs a signed request with `require_cryptographic_holder_binding=false` and no state)
- [ ] `WS_RP_MS_ProtocolMessages__039` (needs an unsigned request with a malformed or non-HTTPS `redirect_uri:` client identifier)
- [ ] `WS_RP_MS_ProtocolMessages__040` (needs a `redirect_uri:` client identifier request without `redirect_uri`)
- [ ] `WS_RP_MS_ProtocolMessages__041` (needs a direct_post.jwt request with `redirect_uri:` client identifier and no `response_uri`)
- [ ] `WS_RP_MS_ProtocolMessages__043` (needs a controllable HTTP request_uri endpoint)
- [ ] `WS_RP_MS_ProtocolMessages__046` (needs a returned signed Request Object with missing or mismatched `wallet_nonce`)
- [ ] `WS_RP_MS_ProtocolMessages__048` (needs a Request URI response with a controllable wrong Content-Type)
- [ ] `WS_RP_MS_ProtocolMessages__049` (needs independently controllable conflicting outer and Request Object parameters)
- [ ] `WS_RP_MS_ProtocolMessages__051` (client_id in deeplink and request object must differ)

### Issuer status-list control

`status_list_enabled` now allocates a Token Status List reference, but the
published contract does not expose control over credential format, status-list
claim omission, index encoding, URI encoding, or the CBOR map shape required by
the following source tests.

- [ ] `WS_RP_MS_Metadata__081` (needs an issuer-controlled JOSE referenced status token)
- [ ] `WS_RP_MS_Metadata__082` (needs a JOSE referenced token with the status claim omitted)
- [ ] `WS_RP_MS_Metadata__083` (needs a JOSE referenced token with controllable status_list values)
- [ ] `WS_RP_MS_Metadata__084` (needs a JOSE referenced token without status_list)
- [ ] `WS_RP_MS_Metadata__085` (needs a JOSE referenced token with controllable status_list.idx)
- [ ] `WS_RP_MS_Metadata__086` (needs a JOSE referenced token with negative status_list.idx)
- [ ] `WS_RP_MS_Metadata__087` (needs a JOSE referenced token with status_list.idx omitted)
- [ ] `WS_RP_MS_Metadata__088` (needs a JOSE referenced token with a controllable status_list.uri)
- [ ] `WS_RP_MS_Metadata__089` (needs malformed status_list.uri variants in an issued JOSE token)
- [ ] `WS_RP_MS_Metadata__090` (needs a JOSE referenced token with status_list.uri omitted)
- [ ] `WS_RP_MS_Metadata__091` (needs an issuer-controlled COSE referenced status token)
- [ ] `WS_RP_MS_Metadata__092` (needs a COSE referenced token with an empty status map)
- [ ] `WS_RP_MS_Metadata__093` (needs a COSE referenced token with controllable status_list fields)
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
- [ ] `WS_RP_MS_Metadata__139` (needs a verifier to return a mismatched wallet_nonce)
- [ ] `WS_RP_MS_Metadata__140` (needs a verifier to omit wallet_nonce after the Wallet posts one)

- [ ] `WS_RP_IA_Engagement__002`        (W3C API)
- [ ] `WS_RP_IA_MainInteraction__006`   (credential with no hb)
- [ ] `WS_RP_IA_MainInteraction__008`   (credential with no hb)
- [ ] `WS_RP_IA_MainInteraction__010`   (credential with no hb)
- [ ] `WS_RP_IA_MainInteraction__032`   (PID with `over 18` set to false)
- [ ] `WS_RP_IA_MainInteraction__033`   (multiple credentials (same type) with different values)
- [ ] `WS_RP_IA_MainInteraction__053`   (response_uri must be missing)
- [ ] `WS_RP_IA_MainInteraction__055`   (redirect_uri must be present with response_mode=direct_post.jwt)
- [ ] `WS_RP_IA_MainInteraction__056`   (response_uri must be wrong)
- [ ] `WS_RP_IA_MainInteraction__060`   (IMPOSSIBLE, credo does not support fragment/query)
- [ ] `WS_RP_IA_MainInteraction__064`
- [ ] `WS_RP_IA_MainInteraction__065`
- [ ] `WS_RP_IA_MainInteraction__066`
- [ ] `WS_RP_IA_Metadata__010`          (redirect_uri must be present with response_mode=direct_post.jwt)
- [ ] `WS_RP_IA_Metadata__011`          (dynamic discovery)
- [ ] `WS_RP_IA_Metadata__014`          (support openid_federation prefix for client_id)
- [ ] `WS_RP_IA_Supportive__002`        (wrong request_uri)
- [ ] `WS_RP_MS_CredentialFormats__029` (issue jwt with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__030` (issue sd-jwt with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__031` (issue sd-jwt vc with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__032` (issue cwt with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__033` (issue iso mdoc with status.status_list)
- [ ] `WS_RP_MS_CredentialFormats__041` (multiple mdoc credentials with different values)
- [ ] `WS_RP_MS_CredentialFormats__044` (requires credential with status)
- [ ] `WS_RP_MS_CredentialFormats__046` (requires credential with no key-binding)
- [ ] `WS_RP_MS_CredentialFormats__048` (strange json encoding vc, not to be done)
- [ ] `WS_RP_MS_Metadata__105`
- [ ] `WS_RP_MS_Metadata__106`
- [ ] `WS_RP_MS_Metadata__107`
- [ ] `WS_RP_MS_Metadata__109`
- [ ] `WS_RP_MS_ProtocolMessages__002`
- [ ] `WS_RP_MS_ProtocolMessages__124` (verifier needs to add unknown param to A.R. when response_mode=direct_post.jwt)
- [ ] `WS_RP_MS_ProtocolMessages__125` (verifier response_uri return 200 + plain text body)
- [ ] `WS_RP_MS_ProtocolMessages__126` (verifier response_uri return 400 + json body)
- [ ] `WS_RP_MS_ProtocolMessages__127` (verifier needs to add unknown param to response after the wallet POST to response_uri)
- [ ] `WS_RP_MS_ProtocolMessages__128` (verifier needs to add unknown param to A.R. and response after the wallet POST to response_uri)
- [ ] `WS_RP_MS_ProtocolMessages__132` (verifier needs to capture compact jwe after decryption)
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
- [ ] `WS_RP_SH_Encoding_TextualEncoding_002` (requires a credential with a param that is an array of objects with length > 1)
- [ ] `WS_RP_SH_Encoding_TextualEncoding_003` (requires a credential with a param that is an array of elements with length > 1)
