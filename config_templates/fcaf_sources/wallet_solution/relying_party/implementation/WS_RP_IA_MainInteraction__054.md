# WS_RP_IA_MainInteraction_054

## Objective
Test that the Wallet sends the Authorization Response to the provided response_uri using an HTTP POST request.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends a request, with parameter `response_mode=direct_post.jwt` and a `response_uri` of the verifier.
3. The wallet processes the request.
4. The wallet responds with an Authorization response.

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. The wallet processes the request successfully and prepares the Authorization Response for submission to the response_uri.
4. Verify the wallet submits the response to the response_uri