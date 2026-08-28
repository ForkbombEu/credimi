# WS_RP_MS_ProtocolMessages_119

## Objective
Verify negative case that the wallet cannot accept a claims path pointer that is an empty array.

## References
[OID4VP] sections 7, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer that is an empty array.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. The Wallet rejects the request by either:
    1. answering with an error with details (`invalid_request`), 
    2. answering with an error without providing details or,
    3. discontinuing the interaction.

