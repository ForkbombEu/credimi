# WS_RP_IA_MainInteraction_061

## Objective
Test the Wallet handles a redirect_uri when a response_uri returns after a successful Response.

## References
[OID4VP Section 8.2]

## Profile applicability
Same device
response_mode=direct_post.jwt

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization request, with response_type=vp_token, response_mode=direct_post.jwt and parameter response_uri.
3. The wallet performs HTTP POST to the response_uri.
4. The verifier responds with HTTP 200 and a JSON body containing a redirect_uri.
5. Wallet processes JSON and triggers User agent.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet posts the Authorization Response to the response_uri.
4. Verify the wallet DOES NOT append the Authorization response to the redirect_uri.
5. Wallet opens redirect_uri, shows user success page.