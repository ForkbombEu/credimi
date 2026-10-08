<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Reference Wallet Issues

## RI-WALLET-001: Empty DCQL credential_sets options is not returned to the verifier

### Summary

The reference Android wallet detects a DCQL `credential_sets[].options` array
that is present but empty and displays an error page, but it does not return
the required `invalid_request` response to the verifier.

### Reproduction

1. Create an OpenID4VP session with `response_mode: direct_post` at
   `https://beta-capture-wallet.credimi.io/openid4vp/sessions`.
2. Send a signed request containing:

```json
{
  "dcql_query": {
    "credentials": [
      {
        "id": "pid",
        "format": "dc+sd-jwt",
        "meta": { "vct_values": ["urn:eudi:pid:1"] }
      }
    ],
    "credential_sets": [{ "options": [] }]
  }
}
```

3. Unlock the wallet and open the generated deeplink.

### Observed

- The wallet briefly displays `Oups! Something went wrong` with
  `options cannot be empty`.
- The beta capture session records `vp_request_retrieved` only.
- No POST reaches the advertised `response_uri`; therefore no
  `error=invalid_request` is captured.

### Expected

The wallet must send an OpenID4VP `direct_post` error response with
`error=invalid_request`, in addition to the user-visible error page.

### Control

The same capture verifier successfully recorded the reference wallet's
`invalid_request` response for the adjacent missing-options case
`WS_RP_MS_ProtocolMessages__096`, so the capture endpoint accepts this error
response class.

### FCAF Impact

- Test: `WS_RP_MS_ProtocolMessages__097`
- The Credimi test requires both the exact protocol error and a strict error
  page; the reference wallet currently fails the protocol assertion.

## MOCK-VERIFIER-001: Beta capture verifier rejects valid PID response on certificate SAN

For `WS_RP_MS_ProtocolMessages__099`, the reference wallet reaches the request,
shares the PID, and submits a `vp_token`. The beta capture verifier marks the
presentation invalid because the PID `iss` URI does not match a SAN-URI or
SAN-DNS entry in the issuer certificate.

Observed verifier event:

```text
pid[0]: The 'iss' claim in the payload does not match a 'SAN-URI' name and the
domain extracted from the HTTPS URI does not match a 'SAN-DNS' name in the x5c
certificate.
error: invalid_request
error_description: One or more presentations failed verification.
```

The wallet-side positive flow is evidenced through `Share`, PIN approval, and
the submitted `vp_token`. Verifier acceptance is blocked by certificate/SAN
validation and must not be reported as a wallet failure.

## RI-WALLET-002: Client Identifier Prefix rejections are not returned to the verifier

### Summary

The reference Android wallet rejects requests whose Client Identifier Prefix it
does not support, or that violate the prefix's rules, and displays an error
page, but it does not return the required `invalid_request` response to the
verifier.

### Reproduction

Create an OpenID4VP session at
`https://beta-capture-wallet.credimi.io/openid4vp/sessions` with one of the
`scenarios/fcaf-wallet-solution-relying-party-client-id-prefix-controls.yaml`
request bodies, unlock the wallet, and open the generated deeplink:

- a signed Request Object with `client_id` `redirect_uri:<response_uri>`;
- an `origin:` Client Identifier outside the Digital Credentials API;
- an unsupported `fcaf_unsupported_prefix:` Client Identifier;
- plain unsigned URL parameters with an `https://` Client Identifier.

### Observed

Wallet 2026.09.42, 28/09/2026:

- The wallet retrieves the Request Object where one is offered, then logs
  `Invalid resolution: UnsupportedClientIdPrefix`, or
  `InvalidClientIdPrefix(value=Invalid client_id: 'ORIGIN' cannot be used as a
  Client ID prefix)` for `origin:`.
- It displays `Oups! Something went wrong`.
- The beta capture session records at most `vp_request_retrieved`; no POST
  reaches the response endpoint, so no `error=invalid_request` is captured.

### Expected

OpenID4VP 1.0 Section 8.5 lists an unsupported Client Identifier Prefix and a
violation of a prefix's requirements as `invalid_request` cases; the source
tests require an error response whose `error` is exactly `invalid_request`.

### Same defect for Client Identifier binding failures

The `scenarios/fcaf-wallet-solution-relying-party-client-id-binding-controls.yaml`
requests show the same behaviour on 28/09/2026: the wallet retrieves the
Request Object, logs `InvalidJarJwt(cause=ClientId not found in certificate's
subject alternative names)` for an `x509_san_dns:` Client Identifier absent
from the leaf, `InvalidJarJwt(cause=ClientId does not match leaf
certificate's SHA-256 hash)` for a wrong `x509_hash:` value, and
`UnsupportedClientIdPrefix` for `decentralized_identifier:`, then shows the
error page without posting `invalid_request`. A signed Request Object without
`x5c` and a plain request with a `redirect_uri:http://` Client Identifier end
on the same error page; the captured logcat window recorded no reason for
those two.

### FCAF Impact

- Tests: `WS_RP_MS_Metadata__110`, `WS_RP_MS_Metadata__126`,
  `WS_RP_MS_Metadata__130`, `WS_RP_MS_Metadata__133`,
  `WS_RP_MS_ProtocolMessages__039`, `WS_RP_MS_ProtocolMessages__143`,
  `WS_RP_MS_ProtocolMessages__144`, `WS_RP_SM_RpIntegrity__007`,
  `WS_RP_SM_RpIntegrity__014`.
- The request-shape and visual-evidence assertions pass; the reference wallet
  fails the protocol assertion.

