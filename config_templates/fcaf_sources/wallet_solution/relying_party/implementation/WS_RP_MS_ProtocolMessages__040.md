# WS_RP_MS_ProtocolMessages_040

## Objective
Verify that when the Verifier uses the redirect_uri Client Identifier Prefix and omits the redirect_uri Authorization Request parameter, the Wallet derives the redirect target from the Client Identifier and processes the request successfully.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with client_id using the redirect_uri: prefix and no redirect_uri parameter; response_mode is not direct_post.
3. Wallet parses the Authorization Request.
4. Wallet derives the redirect target from the Client Identifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet derives the redirect URI from the Client Identifier and processes the request successfully.

