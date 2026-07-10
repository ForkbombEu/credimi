# WS_RP_MS_Metadata_140

## Objective
Verify that when the Wallet sent a wallet_nonce in the POST request and the received Request Object does NOT contain a wallet_nonce claim, the Wallet terminates request processing.

## References
[OIDF.OID4VP] section 5.10.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request with a wallet_nonce value.
4. Wallet receives a Request Object where the wallet_nonce claim is absent.
5. Wallet checks for the wallet_nonce claim.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet_nonce is correctly sent in the POST request.
4. Wallet successfully receives the Request Object.
5. Wallet terminates request processing due to missing wallet_nonce claim; presentation flow is not initiated.

