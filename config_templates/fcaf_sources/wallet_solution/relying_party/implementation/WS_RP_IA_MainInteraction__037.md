# WS_RP_IA_MainInteraction_037

## Objective
Test that the vp_token parameter returned by the wallet, contains ID(s), aligning with the ID value of the credential query.

## References
[OID4VP Section 8.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a valid, matching credential requested by the Verifier.

## Test Scenario
1. The wallet engages with the verifier.
2. The Verifier sends an Authorization Request with a valid dcql_query requesting the wallet's stored credential using a specific, defined credential query ID.
3. The Wallet processes the request, the user Authorizes the presentation, and the wallet transmits the response payload.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet prompts the user to release the requested credential.
3. The Verifier receives an Authorization Response where the top-level keys within the vp_token object exactly match the credential query ID specified in the request.