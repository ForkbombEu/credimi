# WS_RP_SH_Encoding_TextualEncoding019

## Objective
Test that the claims path pointer is not a string then abort processing and return an error.

## References
[OID4VP 7]

## Profile applicability
claims path pointer when applied to a credential in ISO mdoc

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where a component is not a string (e.g. path: ["org.iso.18013.5.1", 123]).
3. Wallet parses the Authorization Request and validates the structure of the claims path pointer.
4. Wallet detects a non-string component.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses and validates the path pointer structure.
4. Wallet aborts processing and returns an error (e.g. invalid_request) due to a non-string component in the mdoc claims path pointer; presentation flow is not initiated.

