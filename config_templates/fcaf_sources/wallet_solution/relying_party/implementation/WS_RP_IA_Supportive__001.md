# WS_RP_IA_Supportive_%001

## Objective
Verify that, in the presentation flow via Redirects, Wallet processes successfully HTTP response to the Wallet's HTTP POST to the `response_uri', including `redirect_uri`.

## References
[OIDF.HAIP] section 5.1 
[OIDF.OID4VP] section 8.2

## Profile applicability
Presentations via Redirects

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request. 
2. Verifier sends presentation request using `response_mode` `direct_post` and including `response_uri`.
3. The Wallet sends an HTTP POST to the response_uri.
4. Verifier responds to the Wallet's HTTP POST with a body including redirect_uri.
5. Wallet processes the request.

## Expected results
1. The Verifier successfully initiates the presentation request flow with the end-user.
2. The Wallet receives the presentation request with response_mode = direct_post and a response_uri parameter.
3. The Wallet sends the Authorization Response to the response_uri via HTTP POST.
4. The Wallet receives the verifier's HTTP response containing a redirect_uri.
5. The Wallet successfully processes the verifier's response and opens the redirect_uri in the user agent.
