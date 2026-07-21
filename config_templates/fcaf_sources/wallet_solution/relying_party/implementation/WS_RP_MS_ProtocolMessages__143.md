# WS_RP_MS_ProtocolMessages_143

## Objective
Test the Wallet responds with invalid_request when the Wallet does not support the Client Identifier Prefix passed in the Authorization Request.

## References
[OID4VP] Sections 8.5, 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request with an unsupported client_id prefix.
3. Wallet processes the request and identifies unsupported prefix.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. True: The Wallet does NOT proceed to credential selection or user authorization.
4. Verify: The Wallet returns an error response where the error parameter is exactly invalid_request.

