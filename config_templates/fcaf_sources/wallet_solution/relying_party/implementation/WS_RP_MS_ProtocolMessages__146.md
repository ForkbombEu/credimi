# WS_RP_MS_ProtocolMessages_146

## Objective
Verify that the Wallet returns invalid_client when the client_id resolves to a trusted registry entry, but the request attempts to override it with dynamic client_metadata.

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet is configured to resolve the client_id, via a trusted registry where Verifier A is pre-registered.

## Test Scenario
1. The Wallet engages with verifier A.
2. The Verifier sends an Authorization request containing both a known client_id, and a client_metadata parameter.
3. Wallet identifies the conflict between the incoming metadata and the pre-registered client.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. True: The Wallet does NOT proceed to credential selection or user authorization.
4. Verify: The Wallet returns an error response where the error parameter is exactly invalid_client.

