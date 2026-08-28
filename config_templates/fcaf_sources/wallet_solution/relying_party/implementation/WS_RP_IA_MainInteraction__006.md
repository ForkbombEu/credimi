# WS_RP_IA_MainInteraction_006

## Objective
Test that the Wallet processes DCQL-query with credential property "require_cryptographic_holder_binding" with value `true`, by handling credentials without cryptographic holder binding.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a credential of a specified credential type, where the credential does not have cryptographic holder binding.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with:
   a. the "require_cryptographic_holder_binding" property present with value `true`.
   b. a 'claims' property requesting the specified credential type.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet informs the user it has no matching credential available, and does not continue to present any credential to the verifier.