# WS_RP_MS_ProtocolMessages_145

## Objective
Verify that the Wallet returns an invalid_client error when it receives client_metadata in a request from a Verifier that the Wallet already has locally stored metadata for.

## References
[OID4VP] Sections 8.5, 11.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has stored trusted metadata for Verifier A.

## Test Scenario
1. The Wallet engages with verifier A.
2. The Verifier sends an Authorization request that includes the client_metadata parameter, even though the Wallet already knows this client.
3. Wallet identifies the conflict between the incoming metadata and the stored trusted metadata.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. True: The Wallet does NOT proceed to credential selection or user authorization.
4. Verify: The Wallet returns an error response where the error parameter is exactly invalid_client.

