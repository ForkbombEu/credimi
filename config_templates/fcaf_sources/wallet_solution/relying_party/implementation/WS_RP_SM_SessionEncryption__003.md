# WS_RP_SM_SessionEncryption_003

## Objective
Test the Wallet checks the JWE alg value exactly matches the alg value from the chosen JWK.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization request to the wallet, with response mode=direct_post.jwt, and a client_metadata object containing a JWK with an alg value different to the JWE alg value
3. Wallet processes request

## Expected results
1. Wallet-verifier interaction is successfully initiated
2. Wallet receives request
3. Verify the wallet does NOT send a response, instead returns an error

