# WS_RP_IA_MainInteraction_026

## Objective
Test when the Wallet cannot deliver all claims requested by the Verifier, because it doesn't have one of them, then it does NOT return the respective Credential.

## References
[OID4VP] Section 6.4, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a credential that satisfies only one of the two required claims requested by the Verifier.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL query requesting a specific credential type with 2 required claims.
3. The Wallet processes the request.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet does not prompt the user to release the partially matching credential, or indicates to the user that no matching credentials are available.
3. The Verifier receives a privacy-preserving error response access denied.