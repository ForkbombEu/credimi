<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Capture Wallet API capability reference

Capture Wallet is a stateful [OpenID4VCI 1.0](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html) credential issuer and [OpenID4VP 1.0](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html) verifier used to capture Wallet protocol evidence per session. Its base URL is deployment-configured (`--issuer-base-url`): production is `https://capture-wallet.credimi.io`, and FCAF scenarios currently target the beta deployment `https://beta-capture-wallet.credimi.io`. This reference answers a narrow FCAF implementation question: can the service create, deliver, and observe the protocol exchange needed by a test?

It is not a substitute for the OpenID4VCI, OpenID4VP, DCQL, or FCAF specifications. A service accepting a session-creation body does not prove that the same property reached the Wallet in its signed request, nor that the service captured the Wallet response.

## Sources and confidence

Published contract source: the upstream [Capture Wallet API reference](https://github.com/ForkbombEu/credimi-capture-wallet/blob/master/CAPTURE_WALLET_API.md), the deployment's [interactive documentation](https://beta-capture-wallet.credimi.io/docs), and the linked OpenAPI document. Entries marked **published** below come from that contract. Entries marked **observed** come from the named local evidence record. Do not treat an unlisted field or behaviour as supported.

| Status | Meaning |
| --- | --- |
| Supported | Published as an input or endpoint. Still inspect the delivered request and captured response for the individual test. |
| Observed | Verified in a dated Credimi evidence record; it may differ across deployments. |
| Unknown | Not established by the published contract or local evidence. It requires a probe before it can justify a test implementation. |
| Blocked | The current service or reference wallet is known not to produce the required evidence. |

### Service conventions

- `sessionId` is a UUID returned by a session-creation response. JSON is the default representation unless a route states another media type.
- An unknown session ID returns `404` with an error object. Invalid protocol input normally returns `400`.
- Session and event responses are evidence records: their `observed`, `checks`, `raw`, and event `detail` members can gain fields as the service captures more protocol information. Never depend on an undocumented member.
- Never store access tokens, DPoP proofs, credential offers containing pre-authorized codes, or raw presentation payloads in logs, fixtures, or test output.

## Decision sequence for an FCAF test

Classify the test at all three boundaries, in order:

1. **Session input:** can `POST /sessions` or `POST /openid4vp/sessions` express the requested setup?
2. **Wallet delivery:** can `GET /openid4vp/sessions/{sessionId}/request` or the session/deeplink evidence prove that the exact resulting Authorization Request contains the required property?
3. **Evidence capture:** can the relevant session record or events prove the Wallet's protocol response, rather than only its UI state?

If a required property is rejected before a signed request is delivered, mark the case verifier-blocked; do not create synthetic Wallet evidence. If the Wallet is shown the request but does not submit the required response, record a reference-wallet failure or discontinuation only when the FCAF source permits it.

## Shared service and metadata

| Endpoint | Capability | Status |
| --- | --- | --- |
| `GET /healthz` | Readiness response `{ "status": "ok" }`. | Supported |
| `GET /openapi.json` | OpenAPI 3.1 contract for the public REST and protocol surface. | Supported |
| `GET /docs` | Interactive API documentation. | Supported |
| `GET /issuers` | Lists the always-on issuer configurations, metadata URLs, warnings, and credential configuration IDs. | Supported |
| `GET /oid4vci/requests` | Bounded chronological OpenID4VCI request ledger. Sensitive values are redacted to presence/length metadata. | Supported |
| `GET /.well-known/openid-credential-issuer/issuers/{issuerConfigurationId}` | OpenID4VCI issuer metadata; send `Accept: application/jwt` for signed metadata. | Supported |
| `GET /.well-known/oauth-authorization-server/issuers/{issuerConfigurationId}` | OAuth authorization-server metadata. | Supported |
| `GET /.well-known/jwt-vc-issuer/issuers/{issuerConfigurationId}` | JWT VC issuer metadata. | Supported |
| `GET /issuers/{issuerConfigurationId}/jwks.json` | Authorization-server signing JSON Web Key Set. | Supported |
| `GET /issuers/{issuerConfigurationId}/credential-jwks.json` | Credential-signing JSON Web Key Set. | Supported |

Use the metadata endpoints rather than hard-coding credential configuration identifiers, authorization-server settings, or issuer keys in a test.

## Issuer: OpenID4VCI capture sessions

### Session surface

| Endpoint | Inputs / output | Status |
| --- | --- | --- |
| `POST /sessions` | Optional JSON: `issuer_configuration_id` (`eu-pid-device-bound`, the default, or `eu-pid-jwt-proof-only`), `flow` (`pre_authorized_code` or `authorization_code`, default `authorization_code`), `credential_offer_mode` (`credential_offer`, the default, or `credential_offer_uri`), a metadata-advertised `credential_configuration_id` (default: the issuer's first configuration), and `status_list_enabled` (boolean, default `false`). Returns `201` with `session_id`, the selected issuer and authorization-server identifiers, the selected flow and configuration, `offer_url`, `deeplink`, and `status: "created"`. A configuration belonging to another issuer is rejected. | Supported |
| `GET /sessions/{sessionId}` | Current issuance capture containing `observed`, `checks`, and `events` in addition to session state. | Supported |
| `GET /sessions/{sessionId}/offer` | Credential Offer object; returns `409` while no offer is available. | Supported |
| `GET /sessions/{sessionId}/deeplink` | `deeplink` and `credential_offer`; records a deeplink-generation event. | Supported |
| `GET /sessions/{sessionId}/jwks` | Verified Wallet holder-binding JWKS observed from the credential-proof header; returns `409` until a holder key has been observed. | Supported |
| `GET /sessions/{sessionId}/events` | Chronological events with timestamp, type, and arbitrary detail. | Supported |

The service has two always-on issuer configurations, `eu-pid-device-bound` and
`eu-pid-jwt-proof-only`. Obtain their credential configuration IDs and protocol
metadata from `GET /issuers` instead of hard-coding them.

| Issuer configuration ID | Reference EUDI Wallet version |
| --- | --- |
| `eu-pid-device-bound` | Version 39 and later |
| `eu-pid-jwt-proof-only` | Version 38 |

`status_list_enabled: true` allocates and embeds a Token Status List reference in each issued
credential. The former `broken` fixture toggle is not part of the published beta
contract.

Both issuer configurations also advertise a non-PID test credential,
`urn:credimi:degree:1` (`urn:credimi:degree:1.sd-jwt.key-attestation-required`
and `urn:credimi:degree:1.sd-jwt.jwt-proof`). Observed on beta 17/09/2026: it
carries `degrees` with two `type`-bearing entries plus one entry holding only
`university`, and `academic_programmes` as
`[["Bachelor of Science"], ["Master of Science", "Doctor of Philosophy"]]`.
Those are the heterogeneous JSON structures the DCQL claims-path tests need.

| Capability | Status |
| --- | --- |
| Issue `urn:credimi:degree:1` with the heterogeneous `degrees` and `academic_programmes` structures. | Observed on beta 17/09/2026 |
| Prove per-element claims-path removal inside those arrays. | Observed 17/09/2026: `degreeSdJwtCredentialSignOptions` signs a nested `DEGREE_DISCLOSURE_FRAME`, so every `degrees` entry is an array-element disclosure with separate `type` and `university` disclosures and every `academic_programmes` string is its own array-element disclosure. A Wallet can therefore reveal `degrees[0..1].type` without any `university`, and `academic_programmes[1][1]` without `academic_programmes[0][0]`. The issuer metadata still advertises the two claims at whole-claim granularity, which is display metadata only. |

### Issuer protocol surface

| Endpoint | Published requirements | Status |
| --- | --- | --- |
| `GET /issuers/{issuerConfigurationId}/offers/{credentialOfferId}` | Retrieves the Credential Offer referenced by an offer-by-reference (`credential_offer_uri`) deeplink. | Supported |
| `POST /issuers/{issuerConfigurationId}/par` | DPoP header and form `response_type=code`, `client_id`, `redirect_uri`, `scope`, `code_challenge`, and `code_challenge_method=S256`; optional `issuer_state` and `state`. Returns `request_uri` and expiry. | Supported |
| `GET /issuers/{issuerConfigurationId}/authorize` | `client_id` and `request_uri`; begins the auto-approved authorization-code flow. | Supported |
| `GET /issuers/{issuerConfigurationId}/redirect` | Chained OAuth callback; redirects to the Wallet with the issuer authorization code. | Supported |
| `POST /issuers/{issuerConfigurationId}/token` | DPoP header plus either a pre-authorized-code or authorization-code form grant; returns a DPoP token and credential nonce. | Supported |
| `POST /issuers/{issuerConfigurationId}/nonce` | No body; returns a fresh `c_nonce`. | Supported |
| `POST /issuers/{issuerConfigurationId}/credential` | DPoP access token, DPoP header, and either an `application/json` Credential Request carrying `credential_configuration_id` and one `proofs.jwt` or `proofs.attestation` entry, or a compact-JWE `application/jwt` request; returns a credential response with `credentials[].credential`, optionally as a compact JWE. | Supported |

For the token endpoint, use either the pre-authorized-code grant (with optional
`tx_code`) or the authorization-code grant (with `code`, `code_verifier`, and
`redirect_uri`). The selected issuer metadata determines supported proof and
credential-response encryption capabilities. The service records issuance
capture evidence, but the exact `observed`, `checks`, and event-detail shapes
are intentionally open-ended in the public schema. Inspect an actual session
before a validator depends on a particular field.

### Test-only chained OAuth server

The authorization-code issuance flow uses an internal, auto-approving OAuth
server between Credo and the issuer. It is test-service infrastructure, not a
general-purpose identity provider.

| Endpoint | Capability | Status |
| --- | --- | --- |
| `GET /.well-known/oauth-authorization-server/authorization-servers/{issuerConfigurationId}` | Fake OAuth server metadata. | Supported |
| `GET /authorization-servers/{issuerConfigurationId}/authorize` | Validates Credo's authorization request and immediately redirects with a code. | Supported |
| `POST /authorization-servers/{issuerConfigurationId}/token` | Exchanges the chained code; requires configured client credentials, PKCE verifier, redirect URI, and client ID. | Supported |

## Verifier: OpenID4VP capture sessions

### Session creation

`POST /openid4vp/sessions` creates a session and returns `201` with `session_id`, delivery settings, `request_uri`, `response_uri`, `deeplink`, `authorization_request`, and `status: "created"`.

| Field | Published values / shape | Status |
| --- | --- | --- |
| `scheme` | URL-scheme prefix matching `scheme://`; defaults to `openid4vp://`. | Supported |
| `request_uri_method` | Any string; OpenID4VP defines the case-sensitive values `get` and `post`; defaults to `get`. | Published, including deliberate malformed values for Wallet negative tests; production lag observed 14/09/2026. |
| `client_id_scheme` | `x509_hash`, `x509_san_dns`, `decentralized_identifier`, or `redirect_uri`; defaults to `x509_hash`. | Supported, subject to delivery constraints. |
| `request_delivery` | `by_reference`, `by_value`, or `plain`; defaults to `by_reference`. | Supported |
| `response_type` | `vp_token`, `vp_token id_token`, or `code`; default `vp_token`. | Supported |
| `response_mode` | `direct_post` or `direct_post.jwt`; default `direct_post.jwt`. | Supported |
| `presentation_request` | Request-object claim overrides. | Supported as input; delivery semantics must be inspected. |
| `dcql_query` | DCQL query object, or `null` to omit the parameter; defaults to the service's default query. | Supported as input; delivery semantics must be inspected. |
| `scopes` | String or string array. | Supported as input. |
| `transaction_data` | Unconstrained JSON value. | Supported as input. Observed on 02/09/2026: when nested in `presentation_request`, it is preserved in the signed Request Object; supported Wallet types remain unknown. |
| `verifier_info` | Unconstrained JSON value. | Supported as input. Observed on 02/09/2026: when nested in `presentation_request`, it is preserved in the signed Request Object; attestation generation and Wallet support remain unknown. |
| `client_metadata` | Object whose members override individual generated verifier metadata members, or `null` to omit the parameter; top-level only. | Supported, subject to response-mode constraints. |
| `allow_undecryptable_response` | `true` publishes the supplied `jwks` verbatim and waives the verifier-encryption-key check; requires a `client_metadata` object. | Supported; observed on beta 17/09/2026. |
| `redirect_uri` | Absolute URI the Wallet opens after a successful presentation; the exact template `{{base_url}}/openid4vp/redirect` selects the service-hosted capture page. | Supported; the service appends a fresh 128-bit `response_code`. |

`request_uri_method` is valid only with `request_delivery: "by_reference"`.
The published contract states that the service preserves any supplied string in
the deeplink, including values other than the OpenID4VP-defined, case-sensitive
`get` and `post`, exclusively to create malformed requests for Wallet negative
tests. A beta probe on 14/09/2026 accepted `DELETE`, created session
`697125c6-1c20-4e98-81b9-c3b9356a857c`, and returned the value in the deeplink.
Production still returned `400 {"error":"unsupported_request_uri_method"}` at
that time.
`by_value` delivers a signed Request Object in `request`; `plain` delivers
URL-encoded Authorization Request parameters in the deeplink and omits
`request`, `request_uri`, and `request_uri_method`. `client_id_scheme:
"redirect_uri"` requires unsigned `plain` delivery. `x509_san_dns` uses the
verifier certificate DNS Subject Alternative Name; `decentralized_identifier`
uses a separate `did:web` key published at `GET /openid4vp/did.json`.

When `dcql_query` is `null`, Capture omits it from the Wallet-facing request;
the service retains its normal query only as internal verification-session
state, so a response may not validate.

`client_metadata` merges one level deep over the generated metadata: a member
you supply wins, a member you omit keeps its generated value, and a member set
to `null` is dropped from the Wallet-facing request. Because the merge is one
level deep, supplying `jwks` replaces the whole key set rather than editing
individual JWK members. Omitting the field entirely uses generated metadata, and
`null` omits the parameter, which is supported only with `direct_post`.

That merge is how the advertised response encryption is narrowed for a test
requiring one specific JWE `enc`, while the generated `jwks` and
`vp_formats_supported` survive untouched and the service still decrypts:

```json
{ "response_mode": "direct_post.jwt",
  "client_metadata": { "encrypted_response_enc_values_supported": ["A128GCM"] } }
```

Observed on beta 17/09/2026: that body returned `201` advertising exactly
`["A128GCM"]` with the generated `jwks` intact. The production deployment still
returned the full `["A128GCM", "A256GCM", "A128CBC-HS256"]` list on the same
date, so a scenario depending on a narrowed advertisement runs on beta only.

For `direct_post.jwt` the merged metadata must still publish the session's
generated verifier encryption public key. Keys are compared by RFC 7638
thumbprint, which covers public key material only, so optional JOSE members such
as `alg`, `use`, and `kid` may be altered or omitted. That key is minted inside
the same `POST /openid4vp/sessions` call that returns it, so a caller cannot
reproduce it: omit `jwks` to keep it, and expect `invalid_client_metadata` when
`jwks` is replaced or nulled without the flag below.

`allow_undecryptable_response: true` waives that check and publishes a supplied
replacement `jwks` verbatim, including a foreign, static, `alg`-less, or
deliberately mismatched key. Set `"jwks": null` to omit `jwks` entirely, or set
`"jwks": {"keys":[]}` to publish an empty key set; both require the flag.
It exists only to build requests no Wallet should answer. The service can then
no longer decrypt a `direct_post.jwt` response, a Wallet that answers anyway is
captured as a decryption failure, and every such request records a
`vp_undecryptable_response_allowed` event. Without a `client_metadata` object
it is rejected with `allow_undecryptable_response_requires_client_metadata`.

Observed on beta 17/09/2026: with the flag, a static P-256 `use: enc` key was
published verbatim both without `alg` and with `alg: ECDH-ES+A256KW`; the same
body without the flag returned `400 {"error":"invalid_client_metadata"}`.
`"jwks": null` omitted the member from the signed Request Object (session
`bca854ad-9d91-4ea4-a53e-065aa6916a34`), while an empty metadata object retained
the generated key. A single static P-256 `use: enc` key with
`alg: ECDH-ES+A256KW` has no JWK whose `alg` is bare `ECDH-ES`, so it constructs
the source precondition for `WS_RP_SM_SessionEncryption__006` and the
static-key-reuse precondition for `WS_RP_SM_SessionEncryption__010`; it does not
prove a Wallet outcome without a Wallet interaction.

A `client_metadata` value nested inside `presentation_request` is discarded
without an error, and the generated metadata is used. `scheme`,
`request_uri_method`, `client_id_scheme`, `request_delivery`, `response_mode`,
`client_metadata`, and `redirect_uri` are top-level fields only; only
`response_type`, `dcql_query`, `nonce`, `scopes`, `transaction_data`, and
`verifier_info` are honoured in both positions, and a top-level `response_type`
wins over a nested one.

Observed on beta 17/09/2026: omitting `nonce` entirely generates a service
nonce, but setting `presentation_request.nonce` to `null` omits the claim from
the signed Request Object (session `302c9402-d95a-427d-b93b-cbde2f01301e`).
A non-empty supplied nonce is preserved verbatim, including `fcaf nonce/!`
(session `9b24ae01-8c79-4283-9cef-d38a93a7ddbf`). These controls make the missing-
and malformed-nonce request preconditions for `WS_RP_SM_SessionBinding__002`
and `WS_RP_SM_SessionBinding__003` constructible.

When a session sets `redirect_uri`, the service appends a fresh 128-bit
`response_code` and returns the resulting URI in the creation response. After a
successful Wallet submission the response endpoint returns `200`,
`Cache-Control: no-store`, and `{ "redirect_uri": "..." }`, which the Wallet must
open; an invalid presentation still returns the normal `400` error. The concrete
service URI `https://beta-capture-wallet.credimi.io/openid4vp/redirect` selects
the service-hosted capture page. The upstream `{{base_url}}/openid4vp/redirect`
template resolves to the same page. Credimi resolves `${fixture.verifier_url}`
to the beta Capture Wallet endpoint by default; scenarios MAY override it only
when a test requires a different verifier. A visit carrying the generated
`response_code` returns a `200` confirmation page and records
`redirect_uri_visited_at`, `redirect_uri_visit_count`, a
`vp_redirect_uri_visited` event whose `detail.visit_count` counts the visit, and
redacted request headers in `raw.redirect_uri_visits`. Those members are the
only protocol-level proof that the Wallet's user agent opened the
verifier-supplied redirect; a screenshot of the page is not equivalent.

Observed on 16/09/2026 against beta: session creation returns
`…/openid4vp/redirect?response_code=<128-bit>` for both the concrete URL and
the `{{base_url}}` template; a `GET` with that code returns
`200 text/html`, `Cache-Control: no-store`, and the `Presentation complete`
page. Contrary to the upstream document, a missing, empty, or unknown
`response_code` returns `404` with the `Redirect page not found` page and
records nothing, so a recorded visit proves the exact generated URI was opened.
Each `GET` increments `redirect_uri_visit_count`, appends a
`raw.redirect_uri_visits` entry, and adds another `vp_redirect_uri_visited`
event, so assert `>= 1` or an exact count deliberately. `raw.redirect_uri_visits`
entries carry only `method` and redacted headers: no request target or query is
captured, so the capture cannot prove that the Wallet refrained from appending
Authorization Response parameters to the redirect URI. A visit is also recorded
for a session that never received a presentation, and any client can create one:
never fetch a session's configured redirect URI from a pipeline step, or the
evidence is fabricated.

Observed on 03/09/2026: a DCQL credential that omits `meta` is accepted at
session creation and preserved without `meta` in the signed Request Object.
An explicitly empty `meta: {}` object is likewise accepted and preserved.
An empty `trusted_authorities: []` array is also accepted and preserved.
On 03/09/2026, a `trusted_authorities` entry with `type: unsupported` and a
string `values` array was accepted and preserved in the signed Request Object.
On 03/09/2026, a `trusted_authorities` entry with a string `values` array but
no `type` was also accepted and preserved in the signed Request Object.
On 07/09/2026, a nested claim path containing a null selector,
`["address", null, "street_address"]`, was accepted and preserved unchanged in
the returned Authorization Request (Capture session `eb00f511-2fc2-4746-baaf-aea6961d3f71`).
On 07/09/2026, the same was observed for the integer-selector path
`["address", "street_address", 0]` (Capture session
`846a712e-19b2-4234-ad85-26bf51b07867`).
On 07/09/2026, Capture also preserved the unsupported Boolean component in
`["address", "street_address", false]` (Capture session
`d3c78b51-3d53-4e03-87f9-84b003e70384`).
On 07/09/2026, Capture preserved the nested missing-member path
`["address", "unavailable_address_member"]` (Capture session
`bb38af67-df85-4654-88b4-56b2699be424`).
On 07/09/2026, Capture accepted and preserved the mdoc path with an absent
namespace, `["org.iso.18013.5.1", "first_name"]`, under
`format: mso_mdoc` and `doctype_value: eu.europa.ec.eudi.pid.1` (Capture
session `13aa1df4-e5b8-432f-b208-5454d71bbea0`).

`additionalProperties` are accepted by the public request schema, but that does **not** establish that an unknown property appears in the signed Authorization Request. Retrieve and decode the request before using an unknown field as test evidence.

### Delivery and response endpoints

| Endpoint | Capability | Status |
| --- | --- | --- |
| `GET /openid4vp/sessions/{sessionId}` | Current presentation capture with `authorization_request`, `observed`, `checks`, `events`, and raw protocol evidence. | Supported |
| `GET /openid4vp/sessions/{sessionId}/deeplink` | Returned deeplink and decoded `authorization_request`; records a deeplink event. | Supported |
| `GET /openid4vp/sessions/{sessionId}/request` | Retrieves the signed request object as `application/oauth-authz-req+jwt` and marks it as retrieved. | Supported |
| `POST /openid4vp/sessions/{sessionId}/request` | Retrieves the signed request when `request_uri_method: post`; accepts form `wallet_nonce` and additional fields. | Supported |
| `POST /openid4vp/sessions/{sessionId}/response` | Captures and verifies a form-encoded Wallet response for that session; `200` means valid, and `400` returns `invalid_presentation` with the verification errors. | Supported |
| `POST /openid4vp/response` | Alternative form-encoded direct-post endpoint; required `state` identifies the session. | Supported |
| `GET /openid4vp/sessions/{sessionId}/events` | Chronological protocol capture events. | Supported |
| `GET /openid4vp/did.json` | Verifier `did:web` Document used by `client_id_scheme: "decentralized_identifier"`. | Supported |
| `GET /openid4vp/redirect?response_code=...` | Service-hosted capture redirect page created from the `redirect_uri` template; records the visit on the session. | Supported |

The session's `raw` object provides the protocol-evidence surface required for
assertions. `raw.authorization_request_jwt` is the exact signed Request Object
returned to the Wallet. `raw.request_uri_http` records the Wallet's Request URI
retrieval method and redacted headers, and adds the exact POST body when one was
received. The capture is attached to the session-specific `/request` endpoint;
the published schema does not expose a separate raw request-target or query
field.

Observed on beta 17/09/2026: the raw record for session
`627e4f90-b955-43c5-bd3e-c03edd1b2487` preserved a POST method, the
`application/x-www-form-urlencoded` and `application/oauth-authz-req+jwt`
headers, and the exact percent-encoded body
`wallet_nonce=%FF&extra=%E2%82%AC`. The capture can therefore distinguish
invalid UTF-8 form values from correctly UTF-8-encoded values; it can support
the evidence requirements for `WS_RP_MS_ProtocolMessages__042` and
`WS_RP_MS_ProtocolMessages__044`.

The direct-post endpoints return `200` only when the presentation was captured
and verified. A failed verifier check and a Wallet's decision to send no
response are distinct outcomes. `raw.presentation_response_http` and
`raw.presentation_response_verifier_http` provide machine-readable,
sensitive-value-redacted HTTP evidence for valid and invalid responses; the
former retains the exact received method, headers, and body, and the latter the
verifier reply's status, headers, and body. These envelopes are intentionally
not shown as dedicated fields in the operator UI, so read them from the session
record. Inspect the session record and events; never substitute a screenshot for
missing callback evidence.

## Browser-only routes

`/`, `/ui/help`, `/ui/sessions`, `/ui/sessions/{sessionId}`,
`/ui/openid4vp/sessions`, `/ui/openid4vp/sessions/{sessionId}`, `/favicon.svg`,
and `/assets/*` render the server-rendered operator UI. They are not a stable
programmatic contract; never bind a scenario or validator to them. A deployment
can disable the GUI routes without disabling the API or protocol routes, so no
test may depend on the UI being reachable.

## Known local limitations

| Area | Finding | Status / source |
| --- | --- | --- |
| Empty `credential_sets[].options` | The reference Android wallet displayed an error but did not POST `error=invalid_request`; the beta session captured only request retrieval. | Blocked for the required protocol assertion. [RI-WALLET-001](REFERENCE-WALLET-ISSUES.md) |
| Positive PID verification | The beta verifier received a `vp_token` but rejected it because the PID issuer URI did not match the issuer certificate SAN. | Verifier-blocked acceptance, not a Wallet failure. [MOCK-VERIFIER-001](REFERENCE-WALLET-ISSUES.md) |
| Invalid `request_uri_method` | The published contract preserves arbitrary values in the Wallet-facing deeplink; the production deployment still rejected `DELETE` at session creation on 14/09/2026. | Supported for `WS_RP_MS_ProtocolMessages__152` on beta; production deployment lag remains. |

`pkg/fcaf/MEMORY.md` additionally lists test-specific cases blocked because the public verifier validates malformed DCQL before it can create a signed request, or because it cannot expose the raw request/response feature required by the test. Treat that as coordination state and re-probe it when the service changes.

## Safe use in scenarios

- Use a source scenario under `config_templates/fcaf/wallet_solution/relying_party/scenarios/`; do not edit the generated aggregate pipeline directly.
- Persist the created `session_id` only as a pipeline output needed to fetch protocol evidence. Do not put live session URLs or tokens in fixtures.
- For verifier tests, bind validators to the exact scenario output containing the capture session/request/response. A different scenario's successful response is not fallback evidence.
- For issuer tests, use the session, events, and observed wallet JWKS to prove the relevant issuance exchange; inspect their actual shape first.
- When a test must prove that the Wallet opened the post-presentation redirect, set `redirect_uri` to `https://beta-capture-wallet.credimi.io/openid4vp/redirect` and bind the validator to `redirect_uri_visit_count`, `redirect_uri_visited_at`, the `vp_redirect_uri_visited` event, or `raw.redirect_uri_visits`. The capture page renders for invalid visits too, so a screenshot of it proves nothing.
- Record a dated probe and update this document when a previously unknown capability becomes a test prerequisite.
