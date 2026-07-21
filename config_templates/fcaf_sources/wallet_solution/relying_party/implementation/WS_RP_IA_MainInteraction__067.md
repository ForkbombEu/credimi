# WS_RP_IA_MainInteraction_067

## Objective
Verify the HAIP requirement that, in the presentation flow via Redirects, Wallet follows `redirect_uri` sent by the Verifier.

## References
[OIDF.HAIP] section 5.1 
[OIDF.OID4VP] section 8.2

## Profile applicability
Same device
response_mode=direct_post.jwt
OIDF.HAIP

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request. 
2. Verifier sends presentation request using `response_mode` `direct_post.jwt` and including `response_uri`.
3. Verify if Wallet sends HTTP POST to `response_uri`.
4. Verifier response to Wallet HTTP POST includes `redirect_uri`. 
5. Wallet processes the request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Wallet successfully sends the HTTP POST payload to the response_uri.
4. The Verifier's response is received containing the redirect_uri parameter.
5. Verify the Wallet processes the response successfully and triggers the user agent to navigate to the redirect_uri sent by the Verifier.