# WS_RP_MS_ProtocolMessages_115

## Objective
The claims property "path" MUST be an array.

## References
[OID4VP] Sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The verifier sends a DCQL query containing a "claims" object with "path" property whereby it is not an array.
3. Wallet handles Query.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet returns an invalid_request error.