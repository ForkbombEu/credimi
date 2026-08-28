# WS_RP_MS_CredentialFormats_037

## Objective
Verify that Wallet supports IETF SD-JWT VC and ISO mdoc in the presentation flow. 

## References
[OIDF.HAIP] section 5 (introduction)
[OIDF.HAIP] section 5.3.2
[OIDF.HAIP] section 6
[OIDF.OID4VP] appendix B.3

## Profile applicability
Wallet supports both ISO mdoc and IETF SD-JWT VC

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in ISO mdoc format.
3. Verify if Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Verify if Wallet presents Credential to the Verifier.
6. Check value from Credential Format Identifier.
7. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
8. Verifier sends presentation request for Credential in IETF SD-JWT VC format. 
9. Verify if Wallet asks for user consent to present the Credential.
10. User gives consent.
11. Verify if Wallet presents Credential to the Verifier.
12. Check value from Credential Format Identifier.

## Expected results
1. This is the case
2. This is the case.
3. Wallet asks for user consent.
4. This is the case.
6. Wallet presents Credential to the Verifier successfully.
6. Credential presented has Credential Format Identifier value set to `mso_mdoc`.
7. This is the case.
8. This is the case.
9. Wallet asks for user consent.
10. This is the case.
11. Wallet presents Credential to the Verifier successfully.
12. Credential presented has Credential Format Identifier value set to `dc+sd-jwt`.

