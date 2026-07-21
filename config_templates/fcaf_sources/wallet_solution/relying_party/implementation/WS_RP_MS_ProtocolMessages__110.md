# WS_RP_MS_ProtocolMessages_110

## Objective
Test the Wallet checks that within a particular claims array, the same ID MUST NOT be present more than once.

## References
[OID4VP] sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The verifier sends an Authorization request containing a "credentials" object which itself contains "claims" with the same "id" occurring twice in the claims array.
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet detects duplicate claims "id" values in the claims array and returns an invalid_request error

