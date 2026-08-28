# WS_RP_MS_ProtocolMessages_079

## Objective
Test if present the credential property "trusted_authorities", is made of objects defined in [OID4VP 6.1.1]

## References
[OID4VP] sections 6.1.1, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with a non-empty array "trusted_authorities" property, but which includes an object not defined in [OID4VP 6.1.1]
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet responds with an error: "invalid_request"

