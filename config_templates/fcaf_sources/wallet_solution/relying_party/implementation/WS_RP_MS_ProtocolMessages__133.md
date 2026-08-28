# WS_RP_MS_ProtocolMessages_133

## Objective
Test that Response Mode direct_post.jwt causes the Wallet to send the Authorization Response using an HTTP POST request.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request to the Wallet, with response mode=direct_post.jwt.
3. Wallet processes request, returns response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the wallet returns an Authorization response using an HTTP POST request.

