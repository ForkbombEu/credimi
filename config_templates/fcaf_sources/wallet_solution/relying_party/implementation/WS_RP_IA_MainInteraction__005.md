# WS_RP_IA_MainInteraction_005

## Objective
Test that the Wallet processes DCQL-query with credential property "require_cryptographic_holder_binding" with value `true`, by handling credentials with cryptographic holder binding.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a credential of a specified credential type, where the credential has cryptographic holder binding.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL-query with a 'credential' with:
   a. the "require_cryptographic_holder_binding" property present with value `true`.
   b. a 'claims' property requesting the specified credential type.
3. The Wallet evaluates the request and allows the user to continue with presenting matching credentials.
4. User selects the available credential and approves to present it.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet allows the user to select the credential of the specified type.
4. The wallet presents the selected credential to the verifier with a cryptographic proof of holder binding.