# WS_RP_MS_ProtocolMessages_043

## Objective
Verify that when the Verifier's Request URI Endpoint is only accessible over plain HTTP (not HTTPS), the Wallet refuses to connect or rejects the request due to the non-HTTPS scheme.

## References
[OIDF.OID4VP] sections 8.5, 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a request_uri using the http:// scheme (not https://) and request_uri_method = post.
3. Wallet prepares to send the POST request to the request_uri endpoint.
4. Wallet attempts to connect.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet detects that the request_uri scheme is not HTTPS.
4. Wallet refuses to connect or rejects the request and returns an invalid_request error due to non-HTTPS scheme; presentation flow is not initiated.

