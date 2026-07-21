# WS_RP_MS_ProtocolMessages_042

## Objective
Verify that when the Wallet sends a request to the Verifier's Request URI Endpoint, it uses HTTP POST over HTTPS with Content-Type application/x-www-form-urlencoded and Accept header application/oauth-authz-req+jwt.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a request_uri and request_uri_method = post.
3. Wallet prepares to send the POST request to the request_uri endpoint.
4. Wallet sends the POST request. Observe the HTTP method, scheme, Content-Type header, and Accept header used.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet correctly prepares the POST request.
4. Wallet sends an HTTP POST request to the request_uri using HTTPS, with Content-Type: application/x-www-form-urlencoded and Accept: application/oauth-authz-req+jwt.

