# WS_RP_SM_SessionEncryption_010

## Objective
Verify that if Verifier did not supply ephemeral encryption public keys specific to each Authorization Request via client metadata, the Wallet responds with an error (detailed or not) or discontinues the transaction.

## References
[OIDF.HAIP] section 5 (introduction) 
[OIDF.OID4VP] section 8.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends a presentation request and does not supply ephemeral encryption public keys specific to each Authorization Request.
3. Check Wallet behaviour.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet either:
a. answers with an error with details (`invalid_client_metadata`), 
b. answers with an error without providing details or,
c. discontinues the interaction.

