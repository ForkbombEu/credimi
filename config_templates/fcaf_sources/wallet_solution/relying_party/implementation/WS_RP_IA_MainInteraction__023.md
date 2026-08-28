# WS_RP_IA_MainInteraction_023

## Objective
 Test the Wallet SHOULD treat a claim as if it did not exist in the credential, if its value does not match the one held in the wallets credential.

## References
[OID4VP 6.4]
[OID4VP 6.3]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Verifier sends a Claims Query containing a restriction on the values of a claim.

## Expected results
1. Wallet sees values do not match, logically considers that specific claim as not being present for this matching flow.