# WS_RP_MS_Metadata_141

## Objective
Verify that when Wallet is presenting the Credential in IETF SD-JWT VC format, it contains the credential issuer's signing certificate along with a trust chain in the `x5c` JOSE header parameter.

## References
[OIDF.HAIP] section 6.1.1

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
TBD

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in IETF SD-JWT VC format.
2. Verify if Wallet presents the Credential in IETF SD-JWT VC format.
3. Check the presence of credential issuer's signing certificate along with a trust chain in the `x5c` JOSE header parameter.

## Expected results
1. This is the case.
2. Wallet presents the Credential in IETF SD-JWT VC format.
3. Credential presented contains the credential issuer's signing certificate along with a trust chain in the `x5c` JOSE header parameter.

