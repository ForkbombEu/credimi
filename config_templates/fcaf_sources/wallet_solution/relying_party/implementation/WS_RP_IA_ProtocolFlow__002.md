# WS_RP_IA_ProtocolFlow_002

## Objective
Verify that, in the presentation flow via Redirects, the Wallet supports receiving a Signed Authorization Request using JWT-Secured Authorization Request (JAR) [RFC9101] with the `request_uri` parameter. 

## References
[OIDF.HAIP] section 5.1
[RFC9101] section 5

## Profile applicability
Presentations via Redirects

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
A) Signature from authorization request is authentic.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends Signed Authorization Request using JWT-Secured Authorization Request (JAR) [RFC9101] with the `request_uri` parameter, and signature is authentic.
3. Verify if the Wallet obtains the request object from the `request_uri`.
4. Wallet checks signature on the request object.
5. Verify if Wallet asks for user consent to present the Credential.
6. User gives consent.
7. Verify if Wallet answers successfully (presents the Credential) without rejecting transaction.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet obtains the request object from the `request_uri`.
4. Wallet authenticates successfully request object signature. 
5. Wallet asks for user consent.
6. This is the case.
7. Wallet answers successfully (presents the Credential) without rejecting transaction.
