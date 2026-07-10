# WS_RP_MS_ProtocolMessages_132

## Objective
Test the Wallet puts the response content fields at the top level of the JWT payload.

## References
[OID4VP] Section 8.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The verifier sends an Authorization request to the Wallet, with response mode=direct_post.jwt.
3. Wallet processes request.
4. Test the decrypted JWT payload sent by the Wallet contains all response parameters as top-level JSON members (not nested inside a sub-object).

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the wallet returns an encrypted response.
4. The decrypted JWT payload contains all response parameters as top-level JSON members, not nested inside a sub-object.

