# WS_RP_MS_ProtocolMessages_109

## Objective
Test that the Wallet will accept a DCQL query without a claims "id" if claim_sets is NOT present

## References
[OID4VP 6.3]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a credential with "claims" object missing its "id" property, without a claim_sets present.
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet concludes matching without error.

