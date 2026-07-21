# WS_RP_IA_MainInteraction_046

## Objective
Test for responses of size > URL limits of user agents, the Wallet can respond to the Verifier using the direct_post mechanism.

## References
[OID4VP Section 8]

## Profile applicability
Same device
response_mode=direct_post.jwt

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a large credential where the resulting presentation size exceeds the typical URL length limit of a user agent.

## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends a request for the large credential, with parameter `response_mode=direct_post.jwt` and a `response_uri` of the verifier.
3. The wallet processes the request.
4. The wallet responds with an Authorization response.

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. The wallet processes the request successfully, recognises that the response exceeds the user-agent URL limit, and prepares to submit it to the response_uri.
4. Verify the wallet submits the response to the response_uri, with the Authorization Response as body of the HTTP-request using the POST method.