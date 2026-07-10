# WS_RP_MS_Metadata_138

## Objective
Verify that when the Wallet sent a wallet_nonce in the POST request and the received Request Object contains the matching wallet_nonce claim, the Wallet validates the match and continues processing.

## References
[OIDF.OID4VP] section 5.10.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request including a wallet_nonce value.
4. Wallet receives a Request Object containing the matching wallet_nonce claim.
5. Wallet validates the wallet_nonce match.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet_nonce is correctly sent in the POST request.
4. Wallet successfully receives the Request Object.
5. Wallet validates the wallet_nonce match and continues processing the request.

