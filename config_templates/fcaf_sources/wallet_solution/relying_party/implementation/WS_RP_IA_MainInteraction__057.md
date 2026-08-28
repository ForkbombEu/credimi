# WS_RP_IA_MainInteraction_057

## Objective
Test that Wallet accepts an HTTP 200 response with Content-Type: application/json after sending the Authorization Response to the response_uri.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends a request, with parameter 'response_mode=direct_post.jwt', and a valid response_uri.
3. The wallet processes the request.
4. The wallet responds with an Authorization response.
5. Verifier sends an HTTP status code of 200 + a JSON body

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. The wallet processes the request successfully and submits the Authorization Response to the response_uri.
4. The verifier returns an HTTP 200 status code with a JSON body containing a redirect URL or equivalent post-response payload.
5. Wallet does NOT return an error, returns user to URL provided in the redirect_uri parameter.