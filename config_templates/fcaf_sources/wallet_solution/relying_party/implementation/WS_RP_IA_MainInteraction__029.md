# WS_RP_IA_MainInteraction_029

## Objective
Test that if credential_sets is provided, the wallet evaluates the request by satisfying all required or omitted Credential Set Queries in the array, while treating Credential Set Queries explicitly marked as required false as optional.

## References
[OID4VP 6.2], [OID4VP 6.4.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a valid credential that satisfies the criteria for Credential Query A, but does not contain any credential that satisfies Credential Query B.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL query containing a credential_sets array with two Credential Set Query objects:
Set 1: Contains an options array for Credential Query A and is marked as required: true.
Set 2: Contains an options array for Credential Query B and is explicitly marked as required: false.
3. The Wallet processes the request.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet prompts the user to release the credential matching Credential Query A, indicating that the overall presentation requirements can be met despite the missing optional credential.
3. The Verifier receives an response containing the credential for Credential Query A, completing the interaction successfully.