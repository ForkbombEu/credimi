# WS_RP_DM_CredentialMetadata_CredentialStructure007

## Objective
Test when valid credential_sets is present, the Wallet can handle these and applies the constraints when processing the request.

## References
[OID4VP 6]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a valid "credential_sets" property.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet MUST have used the constraints defined by "credential_sets", to create its response.

