# WS_RP_SH_Cryptography_Encryption002

## Objective
Verify that if `encrypted_response_enc_values_supported` within client metadata from the Verifier, does not include A256GCM, and Wallet only supports A256GCM, the Wallet responds with an error.

## References
[OIDF.HAIP] section 5 (introduction) 
[OIDF.OID4VP] section 8.3
[RFC7516] section 4.1.2

## Profile applicability
Wallet supports only A256GCM

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using response mode direct_post.jwt, whose client metadata lists encrypted_response_enc_values_supported without A256GCM.
3. Wallet processes the request.
4. Wallet returns its response to the Authorization Request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives the Authorization Request
3. Wallet successfully proccesses request.
4. Wallet returns an access_denied error.