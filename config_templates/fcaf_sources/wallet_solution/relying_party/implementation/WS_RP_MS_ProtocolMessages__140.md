# WS_RP_MS_ProtocolMessages_140

## Objective
Test that the Wallet will not leave the verifier hanging after an error response, it must terminate the session.

## References
[OID4VP] Sections 5.2, 5.5, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request with a scope value that is intentionally empty (e.g. missing any value).
3. Wallet processes the request and identifies the scope as invalid.
4. Test the response returned by the Wallet to the Verifier.
5. Test the Wallet terminates the session.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet identifies the scope as invalid and does not proceed to credential selection or user authorization.
4. The wallet returns an error response where the error parameter is exactly invalid_scope.
5. The wallet terminates the session without leaving the verifier hanging; no further communication is sent.

