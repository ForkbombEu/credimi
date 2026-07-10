# WS_RP_SH_Cryptography_CryptographicHash008

## Objective
Verify that Wallet raises an error if Verifier uses any hashing function other than SHA-256 and that is not included in the Verifier Client Metadata.

## References
[OIDF.HAIP] section 8

## Profile applicability
Wallet does not support other hash algorithms besides SHA-256.

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
Verifier Client Metadata does not include any other hash algorithms.
Verifier uses other hashing functions than SHA-256.

## Test Scenario
1. Verify if Wallet raises an error when Verifier uses any hashing function other than SHA-256.

## Expected results
1. Wallet raises an error when Verifier uses any hashing function other than SHA-256.