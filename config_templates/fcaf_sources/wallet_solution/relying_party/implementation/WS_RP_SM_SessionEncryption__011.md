# WS_RP_SM_SessionEncryption_011

## Objective
Verify that - for each authorization request - Wallet uses the correct ephemeral key passed via client metadata in the encryption process. 

## References
[OIDF.HAIP] section 5 (introduction) 
[OIDF.OID4VP] section 8.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request including 2 authorization requests, ephemeral public key `key1` for authorization request 1 and ephemeral public key `key2` for authorization request 2.
3. Verify if Wallet uses the correctly ephemeral public key for each one of the authorization requests in the response encryption process.

## Expected results
1. This is the case.
2. Wallet uses `key1` as input for key agreement included in the encryption process of the response related to "authorization request 1".
3. Wallet uses `key2` as input for key agreement included in the encryption process of the response related to "authorization request 2". 

