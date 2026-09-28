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

### FCAF Impact

- Tests: `WS_RP_MS_Metadata__110`, `WS_RP_MS_Metadata__133`,
  `WS_RP_MS_ProtocolMessages__143`, `WS_RP_MS_ProtocolMessages__144`.
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
