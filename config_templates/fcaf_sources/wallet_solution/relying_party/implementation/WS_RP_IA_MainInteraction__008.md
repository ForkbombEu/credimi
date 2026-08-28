# WS_RP_IA_MainInteraction_008

## Objective
Test that the Wallet processes DCQL-query with credential property "require_cryptographic_holder_binding" with value `false`, by handling credentials without cryptographic binding.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a credential of a specified credential type, where the credential does NOT have cryptographic holder binding.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL-query with a 'credential' with:
   a. the "require_cryptographic_holder_binding" property present with value `false`.
   b. a 'claims' property requesting the specified credential type.
3. The Wallet evaluates the request and allows the user to continue with presenting matching credentials.
4. The user selects and approves and presents the credential without cryptographic holder binding.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet allows the user to select the credential of the specified type.
4. The wallet successfully presents the selected credential to the verifier