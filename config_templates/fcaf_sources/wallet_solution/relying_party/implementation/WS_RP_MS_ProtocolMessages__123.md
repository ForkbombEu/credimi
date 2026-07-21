# WS_RP_MS_ProtocolMessages_123

## Objective
Test that when the Wallet is given an invalid claims path pointer, it will abort processing and return an error.

## References
[OID4VP] Sections 7, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage Wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer with an invalid element (e.g. path: [true]).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet validates the structure of the claims path pointer.
5. Wallet detects an element of an unsupported type.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet validates the path pointer structure.
5. Wallet aborts processing and returns an error (e.g. invalid_request) due to invalid path pointer element type; presentation flow is not initiated.

