# WS_RP_SM_RpIntegrity_023

## Objective
Verify that, for signed requests, Wallet accepts Client Identifier Prefix `x509_hash` as defined in section 5.9.3 of [OIDF.OID4VP].

## References
[OIDF.HAIP] section 5 (introduction)
[OIDF.OID4VP] 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request. 
2. Verifier sends signed presentation request.
3. Verifier uses Client Prefix "x509_hash" correctly.
4. Verify if Wallet asks for user consent to present the Credential.
5. User gives consent.
6. Verify if Wallet presents Credential successfully. 

## Expected results
1. This is the case.  
2. This is the case.
3. This is the case.  
4. Wallet asks for user consent.
5. This is the case.
6. Wallet presents Credential successfully. 

