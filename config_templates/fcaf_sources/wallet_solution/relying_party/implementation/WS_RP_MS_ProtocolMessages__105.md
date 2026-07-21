# WS_RP_MS_ProtocolMessages_105

## Objective
Test that the wallet can handle a DCQL query with a "credentials" object with a "claims" property present.

## References
[OID4VP 6.3]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credentials object with a claims property.
3. Wallet can handle Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet processes the DCQL query, it MUST use "id" and "path", and if present "values"
3.1. Claim properties are used to perform matching and complete matching without error.

