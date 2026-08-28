# WS_RP_IA_MainInteraction_013

## Objective
Verify that if a User unchecks a claim in the Wallet UI, it is excluded from the response.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query requesting claims "name" and "address", which the wallet has both.
3. Unchecking claim in UI excludes it from the response.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. User unselects "address" claim in the UI, that claim is absent in the vp_token.