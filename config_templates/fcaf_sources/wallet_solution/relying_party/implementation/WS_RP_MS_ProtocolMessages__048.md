# WS_RP_MS_ProtocolMessages_048

## Objective
Verify that when the Wallet receives a Request URI response with an incorrect Content-Type (not application/oauth-authz-req+jwt), the Wallet rejects or fails to parse the response.

## References
[OIDF.OID4VP] sections 8.5, 5.10.1; [RFC9101]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request to the request_uri endpoint.
4. Wallet receives a response with an incorrect Content-Type (e.g. application/json).
5. Wallet inspects and attempts to parse the response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully sends the POST request.
4. Wallet receives the response.
5. Wallet rejects or fails to parse the response as a valid Request Object; presentation flow is not initiated.

