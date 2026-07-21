# WS_RP_MS_ProtocolMessages_139

## Objective
Test that the Wallet responds with invalid_scope when the Requested scope value is empty.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request with a scope value that is intentionally empty (e.g. missing any value).
3. Wallet processes the request and identifies the scope as invalid.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The Wallet does NOT proceed to credential selection or user authorization.
4. The Wallet returns an error response where the error parameter is exactly invalid_scope.

