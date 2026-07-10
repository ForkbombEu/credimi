# WS_RP_IA_MainInteraction_021

## Objective

Test If the Wallet cannot satisfy any of the options in the claim_sets, it MUST NOT return any claims.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Verifier sends a DCQL query with a claim_sets that includes a claim the Wallet does not have.

## Expected results
1. Wallet will not return any claims.