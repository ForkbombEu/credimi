# WS_RP_MS_ProtocolMessages_004

## Objective
Verify that the Wallet can process an Authorization Request sent as a Request Object by VALUE, where the Request Object includes the typ JOSE header parameter with the value oauth-authz-req+jwt.

## References
[RFC9101]; [OIDF.OID4VP] section 5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a Request Object embedded directly by value with typ header = oauth-authz-req+jwt.
3. Wallet inspects the typ JOSE header parameter of the Request Object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives and parses the Request Object directly from the Authorization Request.
3. Wallet validates the typ header value as oauth-authz-req+jwt and proceeds with the presentation flow.

