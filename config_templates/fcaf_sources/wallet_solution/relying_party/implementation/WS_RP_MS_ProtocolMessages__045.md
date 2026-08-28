# WS_RP_MS_ProtocolMessages_045

## Objective
Verify that when the Wallet sends a POST request to the request_uri endpoint with a wallet_nonce value, the Verifier returns a signed Request Object containing the same wallet_nonce as the wallet_nonce claim, and the Wallet validates the match.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request to the request_uri endpoint including a wallet_nonce value.
4. Wallet receives a signed Request Object containing the same wallet_nonce claim.
5. Wallet validates that the wallet_nonce in the Request Object matches the value sent.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. wallet_nonce is correctly included in the POST request body.
4. Wallet successfully receives the signed Request Object.
5. Wallet validates the wallet_nonce match and continues processing the request.

