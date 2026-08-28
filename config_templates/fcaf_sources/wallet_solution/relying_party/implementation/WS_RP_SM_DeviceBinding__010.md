# WS_RP_SM_DeviceBinding_010

## Objective
Verify that when Wallet uses `cnf` claim within the SD-JWT component of the SD-JWT VC as described in [I-D.ietf-oauth-sd-jwt-vc], the claim contains the confirmation method identifying the proof of possession key.

## References
[OIDF.HAIP] section 6.1
[RFC7800]
[I-D.ietf-oauth-sd-jwt-vc]

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet holds a valid SD-JWT VC credential that supports Cryptographic Key Binding.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier requests presentation from a Credential in IETF SD-JWT VC Credential format that supports and/or requires Cryptographic Key Binding.
3. Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. Check `cnf` claim is used within the SD-JWT 
7. Check `cnf` content.

## Expected results
1. Wallet-verifier interaction is initiated.
2. Wallet receives the presentation request.
3. Wallet asks for user consent.
4. User consent is registered.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. The cnf claim is successfully used within the SD-JWT response payload.
7. Verify the cnf claim contains the confirmation method identifying the proof of possession key as described in [I-D.ietf-oauth-sd-jwt-vc].