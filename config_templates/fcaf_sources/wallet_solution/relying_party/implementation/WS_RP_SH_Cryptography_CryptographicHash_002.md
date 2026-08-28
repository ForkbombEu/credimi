# WS_RP_SH_Cryptography_CryptographicHash002

## Objective
Verify that whenever Wallet uses hash function during presentation for ISO mdoc, it leverages hash algorithm SHA-256.

## References
[OIDF.HAIP] section 9

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. Wallet engages with the Verifier.
2. Wallet receives Authorization Request. 
3. Wallet returns the Authorization Response and receives the response acknowledging the presentation.

## Expected results
1. Wallet successfully engages with the Verifier.
2. Wallet processes Authorization Request. 
3. Wallet sends an Authorization Response whose vp_token holds the ISO mdoc with SHA-256-based digests.