# WS_RP_MS_ProtocolMessages_071

## Objective
If "multiple" is present, the Wallet processes it when determining how many matching Credentials may be returned.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains more than one credential matching the request.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with the "multiple" value: TRUE.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet responds with multiple credentials in the response.

