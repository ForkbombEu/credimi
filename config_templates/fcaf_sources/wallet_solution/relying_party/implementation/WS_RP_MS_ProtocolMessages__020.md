# WS_RP_MS_ProtocolMessages_020

## Objective
Verify that the Wallet processes an Authorization Request containing either a dcql_query parameter, or  ascope parameter representing a DCQL Query.

## References
[OIDF.OID4VP] section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions
Need to check if EUDIW supports both by DCQL and scopes - and if scopes are supported which values are defined.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request containing only a scope parameter representing a DCQL Query.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet processes the Authorization Request successfully using the scope parameter.

