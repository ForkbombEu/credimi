# WS_RP_IA_MainInteraction_020

## Objective
Verify that if the Wallet can satisfy multiple options in claim_sets, it selects the one appearing earliest in the array.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has data for #2 and #3 of verifiers claim_sets

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends claim_sets with 3 options (#1,#2,#3)

## Expected results
1. Wallet and Verifier can interact.
2. Wallet responds with #2