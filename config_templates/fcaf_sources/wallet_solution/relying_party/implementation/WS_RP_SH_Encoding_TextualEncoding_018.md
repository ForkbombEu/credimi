# WS_RP_SH_Encoding_TextualEncoding018

## Objective
Test that the claims path pointer does not contain exactly two components then abort processing and return an error.

## References
[OID4VP 7]

## Profile applicability
claims path pointer when applied to a credential in ISO mdoc

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer with a wrong number of components for an mdoc (e.g. path: ["org.iso.18013.5.1"]).
3. Wallet parses the Authorization Request and validates the structure of the claims path pointer.
4. Wallet detects the wrong number of components.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses and validates the path pointer structure.
4. Wallet aborts processing and returns an error (e.g. invalid_request) due to the path not containing exactly two components; presentation flow is not initiated.

