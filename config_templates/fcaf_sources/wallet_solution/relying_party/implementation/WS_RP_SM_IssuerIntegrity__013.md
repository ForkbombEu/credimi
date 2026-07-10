# WS_RP_SM_IssuerIntegrity_013

## Objective
Verify that Wallet supports X.509 certificate-based key resolution to validate the issuer signature of an SD-JWT VC.

## References
[OIDF.HAIP] section 6.1.1

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request for Credential in IETF SD-JWT VC format.
3. Verify if Wallet presents the Credential in IETF SD-JWT VC format.
4. Check header's content from Credential presented by the Wallet.
 

## Expected results
1. This is the case.
2. Wallet presents the Credential in IETF SD-JWT VC format.
3. Wallet passes the headers properly and unmodified around. That is Credential's header Wallet received from the Credential Issuer are passed to the Verifier without any modification.
4. Wallet verifies the header's content from Credential