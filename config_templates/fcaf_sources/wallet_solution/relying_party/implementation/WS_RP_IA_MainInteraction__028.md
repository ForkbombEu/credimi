# WS_RP_IA_MainInteraction_028

## Objective
Test the wallet for non-selectively disclosable credentials, treats an empty query (no claims or claim_sets) as a request for the entire credential rather than returning an error or an empty token.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet holds credential that does NOT support selective disclosure

## Test Scenario
1. Verifier sends a DCQL query with a credentials object with no claims or claim_sets
2. Wallet processes DCQL

## Expected results
1. Wallet received query
2. Wallet identifies the format as non-selective and interprets the missing claims as a "Full Disclosure" request.