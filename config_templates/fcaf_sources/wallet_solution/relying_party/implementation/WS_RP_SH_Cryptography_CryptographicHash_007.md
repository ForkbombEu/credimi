# WS_RP_SH_Cryptography_CryptographicHash007

## Objective
Verify that Wallet does not use other hash algorithm than SHA-256 during presentation if that is not supported by the Verifier.

## References
[OIDF.HAIP] section 8

## Profile applicability
Wallet supports other hash algorithms besides SHA-256.

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
A) Verifier Client Metadata does not include any other hash algorithms. 

## Test Scenario
1. Wallet engages with the Verifier. 
2. Wallet receives the Authorization Request whose client metadata advertises SHA-256 only.
3. Wallet returns the Authorization Response and receives the response acknowledging the presentation.

## Expected results
1. Wallet successfully initiates the interaction and 
2. Wallet receives the Authorization Request advertising SHA-256 only.
3. Wallet sends an Authorization Response whose vp_token holds SHA-256-based digests.