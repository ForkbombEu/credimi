# WS_RP_MS_ProtocolMessages_007

## Objective
Verify that the Wallet rejects a Request Object where the typ header parameter does not equal oauth-authz-req+jwt.

## References
[RFC9101]; [OIDF.OID4VP] section 5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a request_uri parameter.
3. Wallet fetches the Request Object from the reference URI.
4. Wallet inspects the typ JOSE header parameter of the retrieved Request Object; value is set to an invalid value (not oauth-authz-req+jwt).

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives the Authorization Request and recognizes the request_uri parameter.
3. Wallet successfully fetches the Request Object from the reference URI.
4. Wallet does NOT process the request and returns an invalid_request error.

