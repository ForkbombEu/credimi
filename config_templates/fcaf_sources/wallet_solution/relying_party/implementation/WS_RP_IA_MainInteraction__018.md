# WS_RP_IA_MainInteraction_018

## Objective
Test that if claims is present, but claim_sets is absent the wallet treats all entries listed in claims as mandatory and returns them all.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a credentials object with claims but no claim_sets
3. Wallet processes DCQL

## Expected results
1. Wallet and Verifier can interact.
2. Wallet received query
3. Wallet identifies the missing claim_sets and returns all claims listed