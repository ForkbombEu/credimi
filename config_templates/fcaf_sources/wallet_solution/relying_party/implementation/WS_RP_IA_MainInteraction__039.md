# WS_RP_IA_MainInteraction_039

## Objective
Test that when the wallet returns a vp_token parameter for a multi-credential request, all returned key-value pairs simultaneously and correctly match their respective credential queries.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains two distinct valid credentials that can satisfy separate queries simultaneously.

## Test Scenario
1. The wallet engages with the verifier.
2. The Verifier sends an Authorization Request with a valid dcql_query requesting two separate credentials using two distinct query IDs.
3. The Wallet processes the request, the user Authorizes the presentation of both credentials, and the wallet transmits the response payload.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet prompts the user to release both requested credentials.
3. The Verifier receives an Authorization Response where the vp_token parameter is a single JSON object containing both distinct keys, with each key strictly mapping to its respective signed presentation value.