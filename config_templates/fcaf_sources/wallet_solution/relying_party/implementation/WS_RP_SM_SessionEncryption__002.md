# WS_RP_SM_SessionEncryption_002

## Objective
Test the Wallet ensures that the selected Verifier JWK contains an alg parameter.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization request to the wallet, with response mode=direct_post.jwt, and a client_metadata object containing a JWK without an alg parameter.
3. Wallet processes request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the wallet does NOT send a response, instead returns an error.