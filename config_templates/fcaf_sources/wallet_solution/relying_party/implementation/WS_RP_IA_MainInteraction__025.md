# WS_RP_IA_MainInteraction_025

## Objective
Test that if the wallet can only return part of the claims in the claim_sets object, it will not return that set but move onto next one.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
Verifier sends a claim_set requiring 3 claims, but the Wallet only has 2 of them, it MUST NOT return that set. It must move to the next option in the array.

## Expected results
The Wallet only has 2 of them, so it MUST NOT return that set. It must move to the next option in the array. The response should reflect this.