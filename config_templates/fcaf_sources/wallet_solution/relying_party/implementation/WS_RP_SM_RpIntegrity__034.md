# WS_RP_SM_RpIntegrity_034

## Objective
Verify that Wallet supports ECDSA with P-256 and SHA-256 (COSE algorithm identifier -9) for validating signed presentation requests.

## References
[OIDF.HAIP] section 7 

## Profile applicability
COSE algorithm identifier -9 supported

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential.
3. Verify that Wallet receives signed request using ECDSA with P-256 and SHA-256 (COSE algorithm identifier -9) from the Verifier.
4. Verify if Wallet is able to validate signed request from the Verifier that used ECDSA with P-256 and SHA-256 (COSE algorithm identifier -9) 

## Expected results
1. This is the case.
2. This is the case.
3. Wallet receives signed request using ECDSA with P-256 and SHA-256 (COSE algorithm identifier -9) from the Verifier.
4.  Wallet is able to validate signed request from the Verifier that used ECDSA with P-256 and SHA-256 (JOSE algorithm identifier ES256; COSE algorithm identifier -9) 

