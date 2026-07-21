# WS_RP_MS_ProtocolMessages_046

## Objective
Verify that when the Wallet sends a POST request to the request_uri endpoint with a wallet_nonce and the returned Request Object contains a different or missing wallet_nonce claim, the Wallet rejects the Request Object.

## References
[OIDF.OID4VP] sections 8.5, 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request to the request_uri endpoint including a wallet_nonce value.
4. Wallet receives a signed Request Object whose wallet_nonce claim is different or absent.
5. Wallet validates the wallet_nonce.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. wallet_nonce is correctly included in the POST request body.
4. Wallet successfully receives the Request Object.
5. Wallet rejects the Request Object and returns an invalid_request error due to wallet_nonce mismatch or absence; presentation flow is not initiated.

