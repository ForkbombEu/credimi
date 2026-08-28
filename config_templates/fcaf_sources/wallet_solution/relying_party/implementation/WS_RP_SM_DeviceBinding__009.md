# WS_RP_SM_DeviceBinding_009

## Objective
Verify that when the Verifier sets require_cryptographic_holder_binding=false, the Wallet still includes the cnf claim within the SD-JWT component of the SD-JWT VC, providing the underlying credential type inherently includes it.

## References
[OIDF.HAIP] section 6.1
[RFC7800]
[I-D.ietf-oauth-sd-jwt-vc]

## Profile applicability
Wallet supports IETF SD-JWT VC

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet holds a valid SD-JWT VC credential that inherently contains a cnf claim.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier requests presentation from a Credential in IETF SD-JWT VC format
3. `require_cryptographic_holder_binding` is set to false in the Credential Query
4. Wallet asks for user consent to present the Credential.
5. User gives consent.
6. Wallet presents Credential in IETF SD-JWT VC format.
7. Check `cnf` claim is present within the SD-JWT component of the response.

## Expected results
1. Wallet-verifier interaction is initiated.
2. Wallet receives the presentation request.
3. Wallet processes the request parameter where holder binding is not mandated by the verifier.
4. Wallet asks for user consent.
5. User consent is registered.
6. Wallet presents the Credential in IETF SD-JWT VC format.
7. Verify that the cnf claim is included within the SD-JWT component of the SD-JWT VC response payload.