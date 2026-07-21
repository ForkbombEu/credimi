# WS_RP_SM_IssuerIntegrity_014

## Objective
Verify that when Wallet is presenting the Credential in IETF SD-JWT VC format, the X.509 certificate of the trust anchor is not included in the `x5c` JOSE header of the SD-JWT VC. 

## References
[OIDF.HAIP] section 6.1.1

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Wallet receives a presentation request for a Credential in IETF SD-JWT VC format.
3. Wallet presents the Credential in IETF SD-JWT VC format.

## Expected results
1. Wallet and verifier interact.
2. Wallet processes the presentation request for a Credential in IETF SD-JWT VC format.
3. The presentation response sent by the Wallet contains the Credential whose `x5c` JOSE header includes the credential issuer's signing certificate along with a trust chain, but does not include the X.509 certificate of the trust anchor.