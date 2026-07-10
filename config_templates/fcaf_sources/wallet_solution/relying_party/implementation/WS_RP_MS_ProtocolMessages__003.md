# WS_RP_MS_ProtocolMessages_003

## Objective
Verify that the Wallet rejects a Request Object where the typ header parameter equals oauth-authz-req+jwt.

## References
[RFC9101]; [OIDF.OID4VP] section 5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_undefined

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends Authorization Request as a Request Object by VALUE with typ header = oauth-authz-req+jwt.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully processes the Authorization Request and proceeds with the presentation flow. Wallet MUST NOT process request objects where the typ does not have the value oauth-authz-req+jwt

