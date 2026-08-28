# WS_RP_IA_MainInteraction_055

## Objective
When the response_uri parameter is present, the redirect_uri Authorization Request parameter MUST NOT be present. 

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends a request, with parameter `response_mode=direct_post.jwt` and a `response_uri` of the verifier, AND a redirect_uri.
3. The wallet processes the request.

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. Wallet returns an error.