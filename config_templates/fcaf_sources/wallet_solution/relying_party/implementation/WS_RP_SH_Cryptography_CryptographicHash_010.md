# WS_RP_SH_Cryptography_CryptographicHash010

## Objective
Verify that when the credential to be presented uses a digest hash algorithm other than SHA-256 that the Wallet does not support, the Wallet does not produce a presentation for that credential.

## References
[OIDF.HAIP] section 8

## Profile applicability
Wallet does not support other hash algorithms besides SHA-256.

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
Credential Issuer Metadata does not include any other hash algorithms.
Credential Issuer uses other hashing functions than SHA-256.

## Test Scenario
1. Wallet engages with the Verifier.  
2. Wallet receives the Authorization Request targeting the credential whose digest algorithm is not SHA-256.
3. Wallet processes the Authorization Request.

## Expected results
1. Wallet successfully initiates the interaction.
2. Wallet receives the Authorization Request.
3. Wallet does not send an Authorization Response containing a vp_token for that credential, presentation is terminated. 