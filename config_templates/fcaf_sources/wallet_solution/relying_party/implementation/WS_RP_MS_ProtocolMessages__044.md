# WS_RP_MS_ProtocolMessages_044

## Objective
Verify that when the Wallet sends a POST request to the Verifier's Request URI Endpoint, all names and values in the request body are encoded using UTF-8.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet prepares the POST body for the request_uri endpoint.
4. Wallet sends the POST request. Inspect the encoding of all names and values in the body.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet correctly prepares the POST body.
4. All names and values in the POST body are encoded using UTF-8.

