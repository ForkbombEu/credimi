# WS_RP_MS_CredentialFormats_039

## Objective
Verify that when the Verifier uses the `vp_token id_token` response type, the Wallet responds with an error (detailed or not) or discontinues the transaction.

## References
[OIDF.HAIP] section 5 (introduction)

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic, EUDI_required

## Preconditions


## Test Scenario
1. End-user interacts with the Verifier, triggering the Verifier to send a presentation request.
2. Verifier sends Authorization Request with response type `vp_token id_token`
3. Wallet processes the request.

## Expected results
1. This is the case.
2. This is the case.
3. Wallet either:
a. answers with an error with details (`unsupported_response_type`), 
b. answers with an error without providing details or,
c. discontinues the interaction.

