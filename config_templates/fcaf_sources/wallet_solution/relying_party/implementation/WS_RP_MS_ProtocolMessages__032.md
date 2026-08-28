# WS_RP_MS_ProtocolMessages_032

## Objective
Verify that when the Wallet receives an Authorization Request without a state parameter, it processes the request successfully without returning a state in the Authorization Response.

## References
[OIDF.OID4VP] section 5.2

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request without a state parameter.
3. Wallet parses the Authorization Request and prepares the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully parses the Authorization Request.
3. Wallet processes the request successfully without returning a state in the Authorization Response.

