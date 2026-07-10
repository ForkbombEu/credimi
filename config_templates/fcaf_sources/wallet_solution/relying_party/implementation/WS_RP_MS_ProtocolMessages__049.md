# WS_RP_MS_ProtocolMessages_049

## Objective
Verify that when the Wallet receives an Authorization Request where the same parameter is present both in the query string and in the Request Object with different values, the Wallet uses only the value from the Request Object.

## References
[OIDF.OID4VP] section 5.10.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request where a parameter (e.g. nonce, response_mode) is present in both the query string and the Request Object with different values.
3. Wallet parses both the query string and the Request Object.
4. Wallet selects the parameter source for processing.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses both the query string and the Request Object.
4. Wallet uses only the parameter values from the Request Object; the conflicting query parameter value is ignored; presentation flow proceeds.

