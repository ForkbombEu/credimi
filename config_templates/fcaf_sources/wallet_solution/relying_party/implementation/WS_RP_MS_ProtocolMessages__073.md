# WS_RP_MS_ProtocolMessages_073

## Objective
Test the credential property "meta" must be used by the wallet to constrain matching of any requested credential.

## References
[OID4VP 6.1]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
 

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with "meta"
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The wallet responds without error with credential matching dependent on the "meta" property.

