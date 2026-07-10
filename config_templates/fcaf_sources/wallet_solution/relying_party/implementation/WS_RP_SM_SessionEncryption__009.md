# WS_RP_SM_SessionEncryption_009

## Objective
Verify that if both A128GCM and A256GCM are supported for the JWE `enc` (encryption algorithm), Wallet will use A256GCM for the JWE `enc`

## References
[OIDF.HAIP] section 5 (introduction) 
[OIDF.OID4VP] section 8.3
[RFC7516] section 4.1.2

## Profile applicability
Wallet supports A128GCM and A256GCM

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends a presentation request.
3. Verify if Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Verify if Wallet sends an encrypted response.
6. Check value from JWE "enc" (encryption algorithm) header parameter (section 4.1.2 of [RFC7516]) used by the Wallet.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet asks for user consent.
4. This is the case.
5. Wallet sends an encrypted response.
6. JWE "enc" value is set to A256GCM.

