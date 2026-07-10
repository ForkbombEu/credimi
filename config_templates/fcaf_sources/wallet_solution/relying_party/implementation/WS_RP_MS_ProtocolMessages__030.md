# WS_RP_MS_ProtocolMessages_030

## Objective
Verify that when the Wallet supports scope-based Presentation requests and receives an Authorization Request containing a scope value it does not recognize, it rejects the request with an appropriate error.

## References
[OIDF.OID4VP] Section 5.5, 8.5; [RFC6749]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing only a scope parameter with a value NOT recognized by the Wallet.
3. Wallet parses the Authorization Request and attempts to resolve the scope value.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives and parses the Authorization Request.
3. Wallet rejects the Authorization Request and returns an invalid_scope error; presentation flow is not initiated.

