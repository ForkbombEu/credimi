# WS_RP_MS_ProtocolMessages_113

## Objective
Test that the Wallet will NOT accept a DCQL query with credentials with a claims property but without a claims "path" property.

## References
[OID4VP] Sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a "claims" object missing its "path" property.
3. Wallet handles Query.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet returns an invalid_request error.