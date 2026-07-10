# WS_RP_SM_SessionEncryption_012

## Objective
Verify that, in the presentation flow via Redirects, Wallet sends encrypted response when Authorization Request includes `response_mode` parameter set to `direct_post.jwt`.

## References
[OIDF.HAIP] section 5.1
[OIDF.OID4VP] section 8.3

## Profile applicability
Presentations via Redirects

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends presentation request with response mode direct_post.jwt.
3. Verify if Wallet asks for user consent to present the Credential.
4. User gives consent.
5. Verify if Wallet sends encrypted response using parameters provided in "client_metadata".
6. Verify if Verifier is able to decrypt response sent by the Wallet.  

## Expected results
1. This is the case.
1. This is the case.
3. Wallet asks for user consent.
4. This is the case.
5. Wallet sends encrypted response using parameters provided in "client_metadata".
6. Verifier is able to decrypt response sent by the Wallet.  

