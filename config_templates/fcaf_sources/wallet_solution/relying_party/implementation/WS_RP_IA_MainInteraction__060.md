# WS_RP_IA_MainInteraction_060

## Objective
Test the Wallet redirects the user agent to the redirect_uri if this parameter is included in the initial Authorization Request.

## References
[OID4VP Section 8]

## Profile applicability
Same device
response_mode=direct_post

## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization request, with response_type=vp_token, and parameter redirect_uri.
3. The wallet processes the request.
4. The wallet constructs the URL.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet processes the request successfully and proceeds to construct the URL with the Authorization Response appended.
4. Verify the wallet appends the Authorization response to the redirect_uri, opens URL in the browser.