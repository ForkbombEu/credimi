# WS_RP_SM_RpIntegrity_024

## Objective
Verify that, for signed requests, if Verifier does not use Client Identifier Prefix `x509_hash`, the Wallet responds with an error (detailed or not) or discontinues the transaction.

## References
[OIDF.HAIP] section 5 (introduction)
[OIDF.OID4VP] 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request. 
2. Verifier sends a signed presentation request.
3. Verifier does not use Client Prefix "x509_hash".
4. Wallet processes the request.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet either:
a. answers with an error with details (`invalid_request`), 
b. answers with an error without providing details or,
c. discontinues the interaction.

