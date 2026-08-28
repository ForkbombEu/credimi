# WS_RP_MS_ProtocolMessages_100

## Objective
Test that if elements in "options" are NOT identifiers that reference elements in "credentials", the wallet will reject query and remain privacy preserving.

## References
[OID4VP] sections 6.2, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a credential_sets property with different options' element mapping.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet stops the transaction, producing a privacy preserving error response.

