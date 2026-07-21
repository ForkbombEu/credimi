# WS_RP_SM_SessionEncryption_005

## Objective
Verify that in the encrypted response sent by the Wallet: the JWE `alg` (algorithm) header parameter (as defined in section 4.1.1 of [RFC7516]) value is set to ECDH-ES (as defined in section 4.6 of [RFC7518]), with key agreement utilizing keys on the P-256 curve (as defined in section 6.2.1.1 of [RFC7518]).

## References
[OIDF.HAIP] section 5 (introduction) 
[OIDF.OID4VP] section 8.3
[RFC7516] section 4.1.1
[RFC7518] 6.2.1.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends a presentation request
3. Verify if Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Verify if Wallet sends encrypted response by using utilizing response mode direct_post.jwt.
6. Verify if, in the encrypted response, the JWE alg (algorithm) header parameter value is set ECDH-ES.
7. Verify if key agreement uses keys on the P-256 curve.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet asks for user consent.
4. This is the case.
5. Wallet sends encrypted response.
6. In the encrypted response, JWE alg (algorithm) header parameter value is set ECDH-ES.
7. Key agreement uses keys on the P-256 curve. 

