# WS_RP_IA_MainInteraction_010

## Objective
Test that the Wallet processes DCQL-query without a credential property "require_cryptographic_holder_binding" as if it had value `true`.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains credentials of a specified type, with both a credential that has and a credential that does not have cryptographic holder binding.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a 'credential' without the "require_cryptographic_holder_binding" property
3. The Wallet evaluates the request and allows the user to continue with presenting matching credentials.
4. User selects the available credential(s) and approves to present it.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet allows the user to select only the credential that has cryptographic holder binding.
4. The wallet presents the selected credential to the verifier with a cryptographic proof of holder binding.