## RI-WALLET-003: Malformed client_metadata is not returned to the verifier

### Summary

The reference Android wallet rejects a signed Request Object whose
`client_metadata` is not a JSON object and displays an error page, but it does
not return the required `invalid_request` response to the verifier.

### Reproduction

Create an OpenID4VP session at
`https://beta-capture-wallet.credimi.io/openid4vp/sessions` with the
`create-non-object-client-metadata` body of
`scenarios/fcaf-wallet-solution-relying-party-metadata-request-controls.yaml`
(`request_mutation.request_object.set` `/client_metadata` to a string), unlock
the wallet, and open the generated deeplink.

### Observed

Wallet 2026.09.42, 28/09/2026:

- The wallet retrieves the Request Object by POST and logs `Unexpected JSON
  token ... Expected start of the object '{', but had '"' instead at path:
  $.client_metadata`.
- It displays `Oups! Something went wrong`.
- The beta capture session records `vp_request_retrieved` only; no
  `error=invalid_request` reaches the response endpoint.

### Expected

OpenID4VP 1.0 Section 5.1 requires `client_metadata` to be a JSON object; the
source test requires the Wallet to reject the request with `invalid_request`.

### FCAF Impact

- Test: `WS_RP_MS_Metadata__106`.
- The delivered-request and visual-evidence assertions pass; the reference
  wallet fails the protocol assertion.

## RI-WALLET-004: Authorization Response encrypted with A128CBC-HS256 instead of GCM

### Summary

The reference Android wallet encrypts the `direct_post.jwt` Authorization
Response with `enc: A128CBC-HS256` when the verifier also offers `A128GCM` and
`A256GCM`.

### Reproduction

Run `scenarios/fcaf-wallet-solution-relying-party-dcql-session-encryption.yaml`
against `https://capture-wallet.credimi.io`. On 08/10/2026 its Request Object
carried the generated `client_metadata.encrypted_response_enc_values_supported`
`[A128GCM, A256GCM, A128CBC-HS256]` with one `ECDH-ES` P-256 key.

### Observed

Pipeline run reported on 08/10/2026: the captured compact JWE in
`observed.wallet_response.value.response` has protected header
`enc: A128CBC-HS256`.

### Expected

The source test requires `enc` to be `A256GCM` or `A128GCM`. HAIP 1.0 Section 5
requires Wallets to support `A128GCM` or `A256GCM` and says a Wallet supporting
both SHOULD use `A256GCM`. OpenID4VP allows any advertised value, so the choice
is not a protocol error, but it is not the algorithm the source expects.

### FCAF Impact

- Test: `WS_RP_SM_SessionEncryption__001b` fails on
  `content_encryption_is_a256gcm_or_a128gcm`. The assertion is correct and the
  verifier list is left as Capture generates it, so the result reflects the
  wallet's own choice.
- `WS_RP_SM_SessionEncryption__007`–`009` advertise a single GCM value or only
  the two GCM values in dedicated scenarios, so they test whether the wallet
  honours a narrowed list, not this default choice.

## MOCK-VERIFIER-002: Beta signs decentralized_identifier requests with the X.509 key

On 28/09/2026 a beta session with `client_id_scheme: decentralized_identifier`
and no `request_behavior` delivered a Request Object whose `client_id` was
`decentralized_identifier:did:web:beta-capture-wallet.credimi.io:openid4vp` but
whose JOSE header carried `kid: credimi-fake-verifier-key` and the X.509
verifier `x5c`. The signature verified against that `x5c` leaf and not against
the only verification method in `/openid4vp/did.json`
(`#credimi-fake-verifier-did-key`). The published contract says this scheme
signs with a separate `did:web` key.

### FCAF Impact

- `WS_RP_IA_Metadata__015`, `016` and `WS_RP_SM_RpIntegrity__006` require the
  DID-published key to verify the request, so beta cannot satisfy them.
- `WS_RP_SM_RpIntegrity__007` still holds its precondition, because
  `request_behavior.signing_key: unrelated` also signs with a key the DID
  document does not list, but on beta the normal DID request would satisfy it
  too. The reference wallet additionally rejects `decentralized_identifier:`
  as an unsupported prefix, so it never reaches the signing-key check.

## MOCK-VERIFIER-003: Beta signed a configured redirect_uri beside response_uri (resolved)

On 30/09/2026 a beta `direct_post.jwt` session created with a top-level
`redirect_uri` delivered a Request Object carrying both `response_uri` and
`redirect_uri` (the configured URI plus its fresh `response_code`). OpenID4VP
1.0 Section 8.2: "When the response_uri parameter is present, the redirect_uri
Authorization Request parameter MUST NOT be present", and the Wallet must answer
`invalid_request`. Wallet 2026.09.42 did exactly that:

```text
error: invalid_request
error_description: RedirectUriMustNotBeProvided
```

The behaviour came from upstream `a2fc6a0`, synced on 28/09/2026.

### Resolution

Fixed on beta later on 30/09/2026: the configured `redirect_uri`, with its
`response_code`, is returned at session creation and in the Response Endpoint
reply, and is no longer signed into the Request Object. A probe session
delivered `response_uri` without `redirect_uri`.

### FCAF Impact

- `WS_RP_IA_MainInteraction__064` no longer needs a `request_mutation`
  workaround, and still asserts that the delivered request omits
  `redirect_uri`.
- `WS_RP_IA_MainInteraction__057`, `061`, `067`, `WS_RP_IA_Supportive__001`,
  and `WS_RP_MS_ProtocolMessages__127`, `128` are no longer rejected for this
  reason. None has a reference-Wallet run since the fix.
