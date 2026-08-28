# WS_RP_MS_ProtocolMessages_052

## Objective
Test the Wallet checks the DCQL query has a valid property "credentials" which is a non-empty array, containing valid entries (according to [OID4VP 6.1]).

## References
[OID4VP 6]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a valid "credentials" property (as per section 6.1)
3. Wallet can successfully use DCQL query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet responds to request without error

