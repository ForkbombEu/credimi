# WS_RP_MS_ProtocolMessages_107

## Objective
test the Wallet will not attempt to return a credential when it can't find one due to "values" mismatch.

## References
[OID4VP] sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with "values" mismatched to any of the wallets credentials
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet correctly identifies that no credentials satisfy the query and returns access_denied (with the description that no credentials match).

