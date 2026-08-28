# WS_RP_MS_ProtocolMessages_111

## Objective
Test the Wallet checks a claims object property "id" must be a non-empty.

## References
[OID4VP] Sections 6.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The verifier sends a DCQL query containing a "claims" object with "id" property whereby it is empty
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet detects malformed "id" and returns an invalid_request error.

