# WS_RP_IA_Supportive_007

## Objective
Verify that, in the presentation flow via Redirects, if Verifier sends a Signed Authorization Request, but it does not use a JWT-Secured Authorization Request (JAR) [RFC9101] with the `request_uri` parameter, the Wallet responds with an error (detailed or not) or discontinues the transaction.

## References
[OIDF.HAIP] section 5.1
[RFC9101] section 5

## Profile applicability
Presentations via Redirects

## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request. 
2. In the flow via Redirects, Verifier sends a Signed Authorization Request, but it does not use JWT-Secured Authorization Request (JAR) [RFC9101] with the "request_uri" parameter.
3. Wallet processes the request.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet either:
a. Answers with an error with details (`invalid_request`), 
b. Answers with an error without providing details or,
c. Discontinues the interaction.