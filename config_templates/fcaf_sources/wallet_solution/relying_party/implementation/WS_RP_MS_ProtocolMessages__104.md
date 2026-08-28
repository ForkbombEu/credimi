# WS_RP_MS_ProtocolMessages_104

## Objective
Test that the credential_sets property "required" if set to False, the wallet can proceed with other valid sets flow without error.

## References
[OID4VP 6.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with "required" credential_sets property set to False. The Credential looked for is missing in the wallet.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet sees missing credential but continues with next set and completes successful matching.

