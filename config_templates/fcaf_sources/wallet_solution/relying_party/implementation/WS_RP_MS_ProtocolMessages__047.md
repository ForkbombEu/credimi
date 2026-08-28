# WS_RP_MS_ProtocolMessages_047

## Objective
Verify that when the Wallet receives the Request URI response, it has Content-Type application/oauth-authz-req+jwt and the body is a signed (optionally encrypted) Request Object conforming to RFC9101.

## References
[OIDF.OID4VP] section 5.10.1; [RFC9101]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request to the request_uri endpoint and receives the HTTP response.
4. Inspect the Content-Type header and body of the response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully receives the HTTP response from the request_uri endpoint.
4. HTTP response has Content-Type: application/oauth-authz-req+jwt and the body is a signed (optionally encrypted) Request Object conforming to RFC9101; presentation flow proceeds.

