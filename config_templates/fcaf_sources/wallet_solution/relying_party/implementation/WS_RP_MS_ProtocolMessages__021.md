# WS_RP_MS_ProtocolMessages_021

## Objective
Verify that the Wallet rejects an Authorization Request containing both a dcql_query and a scope parameter simultaneously.

## References
[OIDF.OID4VP] section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Need to check if EUDIW supports both by DCQL and scopes - and if scopes are supported which values are defined.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request containing both a dcql_query and a scope parameter simultaneously.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet rejects the Authorization Request and returns an invalid_request error; presentation flow is not initiated.

