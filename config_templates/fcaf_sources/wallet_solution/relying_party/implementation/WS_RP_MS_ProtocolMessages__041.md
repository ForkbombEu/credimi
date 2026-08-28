# WS_RP_MS_ProtocolMessages_041

## Objective
Verify that when the Verifier uses the redirect_uri Client Identifier Prefix with response_mode = direct_post.jwt and omits the response_uri Authorization Request parameter, the Wallet derives the response target from the Client Identifier and processes the request successfully.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with client_id using the redirect_uri: prefix, response_mode = direct_post.jwt, and no response_uri parameter.
3. Wallet parses the Authorization Request.
4. Wallet derives the response target from the Client Identifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet derives the response URI from the Client Identifier and processes the request successfully.

