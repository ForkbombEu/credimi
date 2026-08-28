# WS_RP_IA_Supportive_002

## Objective
Verify that upon any HTTP error response from the Request-URI endpoint the Wallet terminates the process.

## References
[OIDF.OID4VP] section 5.10.2

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request to the request_uri endpoint.
4. Wallet receives an HTTP error response (e.g. 4xx or 5xx).

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully sends the POST request.
4. Wallet terminates the Authorization Request process; presentation flow is not initiated.