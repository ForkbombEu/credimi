# WS_RP_SM_DeviceBinding_007

## Objective
Verify that whenever Wallet presents a SD-JWT containing a `cnf` claim, `cnf` conforms to definition given in [I-D.ietf-oauth-sd-jwt-vc].

## References
[OIDF.HAIP] section 6.1
[RFC7800]
[I-D.ietf-oauth-sd-jwt-vc]

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier requests Credential presentation in IETF SD-JWT VC format.
3. Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. Check `cnf` claim is used within the SD-JWT component of the SD-JWT VC  
7. Check `cnf`.

## Expected results
1. Wallet-verifier interaction is initiated.
2. Wallet receives the presentation request.
3. Wallet asks for user consent.
4. User consent is registered.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. `cnf` claim is used within the SD-JWT component of the SD-JWT VC  
7. `cnf` complies with definition given in [I-D.ietf-oauth-sd-jwt-vc].