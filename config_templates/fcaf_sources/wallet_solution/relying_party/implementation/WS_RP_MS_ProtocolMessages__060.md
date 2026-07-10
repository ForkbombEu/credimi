# WS_RP_MS_ProtocolMessages_060

## Objective
Test that the Wallet ignores unknown properties in the structure (non-top-level) of the DCQL-query.

## References
[OID4VP 6]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with unknown properties in the DCQL-query structure (non-top-level) to the wallet.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet responds to request without error, the Wallet should not change its behavior in any way due to the unknown property.

