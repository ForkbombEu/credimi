# WS_RP_MS_ProtocolMessages_033

## Objective
Verify that when the Wallet receives an Authorization Request containing a state parameter whose value is not a valid ASCII URL-safe string (e.g. a non-string type or contains disallowed characters), it rejects the request.

## References
[OIDF.OID4VP] sections 5.2, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction.
2. Wallet receives an Authorization Request containing an invalid state parameter.
3. Wallet parses and validates the state parameter.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully parses the Authorization Request.
3. Wallet rejects the Authorization Request and returns an invalid_request error; presentation flow is not initiated.

