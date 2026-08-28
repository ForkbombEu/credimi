# WS_RP_DM_CredentialMetadata_CredentialStructure009

## Objective
Test that a wallet which holds no matching credentials to the DCQL "credentials", will NOT return any credentials

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with "credentials" that the wallet does NOT contain.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet will either return an "invalid request" error, or an empty VP token

