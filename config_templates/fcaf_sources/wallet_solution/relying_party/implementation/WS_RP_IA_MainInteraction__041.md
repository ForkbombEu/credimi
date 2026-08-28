# WS_RP_IA_MainInteraction_041

## Objective
Test that if the multiple parameter is explicitly set to false in a credential query, the wallet includes only one presentation in the corresponding response array.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains at least two distinct valid credentials that can satisfy the same single credential query.

## Test Scenario
1. The wallet engages with the verifier.
2. The Verifier sends an Authorization Request with a valid dcql_query requesting a credential, where the multiple attribute for that query is explicitly set to false.
3. The Wallet processes the request, the user selects and Authorizes the maximum no. of credential for presentation allowed, and the wallet transmits the response payload.


## Expected results
1. Wallet and Verifier can interact.
2. The Wallet recieves request.
3. The Verifier receives an Authorization Response where the array associated with the credential query ID inside the vp_token object contains exactly one presentation.