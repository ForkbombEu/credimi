# WS_RP_MS_ProtocolMessages_077

## Objective
Test the Wallet rejects a DCQL-query with the credential property "trusted_authorities" left empty.

## References
[OID4VP] sections 6.1, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with an empty "trusted_authorities" property.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet rejects the request by either:
    1. answering with an error with details (`invalid_request`),
    2. answering with an error without providing details or,
    3. discontinuing the interaction.

