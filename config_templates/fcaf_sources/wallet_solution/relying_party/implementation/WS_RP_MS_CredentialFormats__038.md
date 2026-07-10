# WS_RP_MS_CredentialFormats_038

## Objective
Verify that Wallet can handle receiving authorization request with response type `vp_token` and answering properly.

## References
[OIDF.HAIP] section 5 (introduction)

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends Authorization Request with response type `vp_token`
3. Verify if Wallet process successfully Authorization Request with response type `vp_token` sent by the Verifier.
4. Verify if Wallet asks for user consent to present the Credential.
5. User gives consent.
6. Verify if Wallet includes `vp_token` parameter in its response.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet process successfully Authorization Request with response type `vp_token` sent by the Verifier.
4. Wallet asks for user consent.
5. This is the case.
6. Wallet includes `vp_token` parameter in its response.

