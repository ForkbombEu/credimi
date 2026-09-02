<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Capture Wallet API capability reference

`https://capture-wallet.credimi.io` is a stateful OpenID4VCI issuer and OpenID4VP verifier used to capture Wallet protocol evidence. This reference answers a narrow FCAF implementation question: can the public service create, deliver, and observe the protocol exchange needed by a test?

It is not a substitute for the OpenID4VCI, OpenID4VP, DCQL, or FCAF specifications. A service accepting a session-creation body does not prove that the same property reached the Wallet in its signed request, nor that the service captured the Wallet response.

## Sources and confidence

Published contract source: [Capture Wallet API documentation](https://capture-wallet.credimi.io/docs) and its linked OpenAPI document, retrieved on 02/09/2026. Entries marked **published** below come from that OpenAPI document. Entries marked **observed** come from the named local evidence record. Do not treat an unlisted field or behaviour as supported.

| Status | Meaning |
| --- | --- |
| Supported | Published as an input or endpoint. Still inspect the delivered request and captured response for the individual test. |
| Observed | Verified in a dated Credimi evidence record; it may differ across deployments. |
| Unknown | Not established by the published contract or local evidence. It requires a probe before it can justify a test implementation. |
| Blocked | The current service or reference wallet is known not to produce the required evidence. |

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
| `GET /.well-known/openid-credential-issuer` | OpenID4VCI issuer metadata. | Supported |
| `GET /.well-known/oauth-authorization-server` | OAuth authorization-server metadata. | Supported |
| `GET /.well-known/jwt-vc-issuer` | JWT VC issuer metadata. | Supported |
| `GET /jwks.json` | Issuer JSON Web Key Set. | Supported |

Use the metadata endpoints rather than hard-coding credential configuration identifiers, authorization-server settings, or issuer keys in a test.

## Issuer: OpenID4VCI capture sessions

### Session surface

| Endpoint | Inputs / output | Status |
| --- | --- | --- |
| `POST /sessions` | Optional JSON: `credential_configuration_id` advertised by issuer metadata and `broken` (boolean, default `false`). Returns `201` with `session_id`, selected configuration, `broken`, `offer_url`, `deeplink`, and `status: "created"`. | Supported |
| `GET /sessions/{sessionId}` | Current issuance capture containing `observed`, `checks`, and `events` in addition to session state. | Supported |
| `GET /sessions/{sessionId}/offer` | Credential offer. | Supported |
| `GET /sessions/{sessionId}/deeplink` | `deeplink` and `credential_offer`. | Supported |
| `GET /sessions/{sessionId}/jwks` | Wallet holder-binding JWKS observed from the proof header; returns `409` until a proof header JWK exists. | Supported |
| `GET /sessions/{sessionId}/events` | Chronological events with timestamp, type, and arbitrary detail. | Supported |

`broken: true` requests an intentionally malformed legacy PID fixture instead of the conforming fixture. This is the only published fixture toggle. Which malformation it produces, and whether other credential fixture variants are available, are **unknown** until verified from the session evidence.

### Issuer protocol surface

| Endpoint | Published requirements | Status |
| --- | --- | --- |
| `POST /par` | Form fields `issuer_state`, `client_id`, `redirect_uri`, `scope`, `code_challenge`, and `code_challenge_method=S256`; returns `request_uri` and expiry. | Supported |
| `GET /authorize` | `request_uri`; resolves a valid PAR request and redirects with code, state, and `iss`. | Supported |
| `POST /token` | DPoP header plus form `code` and `code_verifier`; optional client-attestation fields; returns a DPoP token and credential nonce. | Supported |
| `POST /nonce` | Returns `c_nonce` and its expiry. | Supported |
| `POST /credential` | DPoP access token, DPoP header, and JSON `credential_configuration_id` plus `proofs.jwt`; returns `credentials[].credential`. | Supported |

The service records issuance capture evidence, but the exact `observed`, `checks`, and event-detail shapes are intentionally open-ended in the public schema. Inspect an actual session before a validator depends on a particular field.

## Verifier: OpenID4VP capture sessions

### Session creation

`POST /openid4vp/sessions` creates a session and returns `201` with `session_id`, delivery settings, `request_uri`, `response_uri`, `deeplink`, `authorization_request`, and `status: "created"`.

| Field | Published values / shape | Status |
| --- | --- | --- |
| `scheme` | URL-scheme prefix matching `scheme://`; defaults to `openid4vp://`. | Supported |
| `request_uri_method` | `get` or `post`; defaults to `get`. | Supported |
| `request_delivery` | `by_reference` or `by_value`; defaults to `by_reference`. | Supported |
| `response_type` | `vp_token`, `vp_token id_token`, or `code`; default `vp_token`. | Supported |
| `response_mode` | `direct_post` or `direct_post.jwt`; default `direct_post.jwt`. | Supported |
| `presentation_request` | Open-ended JSON object. | Supported as input; delivery semantics must be inspected. |
| `dcql_query` | Open-ended JSON object. | Supported as input; delivery semantics must be inspected. |
| `scopes` | String or string array. | Supported as input. |
| `transaction_data` | Unconstrained JSON value. | Supported as input. Observed on 02/09/2026: when nested in `presentation_request`, it is preserved in the signed Request Object; supported Wallet types remain unknown. |
| `verifier_info` | Unconstrained JSON value. | Supported as input. Observed on 02/09/2026: when nested in `presentation_request`, it is preserved in the signed Request Object; attestation generation and Wallet support remain unknown. |

`additionalProperties` are accepted by the public request schema, but that does **not** establish that an unknown property appears in the signed Authorization Request. Retrieve and decode the request before using an unknown field as test evidence.

### Delivery and response endpoints

| Endpoint | Capability | Status |
| --- | --- | --- |
| `GET /openid4vp/sessions/{sessionId}` | Current presentation capture with `authorization_request`, `observed`, `checks`, and `events`. | Supported |
| `GET /openid4vp/sessions/{sessionId}/deeplink` | Returned deeplink and decoded `authorization_request`. | Supported |
| `GET /openid4vp/sessions/{sessionId}/request` | Retrieves the signed request object as `application/oauth-authz-req+jwt`. | Supported |
| `POST /openid4vp/sessions/{sessionId}/request` | Retrieves the signed request when `request_uri_method: post`; accepts form `wallet_nonce` and additional fields. | Supported |
| `POST /openid4vp/sessions/{sessionId}/response` | Captures a form-encoded Wallet response for that session. | Supported |
| `POST /openid4vp/response` | Alternative form-encoded direct-post endpoint; required `state` identifies the session. | Supported |
| `GET /openid4vp/sessions/{sessionId}/events` | Chronological protocol capture events. | Supported |

The direct-post endpoints return `200` only when the presentation was captured and verified. A failed verifier check and a Wallet's decision to send no response are distinct outcomes. Inspect the session record and events; never substitute a screenshot for the missing callback.

## Known local limitations

| Area | Finding | Status / source |
| --- | --- | --- |
| Empty `credential_sets[].options` | The reference Android wallet displayed an error but did not POST `error=invalid_request`; the beta session captured only request retrieval. | Blocked for the required protocol assertion. [RI-WALLET-001](REFERENCE-WALLET-ISSUES.md) |
| Positive PID verification | The beta verifier received a `vp_token` but rejected it because the PID issuer URI did not match the issuer certificate SAN. | Verifier-blocked acceptance, not a Wallet failure. [MOCK-VERIFIER-001](REFERENCE-WALLET-ISSUES.md) |

`pkg/fcaf/MEMORY.md` additionally lists test-specific cases blocked because the public verifier validates malformed DCQL before it can create a signed request, or because it cannot expose the raw request/response feature required by the test. Treat that as coordination state and re-probe it when the service changes.

## Safe use in scenarios

- Use a source scenario under `config_templates/fcaf/wallet_solution/relying_party/scenarios/`; do not edit the generated aggregate pipeline directly.
- Persist the created `session_id` only as a pipeline output needed to fetch protocol evidence. Do not put live session URLs or tokens in fixtures.
- For verifier tests, bind validators to the exact scenario output containing the capture session/request/response. A different scenario's successful response is not fallback evidence.
- For issuer tests, use the session, events, and observed wallet JWKS to prove the relevant issuance exchange; inspect their actual shape first.
- Record a dated probe and update this document when a previously unknown capability becomes a test prerequisite.
