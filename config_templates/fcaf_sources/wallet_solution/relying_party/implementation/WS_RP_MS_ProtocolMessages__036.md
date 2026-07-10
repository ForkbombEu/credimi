# WS_RP_MS_ProtocolMessages_036

## Objective
Test that when wallet receives a response_type of "vp_token" in an Authorization Request, that a successful response includes the vp_token parameter.

## References
[OIDF.OID4VP] section 5.6

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with response_type = "vp_token".
3. Wallet parses and validates the Authorization Request.
4. Wallet selects a matching credential based on the request parameters (e.g. dcql_query).
5. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses and validates the Authorization Request.
4. Wallet identifies and selects a matching credential.
5. Wallet returns a successful Authorization Response that includes the vp_token parameter.

