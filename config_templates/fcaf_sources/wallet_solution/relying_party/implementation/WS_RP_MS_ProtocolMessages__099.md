# WS_RP_MS_ProtocolMessages_099

## Objective
Test the elements in "options" MUST be identifiers that reference elements in "credentials".

## References
[OID4VP 6.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a credential_sets property
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet confirms each "options" value maps to the "credentials" in the query, then successfully performs matching.

