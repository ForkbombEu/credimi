# WS_RP_SH_Cryptography_CryptographicHash006

## Objective
Verify that Wallet includes in its metadata information related to other hash algorithms supported, if the Wallet profile allows other hash algorithms

## References
[OIDF.HAIP] section 8

## Profile applicability
Wallet supports other hash algorithms besides SHA-256.

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
There is a defined mechanism for this Wallet profile, through which Verifier receives Wallet Metadata.

## Test Scenario
1. Verify if Wallet includes other hash algorithms supported (besides SHA-256) in its Metadata.

## Expected results
1. Wallet does include other hash algorithms supported (besides SHA-256) in its Metadata.