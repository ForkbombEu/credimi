# WS_RP_MS_ProtocolMessages_131

## Objective
Test the Wallet will default to A128GCM for enc value, when Verifier metadata does NOT explicitly set it.

## References
[OID4VP Section 8.3.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The verifier sends an Authorization request to the Wallet, with response mode=direct_post.jwt, and a client_metadata object containing NO enc parameter.
3. Wallet processes request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the Wallet returns a response, without an error, and that it uses a default enc of A128GCM to perform its encryption.

