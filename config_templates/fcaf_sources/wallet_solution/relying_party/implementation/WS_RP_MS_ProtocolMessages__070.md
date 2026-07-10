# WS_RP_MS_ProtocolMessages_070

## Objective
Test the Wallet handles DCQL-query when credentials "multiple" property is omitted, as having the default value "false".

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains more than one credential matching the request.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential without the "multiple" property.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet responds with only one credential in the response without an error.

