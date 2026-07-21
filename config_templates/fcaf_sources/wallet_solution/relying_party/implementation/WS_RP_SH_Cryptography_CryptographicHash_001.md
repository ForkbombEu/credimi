# WS_RP_SH_Cryptography_CryptographicHash001

## Objective
Verify that when presenting an IETF SD-JWT VC, the Wallet produces a Verifiable Presentation whose digests are generated using SHA-256.

## References
[OIDF.HAIP] section 8

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
Wallet holds a presentable IETF SD-JWT VC.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives Authorization Request. 
3. Wallet returns the Authorization Response. Inspect the vp_token and the Verifier's validation outcome of the SD-JWT VC digests under SHA-256.

## Expected results
1. Wallet-Verifier interaction is successfully initiated.
2. Wallet successfully processes Authorization Request. 
3. Wallet sends an Authorization Response whose vp_token holds the SD-JWT VC with SHA-256-based digests.
