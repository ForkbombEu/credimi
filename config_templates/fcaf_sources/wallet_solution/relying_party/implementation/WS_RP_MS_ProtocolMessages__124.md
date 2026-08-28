# WS_RP_MS_ProtocolMessages_124

## Objective
Test that the Wallet ignores any unrecognized parameters when Response Mode = direct_post.jwt.

## References
[OID4VP Section 8.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends a request, with parameter `response_mode=direct_post.jwt` plus an additional unknown parameter.
3. The Wallet processes the request.
4. The Wallet responds with request object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet evaluates the request, ignores the unknown parameter, and proceeds without returning an error.
4. The wallet submits the Authorization Response to the verifier successfully.