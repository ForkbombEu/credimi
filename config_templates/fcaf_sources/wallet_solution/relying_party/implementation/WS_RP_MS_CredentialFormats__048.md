# WS_RP_MS_CredentialFormats_048

## Objective
Verify that Wallet can handle presenting a Credential using JSON serialization if required by the Wallet profile.

## References
[OIDF.HAIP] section 6.1

## Profile applicability
Wallet supports IETF SD-JWT VC 
Wallet profile supports JSON serialization and Credential profile requires it

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in IETF SD-JWT VC format.
3. Verify if Wallet presents successfully Credential in IETF SD-JWT VC format using JSON serialization.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet presents successfully Credential in IETF SD-JWT VC format using JSON serialization.

