# WS_RP_MS_ProtocolMessages_057

## Objective
Test when given a valid DCQL query that does not contain the credential_sets property, the Wallet is able to accept it.

## References
[OID4VP 6]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a valid DCQL-query without a "credential_sets" property.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet responds to request without error

