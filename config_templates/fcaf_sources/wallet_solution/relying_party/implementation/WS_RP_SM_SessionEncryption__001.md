# WS_RP_SM_SessionEncryption_001

## Objective
Verify that if the Wallet encrypts the Authorization Response it uses an unsigned, encrypted JWT.

## References
[OID4VP Section 8]
[RFC7519]  

## Profile applicability
Encrypted response

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The wallet engages with the verifier.
2. The verifier sends an Authorization request to the wallet, with response mode=direct_post.jwt.
3. Wallet returns an encrypted Authorization response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the Encryption the wallet used is an unsigned, encrypted JWT [RFC7519]. 

