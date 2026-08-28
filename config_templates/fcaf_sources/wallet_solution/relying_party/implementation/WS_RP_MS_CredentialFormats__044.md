# WS_RP_MS_CredentialFormats_049

## Objective
Verify that when the Wallet presentation to the Verifier includes `status` claim, it contains `status_list` as defined in [I-D.ietf-oauth-status-list].

## References
[OIDF.HAIP] section 6.1
[RFC7800]
[I-D.ietf-oauth-sd-jwt-vc]

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in IETF SD-JWT VC format.
3. Verify if Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Verify if Wallet presents Credential in IETF SD-JWT VC format.
6. Verify if `status` claim is present.
7. Check `status` claim content.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet asks for user consent.
4. This is the case.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. Status` claim is present.
7. "Status" claim contains "status_list".

