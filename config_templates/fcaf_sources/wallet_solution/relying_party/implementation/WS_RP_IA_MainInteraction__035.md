# WS_RP_IA_MainInteraction_035

## Objective
Test that the wallet returns the VP token in the Authorization response if the response type value is vp_token.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a valid, matching credential requested by the Verifier.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with the response_type value set to vp_token and a valid dcql_query requesting the wallet's stored credential.
3. The Wallet processes the request and the user Authorizes the presentation.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet prompts the user to release the requested credential.
3. The Verifier receives an Authorization Response containing a valid vp_token parameter holding the cryptographically signed credential data.