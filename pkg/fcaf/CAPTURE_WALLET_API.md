<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Capture Wallet API capability reference

This Credimi reference tracks the published Capture Wallet API and records the
additional deployment facts required to decide whether an FCAF test is
implementable. The upstream API and the applicable OpenID specifications remain
the wire-contract authorities.
Synced from the upstream [Capture Wallet API reference](https://github.com/ForkbombEu/credimi-capture-wallet/blob/master/CAPTURE_WALLET_API.md)
at `5b03750` on 25/09/2026.

## Published service contract

The Capture Wallet service is a stateful [OpenID4VCI 1.0](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html) credential issuer and [OpenID4VP 1.0](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html) verifier. It issues deterministic PID, degree, and numeric test credentials and captures wallet protocol evidence per session.

This is a companion to the machine-readable [OpenAPI document](/openapi.json). The OpenAPI document and the applicable OpenID specifications are authoritative for wire-level details. Use the issuer metadata rather than hard-coding credential configuration IDs, endpoints, or keys.

## Base URL and conventions

The base URL is configured by `--issuer-base-url`; production uses `https://capture-wallet.credimi.io`. Paths below are relative to that URL.

- Issuer configuration IDs are `eu-pid-device-bound` and `eu-pid-jwt-proof-only`.
- `sessionId` is a UUID returned by a session-creation response.
- JSON is the default representation unless a route states another media type.
- Unknown session IDs return `404` with an error object. Invalid protocol input normally returns `400`.
- Session and event responses are evidence records: their `observed`, `checks`, `raw`, and event `detail` members can grow as the service captures more protocol information. Do not rely on undocumented members.
- Never store access tokens, DPoP proofs, credential offers containing pre-authorized codes, or raw presentation payloads in logs or fixtures.

## Service and discovery

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Readiness response: `{ "status": "ok" }`. |
| `GET` | `/openapi.json` | OpenAPI 3.1 contract for the public REST and protocol surface. |
| `GET` | `/docs` | Interactive API documentation. |
| `GET` | `/issuers` | Lists the always-on issuer configurations, metadata URLs, warnings, and credential configuration IDs. |
| `GET` | `/oid4vci/requests` | Bounded chronological OpenID4VCI request ledger. Sensitive fields are redacted to presence/length metadata. |

For each `{issuerConfigurationId}`:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/.well-known/openid-credential-issuer/issuers/{issuerConfigurationId}` | OpenID4VCI credential issuer metadata. Send `Accept: application/jwt` for signed metadata; JSON is the default. |
| `GET` | `/.well-known/oauth-authorization-server/issuers/{issuerConfigurationId}` | OAuth authorization-server metadata for the issuer. |
| `GET` | `/.well-known/jwt-vc-issuer/issuers/{issuerConfigurationId}` | JWT VC issuer metadata. |
| `GET` | `/issuers/{issuerConfigurationId}/jwks.json` | Authorization-server signing JWKS. |
| `GET` | `/issuers/{issuerConfigurationId}/credential-jwks.json` | Credential-signing JWKS. |

## OpenID4VCI issuance capture

### Create and inspect a session

`POST /sessions` creates an issuance session. Its optional JSON body is:

| Field | Values | Default |
| --- | --- | --- |
| `issuer_configuration_id` | `eu-pid-device-bound`, `eu-pid-jwt-proof-only` | `eu-pid-device-bound` |
| `flow` | `pre_authorized_code`, `authorization_code` | `authorization_code` |
| `credential_offer_mode` | `credential_offer`, `credential_offer_uri` | `credential_offer` |
| `credential_configuration_id` | A configuration advertised by the selected issuer metadata | First configuration for the issuer |
| `status_list_enabled` | Boolean; allocate and embed a Token Status List reference in each issued credential | `false` |
| `fixture_id` | Predefined PID claim set the issued credential carries | `pid_default` |
| `status_reference` | Shape of the `status` structure in an issued SD-JWT VC or mdoc; anything but `valid` is test-only | `valid` |
| `digest_algorithm` | Hash function the issued SD-JWT VC digests its disclosures with: `sha-256`, `sha-384`, `sha-512` | `sha-256` |

It returns `201` with `session_id`, issuer and authorization-server identifiers, the selected flow and configuration, `fixture_id`, `offer_url`, `deeplink`, and `status: "created"`. A configuration belonging to another issuer is rejected.

`fixture_id` names a predefined PID claim set; an unknown value is rejected with `unsupported_fixture_id` and the supported list. Every fixture is a fully valid, normally signed PID and differs from the baseline along one value axis, which is what the DCQL value-matching tests need: `pid_default`, `pid_person_b` (a second complete identity), `pid_under_18` (`age_over_18: false`), `pid_family_name_uppercase`, `pid_family_name_trailing_space`, `pid_locality_diacritics`, `pid_locality_no_diacritics`, `pid_multiple_nationalities`, and `pid_expiry_2032`. Each carries a distinct `document_number`. There is no caller-supplied claim override, so an issued credential always corresponds to a named fixture, and the selected one is recorded as `fixture_id` in the issuance capture.

`status_reference` shapes the `status` structure of an issued SD-JWT VC or mdoc. `valid` embeds
the allocated Token Status List reference and is the default. The other values are the deliberately
malformed structures the revocation-metadata tests require, and are refused with
`status_reference_not_enabled` unless the deployment sets `FCAF_SCENARIOS_ENABLED`:

| `status_reference` | Issued status structure |
| --- | --- |
| `valid` | `{"status_list":{"uri":"…","idx":42}}` or the equivalent valid COSE status |
| `status_without_status_list` | `{}` — the status structure is present without a `status_list` member |
| `negative_index` | `idx` is `-1` |
| `missing_index` | `status_list` carries only `uri` |
| `malformed_uri` | `uri` is not a parseable URI |
| `missing_uri` | `status_list` carries only `idx` |

A malformed fixture requires `status_list_enabled: true`, otherwise it is refused with
`status_reference_requires_status_list`: the reference is always allocated from the configured
Status List service first, and the fixture only reshapes that real allocation. For mdoc
credentials, the service reconstructs and signs only the Mobile Security Object status structure;
the issuer key remains in Credo's KMS. The selected value is recorded as `status_reference` in the
issuance capture.

An issued credential with no `status` claim at all is `status_list_enabled: false`, which is the default and needs no fixture.

`digest_algorithm` sets the `_sd_alg` of an issued SD-JWT VC. `sha-256` is the Section 4.1.1 default and is omitted from the response and the capture; `sha-384` and `sha-512` are recorded as `digest_algorithm` in both. Credential Issuer Metadata keeps advertising SHA-256 only, so a credential digested with anything else is one a Wallet can present only if it supports that hash function, which is what `WS_RP_SH_Cryptography_CryptographicHash_010` asks for. An unsupported value is refused with `unsupported_digest_algorithm` and the supported list; `sha-1` is not offered, because SD-JWT requires a hash function that is secure at issuance. An mdoc configuration is refused with `digest_algorithm_unsupported_for_mdoc`: those digests belong to the Mobile Security Object, not to `_sd_alg`.

[FCAF_FIXTURES.md](FCAF_FIXTURES.md) catalogues every credential, claim-set fixture, and status-list fixture together with the conformance tests each serves.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/sessions/{sessionId}` | Full current issuance capture, including status, observed information, checks, and events. |
| `GET` | `/sessions/{sessionId}/offer` | Credential Offer object. Returns `409` while no offer is available. |
| `GET` | `/sessions/{sessionId}/deeplink` | `{ deeplink, credential_offer }`; records a deeplink-generation event. |
| `GET` | `/sessions/{sessionId}/jwks` | Verified wallet holder-binding JWKS. Returns `409` until a holder key has been observed. |
| `GET` | `/sessions/{sessionId}/events` | Chronological issuance events (`at`, `type`, `detail`). |

### Issuer protocol endpoints

| Method | Path | Required input | Result |
| --- | --- | --- | --- |
| `GET` | `/issuers/{issuerConfigurationId}/offers/{credentialOfferId}` | Credential-offer ID from an offer-by-reference deeplink | Credential Offer object. |
| `POST` | `/issuers/{issuerConfigurationId}/par` | DPoP header; form `response_type=code`, `client_id`, `redirect_uri`, `scope`, `code_challenge`, `code_challenge_method=S256`; optional `issuer_state`, `state` | `201` with `request_uri` and `expires_in`. |
| `GET` | `/issuers/{issuerConfigurationId}/authorize` | Query `client_id` and `request_uri` | Starts the auto-approved authorization-code flow with redirects. |
| `GET` | `/issuers/{issuerConfigurationId}/redirect` | Chained OAuth callback parameters | Redirects back to the wallet with the issuer authorization code. |
| `POST` | `/issuers/{issuerConfigurationId}/token` | DPoP header plus either pre-authorized-code or authorization-code form grant | DPoP-bound access token, expiry, credential nonce, and nonce expiry. |
| `POST` | `/issuers/{issuerConfigurationId}/nonce` | No body | Fresh `c_nonce`. |
| `POST` | `/issuers/{issuerConfigurationId}/credential` | `Authorization: DPoP …`, DPoP header, and a Credential Request | Credential response containing `credentials[].credential`. |

The token endpoint accepts either:

- `grant_type=urn:ietf:params:oauth:grant-type:pre-authorized_code` and `pre-authorized_code`, with optional `tx_code`; or
- `grant_type=authorization_code`, `code`, `code_verifier`, and `redirect_uri`, with optional `client_id`.

The credential request normally uses `application/json` with `credential_configuration_id` and one `proofs.jwt` or `proofs.attestation` entry. For encrypted credential requests, send an `application/jwt` compact JWE; the encrypted response is also a compact JWE. The selected issuer's metadata determines supported proof and encryption capabilities.

## OpenID4VP presentation capture

### Create and inspect a session

`POST /openid4vp/sessions` creates a verifier session. Its optional JSON body accepts:

| Field | Values / shape | Default |
| --- | --- | --- |
| `scheme` | URL-scheme prefix, such as `openid4vp://` | `openid4vp://` |
| `request_uri_method` | Any string; OpenID4VP defines case-sensitive `get`, `post` | `get` |
| `client_id_scheme` | `x509_hash`, `x509_san_dns`, `decentralized_identifier`, `verifier_attestation`, `redirect_uri` | `x509_hash` |
| `verifier_attestation` | Object; requires `client_id_scheme: "verifier_attestation"` | Attestation with the real subject, the fixture issuer, and no `redirect_uris` |
| `request_delivery` | `by_reference`, `by_value`, `plain` | `by_reference` |
| `response_type` | `vp_token`, `vp_token id_token`, `code` | `vp_token` |
| `response_mode` | `direct_post`, `direct_post.jwt`, `dc_api`, `dc_api.jwt` | `direct_post.jwt` |
| `presentation_request` | Request-object claim overrides | — |
| `dcql_query` | DCQL query object, or `null` to omit the parameter from the wallet-facing request | Default query |
| `scopes` | A string or string array | — |
| `transaction_data` | Array of transaction data entries, or any other JSON value | — |
| `verifier_info` | JSON value | — |
| `client_metadata` | Object whose members override generated verifier metadata, or `null` to omit the parameter; top-level only | Generated verifier metadata |
| `redirect_uri` | Absolute URI for the Wallet to open after a successful presentation; use `{{base_url}}/openid4vp/redirect` for a capture redirect page | — |
| `allow_undecryptable_response` | `true` to publish a `client_metadata` object that omits the verifier encryption key | `false` |
| `request_mutation` | Deliberate JSON Pointer edits to the wallet-facing request; test-only | — |
| `request_behavior` | Request-delivery behaviour that is not a payload value; test-only | — |
| `response_scenario` | HTTP response the verifier returns after a presentation; test-only | — |

`request_uri_method` is valid only with `request_delivery: "by_reference"`. The service preserves any supplied string in the deeplink, including values other than the OpenID4VP-defined, case-sensitive `get` and `post`, exclusively to create malformed requests for wallet negative tests. `by_value` supplies a signed Request Object in `request`; `plain` supplies the Authorization Request's URL-encoded parameters directly in the deeplink and omits `request`, `request_uri`, and `request_uri_method`. `response_type`, top-level DCQL, scopes, transaction data, and verifier information are used to construct the wallet-facing request. Inspect the returned `authorization_request` to confirm the exact claims.

`dcql_query: null` omits the query from the request the Wallet receives, which is how a Section 5.1 scope-based request is sent: combine it with `scopes`. The Verifier keeps a query regardless, because the Authorization Response is matched against the request object this service signs, so a presentation returned for a scope-only request still verifies. The kept query appears in `authorization_request` and the delivered request in `raw.authorization_request_delivered`. Scope values are caller-supplied and resolved by the Wallet's profile; this service defines none.

`verifier_info` is delivered exactly as supplied. OpenID4VP Section 5.11 leaves the format and semantics of these attestations to ecosystems and profiles, so this service signs none of its own. To bind a key-bound attestation to the request, read the Client Identifier from `/openid4vp/client-identifiers`, choose the `nonce`, sign the attestation and its proof of possession over that `nonce` and Client Identifier in the structure the profile defines, then create the session with the same `nonce` and the whole `verifier_info` array. A missing `nonce` or `client_id` in the proof, a broken signature, and an unrecognised format are all differences in what the caller signs.

Each decoded SD-JWT VC presentation carries `digest_algorithm`, the credential's `_sd_alg` defaulted to `sha-256` when the issuer omitted it, since that value is absent from the disclosed claims but determines the digest algorithm a Wallet needed to support.

A session that sent transaction data records `checks.transaction_data_verified`: `true` when the presentation was accepted, which includes the Section 8.4 binding Credo-TS verifies from the signed request object; `false` when the presentation was rejected with `invalid_transaction_data`; and `null` when no transaction data was sent, or when the presentation failed for an unrelated reason that says nothing about the binding. The binding requires the Wallet to return, in `transaction_data_hashes`, a hash of each base64url-encoded entry that applies to the presented Credential, using an algorithm the entry offered in `transaction_data_hashes_alg`.

Entries of a `transaction_data` array that are JSON objects are base64url-encoded as OpenID4VP Section 5.1 requires, so a caller supplies the entry it wants the Wallet to decode; entries of any other type, strings included, are delivered exactly as supplied, and a `transaction_data` value that is not an array is passed through untouched. Use `request_mutation` on `/transaction_data` to deliver a parameter that bypasses encoding entirely.

`scheme`, `request_uri_method`, `client_id_scheme`, `request_delivery`, `response_mode`, `client_metadata`, and `redirect_uri` are top-level fields only. They select how the service builds, signs, and delivers the request instead of being request-object claims, so nesting any of them inside `presentation_request` has no effect and is not reported as an error. In particular, a `client_metadata` value inside `presentation_request` is discarded and the generated verifier metadata is used. Only `response_type`, `dcql_query`, `nonce`, `scopes`, `transaction_data`, and `verifier_info` are honoured in both positions, and a top-level `response_type` wins over a nested one.

`client_id_scheme: "x509_san_dns"` signs the request with the existing verifier certificate and uses its DNS Subject Alternative Name as the Client Identifier value. `client_id_scheme: "decentralized_identifier"` signs with a separate `did:web` key and publishes its DID Document at `/openid4vp/did.json`. `client_id_scheme: "verifier_attestation"` signs with the request key but publishes no certificate: the request object's `jwt` JOSE header carries a Verifier Attestation JWT whose `sub` is the Client Identifier after the prefix, whose `cnf.jwk` is the request signing key, and whose `iss` is the fixture attestation issuer published at `/openid4vp/verifier-attestation-issuer/jwks.json`. The optional `verifier_attestation` object sets `subject`, `issuer`, `redirect_uris`, extra `claims`, or `signature: "corrupt"`; the last requires `FCAF_SCENARIOS_ENABLED`, and supplying the object under any other client identifier prefix is refused with `verifier_attestation_requires_verifier_attestation_client_id`. `client_id_scheme: "redirect_uri"` creates an unsigned request and therefore requires `request_delivery: "plain"`; signed and by-reference delivery are rejected. The default remains the certificate hash prefix, `x509_hash`.

When `dcql_query` is `null`, the service omits it from the wallet-facing request. Credo retains the normal default query only as internal verification-session state; a wallet response to this deliberately incomplete request may not validate.

If `client_metadata` is absent, the service uses its generated metadata. `null` omits the parameter entirely, which is intentionally limited to `direct_post`.

An object overrides individual members of the generated metadata rather than replacing the whole object. A member you supply wins, a member you omit keeps its generated value, and a member set to `null` is dropped from the wallet-facing request. The merge is one level deep, so supplying `jwks` replaces the entire key set instead of editing individual JWK members. This is how the advertised response encryption is narrowed for wallet tests that require a specific JWE `enc`:

```json
{ "response_mode": "direct_post.jwt",
  "client_metadata": { "encrypted_response_enc_values_supported": ["A128GCM"] } }
```

The generated `jwks` and `vp_formats_supported` survive that request untouched, so the Wallet has exactly one content-encryption choice and the service still decrypts the response.

For `direct_post.jwt`, the merged metadata must still publish the session's generated verifier encryption public key. Keys are compared by RFC 7638 thumbprint, which covers the public key material only, so optional JOSE members such as `alg`, `use`, and `kid` may be altered or omitted. Because that key is minted inside the same `POST /openid4vp/sessions` call that returns it, a caller cannot reproduce it: omit `jwks` to keep it, and expect `invalid_client_metadata` when `jwks` is replaced or set to `null` without the flag below.

`allow_undecryptable_response: true` waives that check and publishes the supplied `jwks` verbatim, including a foreign or static key. It exists only to build requests that no wallet should answer, such as advertising a key the Verifier does not hold, omitting the key entirely, or reusing one key across sessions. The service can then no longer decrypt a `direct_post.jwt` response, and a wallet that answers anyway is captured as a decryption failure. It requires a `client_metadata` object and is otherwise rejected with `allow_undecryptable_response_requires_client_metadata`. Whenever a `direct_post.jwt` request goes out without the verifier encryption key, the session records a `vp_undecryptable_response_allowed` event.

The `201` response includes `session_id`, delivery and response settings, `deeplink`, `authorization_request`, and `status: "created"`. A redirect session also includes `request_uri`, `request_uri_method`, `response_uri`, and `scheme`; a DC API session omits those and includes `dc_api_request` instead.

When `redirect_uri` is supplied, the service appends a fresh 128-bit `response_code` query parameter and returns the resulting URI in the session-creation response. After a successful wallet submission, the response endpoint returns `200`, `Cache-Control: no-store`, and `{ "redirect_uri": "..." }`; the Wallet must redirect the user agent to it. Invalid presentations retain the normal `400` error response. The exact template `{{base_url}}/openid4vp/redirect`, or the equivalent concrete service URI, creates a service-hosted capture page. It displays the received `response_code` for both valid and invalid visits. A valid visit must include the generated `response_code`; it returns a `200` confirmation page and records `redirect_uri_visited_at`, `redirect_uri_visit_count`, a `vp_redirect_uri_visited` event, and redacted request headers in `raw.redirect_uri_visits`.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/openid4vp/did.json` | Verifier `did:web` Document used by `client_id_scheme: "decentralized_identifier"`. |
| `GET` | `/openid4vp/client-identifiers` | Client Identifier presented under each supported prefix, for binding a caller-signed `verifier_info` attestation. |
| `GET` | `/openid4vp/verifier-attestation-issuer/jwks.json` | Public key of the fixture issuer of Verifier Attestation JWTs, for configuring a Wallet to trust it. |
| `GET` | `/openid4vp/sessions/{sessionId}` | Full current presentation capture. |
| `GET` | `/openid4vp/sessions/{sessionId}/deeplink` | `{ deeplink, authorization_request }`; records a deeplink event. |
| `GET` | `/openid4vp/sessions/{sessionId}/events` | Chronological presentation events. |
| `GET` | `/openid4vp/redirect?response_code=...` | Service-hosted capture redirect page created from the `redirect_uri` template. |

### Deliberate request mutations

`request_mutation` produces the malformed, incomplete, or self-contradicting Authorization
Requests that FCAF negative tests require, as data rather than as named scenarios. It is refused
with `request_mutation_not_enabled` unless the deployment sets `FCAF_SCENARIOS_ENABLED=true`, and
with `invalid_request_mutation` when the shape, a target name, or a pointer is not valid.

```json
{
  "request_mutation": {
    "request_object": { "unset": ["/response_uri"], "set": { "/state": null } },
    "request_object_header": { "set": { "/typ": "jwt" } },
    "outer_request": { "set": { "/client_id": "x509_hash:other" } },
    "verification_applies": false
  }
}
```

| Target | Edits |
| --- | --- |
| `request_object` | The signed Request Object payload. |
| `request_object_header` | The Request Object JOSE header. |
| `outer_request` | The deeplink query parameters, or the DC API `data` member. |

Members are addressed by RFC 6901 JSON Pointer; dotted paths are rejected, because protocol object
keys such as the mdoc namespace `eu.europa.ec.eudi.pid.1` and the credential format `dc+sd-jwt`
contain dots and plus signs. `set` writes any JSON value, including `null` and a wrong type;
`unset` removes the member, which `null` does not. Writes precede removals.

The mutation applies to the wallet-facing copy only. `authorization_request` keeps the request the
service generated and remains the basis of presentation verification, including the nonce, state,
and client identifier it expects; the Request Object is signed twice when a mutation is present, so
the copy the verifier reasons about is never the mutated one. Evidence:

| Field | Content |
| --- | --- |
| `authorization_request` | The generated request, unmutated. |
| `request_mutation` | The mutation as supplied. |
| `raw.authorization_request_delivered` | The mutated Request Object payload that was signed and served. |
| `raw.authorization_request_jwt` | The signed Request Object as served to the Wallet. |
| `raw.outer_request_delivered` | The delivered outer parameters, recorded for every session. |
| `vp_request_mutation_applied` event | The pointers touched and the recorded `verification_applies`. |

`verification_applies` is captured as evidence and never enforced; the verifier keeps its normal
checks in all cases.

### Request-delivery behaviours

`request_behavior` selects a variation that is not a payload value, so no pointer edit can express
it. It is gated by the same `FCAF_SCENARIOS_ENABLED` flag, refused with
`request_behavior_not_enabled` when the flag is off and `invalid_request_behavior` when the shape
is not valid, recorded under `request_behavior`, and logged as `vp_request_behavior_applied`.

| Behaviour | Effect |
| --- | --- |
| `{"signature":"corrupt"}` | The delivered Request Object carries a signature that does not verify. The request is signed by the normal path first and the signature value is then invalidated, so the JWS stays well formed and the Wallet rejects it on the signature. Applies to `request_uri` retrieval, a `by_value` deeplink, the DC API `request` member, and a `wallet_nonce` re-sign. Refused with `signature_behavior_requires_a_signed_request` for the `redirect_uri` client identifier prefix. |
| `{"signing_key":"unrelated"}` | The Request Object is signed with a key that is not the one bound to the advertised client identifier; the certificate in `x5c` and the DID document are untouched. Requires a signed request. |
| `{"certificate_chain":"unrelated_self_signed"\|"untrusted_root"\|"incomplete_chain"}` | A generated X.509 chain replaces `x5c`: one self-signed leaf, a leaf plus an untrusted generated root, or a leaf whose issuer is absent. The request is signed by that chain's leaf key and the `x509_hash` Client Identifier is recomputed from the new leaf, so the chain is the only defect. Requires `client_id_scheme: "x509_hash"`, otherwise `certificate_chain_requires_x509_hash_client_id`. Note that the delivered Client Identifier changes, so a presentation that does arrive fails audience verification. |
| `{"wallet_nonce":"echo"\|"mismatch"\|"omit"}` | How the POST Request URI flow answers the supplied `wallet_nonce`: echo it, return a fresh unrelated value, or leave the parameter out. `echo` is the default. The `vp_request_retrieved` event records `wallet_nonce_present`, `wallet_nonce_behavior`, and `wallet_nonce_returned`. |
| `{"request_uri_response":{"status":…,"content_type":"…","body":"…"}}` | Serve the Request URI with a wrong status, media type, or body; each member is optional. The Request Object is still generated and kept in `raw.authorization_request_jwt`, and the response actually delivered is recorded in `raw.request_uri_response_http`. |

### Verifier response scenarios

`response_scenario` controls the HTTP response returned to the Wallet after it submits an
Authorization Response, at both `POST /openid4vp/sessions/{sessionId}/response` and
`POST /openid4vp/response`. It is gated by `FCAF_SCENARIOS_ENABLED`, refused with
`response_scenario_not_enabled` when the flag is off and `invalid_response_scenario` when the shape
is not valid.

| Member | Effect |
| --- | --- |
| `status` | The delivered HTTP status, 100 to 599. |
| `content_type` | The delivered media type, for example `text/plain`. |
| `body` | The exact delivered body, replacing the normal JSON body. |
| `extra_parameters` | Members merged into the normal JSON body, for an unrecognised response parameter or an error member. |

The scenario is applied at the HTTP boundary and nowhere else. The verification result, session
status, and `checks` are recorded before it and are unaffected: a test-selected `400` does not mark
a valid presentation invalid, and a test-selected `200` does not mark an invalid one verified. The
delivered response is recorded in `raw.presentation_response_verifier_http`, the selection in
`response_scenario`, and a `vp_response_scenario_selected` event is logged at session creation.

### Digital Credentials API presentation

`response_mode: "dc_api"` or `"dc_api.jwt"` presents over the W3C Digital Credentials API, as
defined in OpenID4VP 1.0 Appendix A. The response mode alone selects the flow; there is no
separate transport field.

`request_delivery` decides how the request reaches the browser rather than how it reaches a
wallet: `by_value` — the default for DC API — produces a signed Request Object carrying `client_id`
and `expected_origins`, `plain` produces unsigned request parameters with neither, and
`by_reference` is rejected with `request_delivery_unsupported_for_dc_api`.
`client_id_scheme: "redirect_uri"` is rejected with `client_id_scheme_unsupported_for_dc_api`
because a signed DC API request requires a `client_id`, and `request_uri_method` is rejected with
`request_uri_method_unsupported_for_dc_api`. A DC API request carries no `request_uri`,
`request_uri_method`, `response_uri`, `redirect_uri`, `state`, or `aud`.

`dc_api_request` in the `201` response is the browser invocation payload:

```json
{
  "deeplink": "https://capture-wallet.credimi.io/ui/openid4vp/sessions/sess_123/dc_api_presentation",
  "dc_api_request": {
    "protocol": "openid4vp-v1-signed",
    "data": { "request": "eyJhbGciOiJFUzI1NiIsInR5cCI6Im9hdXRoLWF1dGh6LXJlcStqd3QiLC..." }
  }
}
```

Pass it as one entry of `navigator.credentials.get({ digital: { requests: [ ... ] } })`. For
`plain` delivery the protocol is `openid4vp-v1-unsigned` and `data` holds the Authorization
Request parameters themselves.

`deeplink` keeps its field name and string type, but for DC API it is the HTTPS URL of this
service's presentation page rather than a wallet invocation URL. It is built from the configured
public base URL (`PUBLIC_BASE_URL`, defaulting to `issuer_base_url`) and never from a request
`Host` header. The operator UI renders the usual QR code for it; scanning that code opens a
webpage, and the wallet is invoked only from the button press on that page.

| Method | Path | Input | Result |
| --- | --- | --- | --- |
| `GET` | `/ui/openid4vp/sessions/{sessionId}/dc_api_presentation` | — | Unauthenticated presentation page carrying only this session's `dc_api_request`. Requires `GUI_ENABLED`. |
| `POST` | `/openid4vp/sessions/{sessionId}/response` | The wallet's Authorization Response as JSON or form | Same endpoint as the redirect flows, with no `state` required. |
| `POST` | `/openid4vp/sessions/{sessionId}/dc_api_invocation` | `outcome` plus optional `error_name`, `error_message`, `response_returned`, `vp_token_present` | `202` and `{ "status": "dc_api_invocation_reported" }`; records why the invocation produced no Authorization Response. |

`outcome` is one of `api_unavailable`, `rejected`, `no_vp_token`, or `failed`, and is captured
under `dc_api.invocation` together with the reported `DOMException` name and message, whether a
response object came back, whether it contained a `vp_token`, and the browser `Origin`. This is
the evidence a HAIP wallet leaves when it refuses the unencrypted `dc_api` response mode, and it
is deliberately distinct from a verification failure over a real response: `checks` stays
untouched and the session status becomes `dc_api_invocation_reported`. A wallet refusal and an
End-User cancellation are indistinguishable through the browser API, so `rejected` records what
the browser reported without asserting which occurred.

DC API verification is origin-bound: the expected SD-JWT VC Key Binding JWT audience is
`origin:<origin>` and the mdoc session transcript uses the `OpenID4VPDCAPIHandover` over the bare
origin, the request nonce, and — only for `dc_api.jwt` — the thumbprint of the verifier's
response-encryption key. The origin is the session's configured one; a submission whose `Origin`
header disagrees is refused with `403 unexpected_dc_api_origin`.

A DC API presentation window lasts 10 minutes. A submission after the first captured presentation
is refused with `409 vp_session_already_completed`, and one after the window closes with
`400 vp_session_expired`. `GET` and `POST /openid4vp/sessions/{sessionId}/request` return
`404 vp_request_uri_not_available_for_dc_api`, so a DC API session never records a request_uri
retrieval. Redirect sessions keep their existing behaviour, including recording repeated wallet
submissions.

### Wallet-facing request and response endpoints

| Method | Path | Input | Result |
| --- | --- | --- | --- |
| `GET` | `/openid4vp/sessions/{sessionId}/request` | — | Signed request object with media type `application/oauth-authz-req+jwt`; marks the request as retrieved. |
| `POST` | `/openid4vp/sessions/{sessionId}/request` | Form payload; `wallet_nonce` is recognized and other fields are captured | Signed request object. Use only for a session with `request_uri_method: post`. |
| `POST` | `/openid4vp/sessions/{sessionId}/response` | Form-encoded wallet response | Captures and verifies the response for that session. `200` means valid; `400` returns `invalid_presentation` and verification errors. |
| `POST` | `/openid4vp/response` | Form-encoded wallet response with required `state` | Alternative direct-post endpoint; `state` selects the session. |

Raw form payloads are preserved in the presentation capture for fidelity. Treat them as sensitive evidence.

The session record additionally exposes `raw.authorization_request_jwt`, the exact signed Request Object returned to the Wallet, and `raw.request_uri_http`, the Wallet's Request URI retrieval method, headers with sensitive values redacted, and exact POST body when received. `raw.presentation_response_http` contains the received presentation response HTTP `method`, headers with sensitive values redacted, and exact body. `raw.presentation_response_verifier_http` captures the verifier reply's status, redacted headers, and exact body. The HTTP envelopes are retained for valid and invalid responses and are intentionally not shown as dedicated fields in the operator UI.

## Test-only chained OAuth server

The authorization-code issuance flow uses an internal, auto-approving OAuth server between Credo and the issuer. These are test-service endpoints, not a general-purpose identity provider.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/.well-known/oauth-authorization-server/authorization-servers/{issuerConfigurationId}` | Metadata for the fake OAuth server. |
| `GET` | `/authorization-servers/{issuerConfigurationId}/authorize` | Validates Credo's authorization request and immediately redirects with a code. |
| `POST` | `/authorization-servers/{issuerConfigurationId}/token` | Exchanges the chained code. Requires the configured client credentials, PKCE verifier, redirect URI, and client ID. |

## Browser-only routes

These routes support the server-rendered operator UI. They are not a stable programmatic contract; use the API routes above for integrations.

| Method | Path |
| --- | --- |
| `GET` | `/` |
| `GET` | `/ui/help` |
| `POST` | `/ui/sessions` |
| `GET` | `/ui/sessions/{sessionId}` |
| `POST` | `/ui/openid4vp/sessions` |
| `GET` | `/ui/openid4vp/sessions/{sessionId}` |
| `GET` | `/ui/openid4vp/sessions/{sessionId}/dc_api_presentation` |
| `GET` | `/favicon.svg` |
| `GET` | `/assets/style.css` |
| `GET` | `/assets/credimi_logo.svg` |
| `GET` | `/assets/credimi_logo_negative.svg` |
| `GET` | `/assets/credimi_logo-transp.svg` |
| `GET` | `/assets/credimi_logo-transp_white.svg` |

GUI routes can be disabled with the service configuration; disabling them does not disable API or protocol routes.

## Keeping this reference current

When the public contract changes, update this file and `src/openapi.ts` in the same change. Add or update route-level tests for every changed protocol, metadata, or security boundary. For exact schemas, error objects, and media types, consult `/openapi.json` and the relevant OpenID specification.

## Credimi FCAF deployment notes

FCAF scenarios target `https://beta-capture-wallet.credimi.io`; the public
contract describes the service surface, but beta deployment configuration can
differ. In particular, `request_mutation`, `request_behavior`,
`response_scenario`, and malformed `status_reference` fixtures are only usable
when beta enables `FCAF_SCENARIOS_ENABLED=true`. A published input does not by
itself prove that the Wallet received it or that the desired Wallet outcome was
captured.

Classify each FCAF candidate at three boundaries:

1. **Session input:** can the session-creation API express the setup?
2. **Wallet delivery:** does the returned/decrypted Request Object prove the
   exact property reached the Wallet?
3. **Evidence capture:** does the session record prove the required Wallet
   response rather than merely UI state?

If a required property is rejected before delivery, it remains
verifier-blocked. If the Wallet is shown the request but sends no required
response, record a reference-Wallet failure or discontinuation only when the
source test permits it.

### Beta observations retained for FCAF

- On 17/09/2026, `GET /issuers/eu-pid-device-bound/credential-jwks.json`
  published one leaf `x5c` certificate for `CN = Beta Fake Issuer EU PID Device
  Bound`; its issuing CA was absent from `x5c`. Probe the deployed endpoint
  rather than inferring certificate shape from service source.
- On 17/09/2026, beta preserved a narrowed
  `encrypted_response_enc_values_supported: ["A128GCM"]` advertisement while
  retaining generated JWKs. Production returned the full advertised list on
  the same date.
- On 03/09/2026, beta preserved a DCQL credential without `meta`, an explicit
  empty `meta`, and an empty `trusted_authorities` array in its signed Request
  Object. On 07/09/2026 it also preserved null, integer, Boolean, and absent
  claim-path selectors in the named FCAF probes recorded in git history.
- On 18/09/2026, beta exposed
  `raw.presentation_response_decrypted` and `raw.decoded_presentations` after
  capture. They are parsed evidence, not byte-for-byte plaintext; a validator
  requiring JWT serialization details must pair them with the captured compact
  response and JOSE header evidence.

- Resynced from upstream master on 25/09/2026 (`5b03750`). Since the 22/09/2026
  sync (`6b94fa4`) the contract adds the SD-JWT VC `digest_algorithm` issuance
  option and its per-presentation echo, `client_id_scheme:
  "verifier_attestation"` with the optional `verifier_attestation` object,
  `GET /openid4vp/client-identifiers`, `GET
  /openid4vp/verifier-attestation-issuer/jwks.json`, guidance for binding a
  caller-signed `verifier_info` attestation, mdoc support for
  `status_reference`, and a numeric test credential. The earlier sync's
  `request_behavior.signing_key`, `request_behavior.certificate_chain`, Section
  5.1 `transaction_data` object encoding, and the `dcql_query: null` plus
  `scopes` scope-only request are still unprobed on beta; treat them as
  published contract rather than executable evidence. The tests they reclassify
  are listed in
  `config_templates/fcaf/wallet_solution/relying_party/ASSERTION_REVIEW_BACKLOG.md`.
- On 25/09/2026, beta served `GET /openid4vp/client-identifiers` and `GET
  /openid4vp/verifier-attestation-issuer/jwks.json` with `200`, publishing the
  `x509_hash`, `decentralized_identifier`, and `verifier_attestation` Client
  Identifiers and the ES256 attestation issuer key.
- On 25/09/2026, beta accepted `digest_algorithm: "sha-384"` and `"sha-512"`
  and recorded the selection in the issuance capture; `"sha-1"` was refused
  with `unsupported_digest_algorithm` and the supported list. This is
  executable evidence for `WS_RP_SH_Cryptography_CryptographicHash_010`.
- On 25/09/2026, `eu-pid-device-bound` and `eu-pid-jwt-proof-only` each
  advertised a fourth configuration, `urn:credimi:numeric-claims:1.sd-jwt.*`.
- The `certificate_chain` behaviours recompute the `x509_hash` Client
  Identifier from the replaced leaf. A request built with them therefore
  carries a Client Identifier the verifier no longer expects, so a presentation
  that does arrive fails audience verification; bind those cases to rejection
  evidence only.

### Known FCAF limitations

| Area | Finding | Status / source |
| --- | --- | --- |
| Empty `credential_sets[].options` | The reference Android Wallet displayed an error but did not POST `error=invalid_request`; beta captured only request retrieval. | Blocked for the required protocol assertion. `REFERENCE-WALLET-ISSUES.md` RI-WALLET-001 |
| Positive PID verification | Beta received a `vp_token` but rejected it because the PID issuer URI did not match the issuer certificate SAN. | Verifier-blocked acceptance, not a Wallet failure. `REFERENCE-WALLET-ISSUES.md` MOCK-VERIFIER-001 |
| Invalid `request_uri_method` | The contract permits malformed values in the Wallet-facing deeplink; production rejected `DELETE` on 14/09/2026. | Supported for beta-only negative tests; production deployment lag remains. |
| `client_id_scheme: verifier_attestation` | Session creation returned `500 internal_error`: `the domain of the OpenID4VCI issuer does not match a SAN DNS name in the x5c certificate`, on 25/09/2026. `x509_san_dns` fails identically. | Beta-blocked by the deployed certificate, not by the contract. |

## Safe scenario use

- Change source scenarios under
  `config_templates/fcaf/wallet_solution/relying_party/scenarios/`, never the
  generated aggregate pipeline.
- Bind each validator to the exact scenario output that captured its request or
  response; a successful result from another scenario is never fallback
  evidence.
- Do not store live session URLs, raw tokens, DPoP proofs, or credential offers
  in fixtures or test output.
- A redirect-page screenshot is not proof of a callback. Bind a redirect claim
  to the session's recorded visit evidence.
- Record a dated beta probe before treating a newly published or gated field as
  executable FCAF evidence.
