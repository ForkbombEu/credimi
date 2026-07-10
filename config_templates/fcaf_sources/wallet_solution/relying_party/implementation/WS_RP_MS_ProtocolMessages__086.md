# WS_RP_MS_ProtocolMessages_086

## Objective
Test the Wallet rejects DCQL-query with credentials property "claim_sets" present that is not an array.

## References
[OID4VP] Section 6.1, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with a non-array type claim_sets
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet detects an invalid credential "claims_sets" property and returns an invalid_request error

