# WS_RP_IA_MainInteraction_032

## Objective
Test that the wallet does not return credentials if they do not match the requested constraints.

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has PID saying (over 18: false)

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query, asking for matching credential to wallet where the PID is (over 18: true)
3. Wallet handles query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet received query
3. Wallet sees that constraint is not passed in the matching credential request and hence the credential is not offered. If no others are available wallet states that: I have no credentials to meet your requirement.