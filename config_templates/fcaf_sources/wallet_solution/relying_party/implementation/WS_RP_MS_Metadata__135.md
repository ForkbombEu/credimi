# WS_RP_MS_Metadata_135

## Objective
Verify that when the Wallet has indicated that encryption is required (via wallet_metadata.jwks and supported algorithms) and the Verifier returns an unencrypted Request Object, the Wallet rejects the Request Object.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request with wallet_metadata.jwks indicating encryption is required.
4. Wallet receives an unencrypted Request Object.
5. Wallet inspects the response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet_metadata.jwks is correctly populated.
4. Wallet successfully receives the response.
5. Wallet rejects the unencrypted Request Object and returns an invalid_request error; presentation flow is not initiated.

