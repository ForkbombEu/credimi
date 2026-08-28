# WS_RP_IA_MainInteraction_019

## Objective
Test that when a claim_sets is present, the Wallet automatically selects and returns only the first satisfiable combination from claim_sets based on the verifier's preference order, preventing disclosure of multiple sets.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains credentials that can satisfy all claim sets defined in the request.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL query containing a claim_sets property holding 3 distinct claim set objects in a specific order of preference.
3. The Wallet generates a response and prompts the user to authorize the release of data.
4. The wallet sends the response.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet recieves the request.
3. The user Authorizes the presentation without requiring them to manually choose between different claim sets.
4. The Verifier receives a response containing only the data requested in the first claim set object, proving that the wallet automatically prioritised the first match and excluded all subsequent satisfiable sets.