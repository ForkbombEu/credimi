# WS_RP_DM_CredentialMetadata_CredentialStructure008

## Objective
Test the wallet can process and match each entry in the credentials' parameter of the DCQL, with credentials present in the wallet.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet response must show that it processes each entry in the credentials' parameter, where appropriate matching with credentials present in wallet.

