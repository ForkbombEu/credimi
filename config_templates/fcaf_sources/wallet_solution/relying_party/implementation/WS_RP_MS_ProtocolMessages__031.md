# WS_RP_MS_ProtocolMessages_031

## Objective
Verify that when the Wallet receives an Authorization Request containing a state parameter with a valid ASCII URL-safe value, it processes the request and returns the state unchanged in the Authorization Response.

## References
[OIDF.OID4VP] section 5.2

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a state parameter with a valid ASCII URL-safe value.
3. Wallet parses the Authorization Request and prepares the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully parses the Authorization Request.
3. Wallet processes the request and returns the state parameter unchanged in the Authorization Response.

