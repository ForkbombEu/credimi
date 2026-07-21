# WS_RP_IA_MainInteraction_027

## Objective
Test when the Wallet cannot deliver all claims requested by the Verifier because one of them has a non-matching value, it does NOT return the respective Credential.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a credential that holds both requested claims, but one of those claims has a value that does not match the constraints requested by the Verifier.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL query requesting a specific credential type with 2 required claims containing specific value constraints.
3. The Wallet processes the request.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet does not prompt the user to release the non-matching credential, or indicates to the user that no valid matching credentials are available.
3. The Verifier receives a privacy-preserving response completely excluding the credential with the non-matching value.