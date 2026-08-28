# WS_RP_MS_ProtocolMessages_130

## Objective
Test the Wallet chooses enc from Verifier metadata when explicitly set, to perform the response encryption.

## References
[OID4VP Section 8.3.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The verifier sends an Authorization request to the Wallet, with response mode=direct_post.jwt, and a client_metadata object containing an enc parameter.
3. Wallet processes request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the wallet returns a response, without an error, and that it uses that enc value in the verifier metadata to perform its encryption.

