# WS_RP_MS_ProtocolMessages_118

## Objective
Verify the positive flow that the Wallet can accept a claims path pointer that is a non-empty array whose members include the following: strings, nulls & non-negative integers. 

## References
[OID4VP 7]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer that is a non-empty array of strings, nulls & non-negative integers.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet accepts the claims path pointer and resolves the referenced claim successfully.

