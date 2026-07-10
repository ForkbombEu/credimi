# WS_RP_MS_ProtocolMessages_080

## Objective
Test that if the credential property "trusted_authorities" is not present, it will not invalidate credential

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential without the "trusted_authorities" property
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet is able to respond with credential matching, without an error.

