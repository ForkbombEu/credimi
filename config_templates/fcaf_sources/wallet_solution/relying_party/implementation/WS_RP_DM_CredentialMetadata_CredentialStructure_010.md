# WS_RP_DM_CredentialMetadata_CredentialStructure010

## Objective
Test that if the credentials' property "claim_sets" is present then it is used as per the rules defined in [OID4VP 6.4.1]

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with a claim_sets.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet correctly uses the "claims_sets" to determine which claims to send, as per the rules in [OID4VP 6.4.1]

