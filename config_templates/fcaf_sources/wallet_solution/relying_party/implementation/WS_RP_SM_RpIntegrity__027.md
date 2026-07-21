# WS_RP_SM_RpIntegrity_027

## Objective
Verify that, in the presentation flow via Redirects, Wallet rejects presentation request if the signature validation of the JWT-Secured Authorization Request fails. 

## References
[OIDF.HAIP] section 5.1
[RFC9101] section 5

## Profile applicability
Presentations via Redirects

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions
A Signature from authorization request is authentic.

## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends Signed Authorization Request using JWT-Secured Authorization Request (JAR) [RFC9101] with the `request_uri` parameter, and signature is authentic.
3. Verify if the Wallet obtains the request object from the `request_uri`.
4. Wallet checks signature on the request object.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet obtains the request object from the `request_uri`.
4. Signature validation fails and Wallet either:
a. answers with an error with details (`invalid_request_object`), 
b. answers with an error without providing details or,
c. discontinues the interaction.

