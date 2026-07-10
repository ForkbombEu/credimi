# WS_RP_SM_DeviceBinding_012

## Objective
Verify that if the credential has cryptographic holder binding, Wallet presents a KB-JWT, as defined in [I-D.ietf-oauth-sd-jwt-vc] when presenting SD-JWT VC.

## References
[OIDF.HAIP 6.1.1.1]

## Profile applicability
Wallet supports IETF SD-JWT VC 

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet holds a valid SD-JWT VC credential that mandates Cryptographic Key Binding.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request. 
2. Verifier sends presentation request for Credential in IETF SD‑JWT VC format that has Cryptographic Key Binding.
3. Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. Check if Wallet presents a KB-JWT.

## Expected results
1. Wallet-verifier interaction is initiated.
2. Wallet receives the presentation request.
3. Wallet asks for user consent to present the Credential.
4. User consent is registered.
5. Wallet presents Credential in IETF SD-JWT VC format.
6. Verify that the Wallet successfully includes a structurally valid KB-JWT in the presentation payload, checking it aligns with definition held here: [OIDF.HAIP 6.1.1.1]