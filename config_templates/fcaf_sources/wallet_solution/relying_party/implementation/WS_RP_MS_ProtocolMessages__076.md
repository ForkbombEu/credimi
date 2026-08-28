# WS_RP_MS_ProtocolMessages_076

## Objective
Test the Wallet accepts a DCQL-query with an empty credential property "meta" and handles request without further constraints on the credential.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains a credential in the requested format.

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with the "meta" property present with an empty value and the format matching an available credential.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet responds without error and includes the requested credential

