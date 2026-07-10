# WS_RP_SM_DeviceBinding_008

## Objective
Verify that when Verifier requires Cryptographic Key Binding, `cnf` claim is used within the SD-JWT component of the SD-JWT VC as described in [I-D.ietf-oauth-sd-jwt-vc].

## References
[OIDF.HAIP] section 6.1
[RFC7800]
[I-D.ietf-oauth-sd-jwt-vc]

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Credential requested in the test supports Cryptographic Key Binding.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in the IETF SD‑JWT VC format that supports Cryptographic Key Binding.
3. `require_cryptographic_holder_binding` is set to true in the Credential Query.
4. Verify if Wallet asks for user consent to present the Credential.
5. User gives consent.
6. Verify if Wallet presents Credential in IETF SD-JWT VC format.
7. Check `cnf` claim is used within the SD-JWT component of the SD-JWT VC  

## Expected results
1. This is the case.
2. This is the case.
3. This is the case.
4. Wallet asks for user consent. 
5. This is the case.
6. Wallet presents Credential in IETF SD-JWT VC format.
7. `cnf` claim is used within the SD-JWT component of the SD-JWT VC  

