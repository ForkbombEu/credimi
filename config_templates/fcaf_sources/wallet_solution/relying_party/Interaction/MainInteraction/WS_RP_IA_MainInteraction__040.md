# WS_RP_IA_MainInteraction_040

## Objective

Test that if the multiple parameter is omitted in a credential query, it defaults to false, and the wallet includes only one presentation in the corresponding response array.

## References

- [OpenID4VP] Sections 8, 6.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. The wallet contains at least two distinct valid credentials that can satisfy the same single credential query (e.g., two different digital diplomas).

## Test Scenario

1. The wallet engages with the verifier.
2. The Verifier sends an Authorization Request with a valid dcql_query requesting a credential, where the multiple attribute for that query is completely omitted.
3. The Wallet processes the request, the user selects and Authorizes the maximum no. of credential for presentation allowed, and the wallet transmits the response payload.

## Expected results

1. Wallet and Verifier can interact.
2. The Wallet recieves request.
3. The Verifier receives an Authorization Response where the array associated with the credential query ID inside the vp_token object contains exactly one presentation.
