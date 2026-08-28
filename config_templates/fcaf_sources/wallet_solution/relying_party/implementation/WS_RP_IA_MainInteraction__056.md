# WS_RP_IA_MainInteraction_056

## Objective
Test that the Wallet validates the response_uri according to the strict matching rules defined in [OID4VP] Section 5.9, and rejects the request if the value is invalid.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends a request, with parameter `response_mode=direct_post.jwt` and an invalid `response_uri` (not following rules in [OID4VP] section 5.9)
3. The wallet processes the request.

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. Wallet returns an error.