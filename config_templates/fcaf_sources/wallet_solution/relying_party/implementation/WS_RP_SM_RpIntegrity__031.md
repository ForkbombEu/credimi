# WS_RP_SM_RpIntegrity_031

## Objective
Verify that Wallet supports ECDSA with P-256 and SHA-256 (JOSE algorithm identifier ES256) for validating signed presentation requests.

## References
[OIDF.HAIP] section 7 

## Profile applicability
JOSE algorithm identifier ES256

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. Verify that Wallet receives signed request using ECDSA with P-256 and SHA-256 (JOSE algorithm identifier ES256) from the Verifier.
2. Verify if Wallet is able to validate signed request from the Verifier that used ECDSA with P-256 and SHA-256 (JOSE algorithm identifier ES256) 

## Expected results
1. Wallet receives signed request using ECDSA with P-256 and SHA-256 (JOSE algorithm identifier ES256) from the Verifier.
2. Wallet is able to validate signed request from the Verifier that used ECDSA with P-256 and SHA-256 (JOSE algorithm identifier ES256) 

