# WS_RP_MS_ProtocolMessages_039

## Objective
Verify that when the Wallet receives an Authorization Request using the redirect_uri Client Identifier Prefix where the embedded URI is not a valid HTTPS URL, the Wallet rejects the request and returns an invalid_request error.

## References
[OIDF.OID4VP] sections 8.5, 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the redirect_uri: prefix where the embedded URI is not a valid HTTPS URL (e.g. malformed URI or non-HTTPS scheme).
3. Wallet parses the Authorization Request.
4. Wallet validates the embedded URI.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet rejects the Authorization Request and returns an invalid_request error due to the invalid redirect URI; presentation flow is not initiated.

