# WS_RP_IA_MainInteraction_024

## Objective
Verify that if the Wallet cannot pre-parse the data (e.g., it's encrypted), it still allows the user to proceed with the presentation rather than failing the request.

## References
[OID4VP 6.4]

## Profile applicability
Encrypted data

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Verifier sends a DCQL query with a values restriction. The Wallet holds a matching credential, but the data is encrypted/locked (the Wallet can't check the value yet).

## Expected results
1. The Wallet identifies the Type matches and allows the user to present the credential anyway (as the Credential Type and Path match the request.)