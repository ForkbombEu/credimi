# WS_RP_SM_RpIntegrity_025

## Objective
Verify that if X.509 certificate of the trust anchor is included in the `x5c` JOSE header of the signed request, the Wallet responds with an error (detailed or not) or discontinues the transaction.

## References
[OIDF.HAIP] section 5 (introduction)

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends signed presentation request.
2. Verifier includes X.509 certificate of the trust anchor in the `x5c` JOSE header of the signed request.
3. Wallet processes the request.

## Expected results
1. This is the case.
2. This is the case.
3. This is the case.
4. Wallet either:
a. answers with an error with details (`invalid_client`), 
b. answers with an error without providing details or,
c. discontinues the interaction.

