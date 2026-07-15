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
