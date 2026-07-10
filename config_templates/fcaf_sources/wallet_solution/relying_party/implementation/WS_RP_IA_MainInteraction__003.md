# WS_RP_IA_MainInteraction_003

## Objective
Test the Wallet accepts and processes DCQL-query with credential property "trusted_authorities", matching and returning a credential issued by an Issuer matching one of the Issuers identified in 'trusted_authorities'.

## References
[OID4VP Section 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains more than one credential from the issuer specified in the request.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with a valid "trusted_authorities" property.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet returns response without error and every Credential returned by the Wallet matches at least one of the conditions present in the corresponding trusted_authorities array.