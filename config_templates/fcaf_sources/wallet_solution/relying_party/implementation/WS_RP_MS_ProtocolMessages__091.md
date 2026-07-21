# WS_RP_MS_ProtocolMessages_091

## Objective
Test the Wallet Rejects a DCQL-query with a constraint on "trusted_authorities" with a `type` in an incorrect format.

## References
[OID4VP] sections 6.1.1, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a credential with a "trusted_authorities" property with its type property not as a string.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet rejects the request by either:
    1. answering with an error with details (`invalid_request`),
    2. answering with an error without providing details or,
    3. discontinuing the interaction.

