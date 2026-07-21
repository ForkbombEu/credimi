# WS_RP_MS_Metadata_134

## Objective
Verify that when the Wallet requires encrypted Request Objects, it advertises public encryption keys in wallet_metadata.jwks and the supported algorithms via authorization_encryption_alg_values_supported and authorization_encryption_enc_values_supported, and the Wallet successfully decrypts the encrypted Request Object received from the Verifier.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request to the request_uri endpoint with wallet_metadata including jwks (public encryption keys) and supported encryption algorithms.
4. Wallet receives an encrypted Request Object using the advertised algorithms.
5. Wallet decrypts and processes the Request Object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet_metadata.jwks is correctly populated with public encryption keys and algorithm preferences.
4. Wallet successfully receives the encrypted Request Object.
5. Wallet successfully decrypts and processes the Request Object; presentation flow proceeds.

