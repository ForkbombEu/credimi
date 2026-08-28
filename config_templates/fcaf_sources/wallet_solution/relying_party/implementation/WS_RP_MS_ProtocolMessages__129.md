# WS_RP_MS_ProtocolMessages_129

## Objective
Test that when the JWK has kid parameter, the Wallet MUST include that same kid value, in the kid JWE header of the response.

## References
[OID4VP] Section 8.3.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier
2. The Verifier sends an Authorization request to the Wallet, with response mode=direct_post.jwt, and a client_metadata object containing a JWK with kid parameter
3. Wallet processes request

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. Verify the Wallet returns a response, without an error, and that it includes that same kid value, in the kid JWE header of its response.

