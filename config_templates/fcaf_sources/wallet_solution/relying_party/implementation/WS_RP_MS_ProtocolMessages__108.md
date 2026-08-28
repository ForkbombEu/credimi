# WS_RP_MS_ProtocolMessages_108

## Objective
Test that the Wallet will not accept a DCQL query if "claims" property "id" is not present, when claim_sets is present.

## References
[OID4VP] sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a credential with "claims" object missing its "id" property, but a claim_sets is present.
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet recognises missing "id" and occurring claim_sets, producing an error invalid_request.

