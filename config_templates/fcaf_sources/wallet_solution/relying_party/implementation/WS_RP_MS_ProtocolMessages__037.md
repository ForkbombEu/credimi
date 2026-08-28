# WS_RP_MS_ProtocolMessages_037

## Objective
Test that when wallet receives a response_type of "vp_token" in an Authorization Request, the Wallet does NOT contain an OAuth 2.0 Authorization Code, Access Token, or Access Token Type in a successful response to the grant request.

## References
[OIDF.OID4VP] section 5.6

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet holds at least one credential matching the dcql_query in the Authorization Request.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with response_type = "vp_token".
3. Wallet parses and validates the Authorization Request.
4. Wallet selects a matching credential based on the dcql_query in the Authorization Request.
5. Wallet generates the Authorization Response to the grant request.
6. Inspect the full Authorization Response returned by the Wallet for the presence of OAuth 2.0 grant parameters (code, access_token, token_type).

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses and validates the Authorization Request.
4. Wallet identifies and selects a matching credential from those referenced by the dcql_query.
5. Wallet returns a successful Authorization Response to the grant request.
6. The Authorization Response does NOT contain an OAuth 2.0 Authorization Code (code), Access Token (access_token), or Access Token Type (token_type) parameter.

