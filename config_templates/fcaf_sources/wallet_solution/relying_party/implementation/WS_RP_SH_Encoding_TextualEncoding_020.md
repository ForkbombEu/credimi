# WS_RP_SH_Encoding_TextualEncoding020

## Objective
Test claims path pointer processing by selecting the data element referenced by the second component. If the data element does not exist in the Credential it MUST abort processing and return an error.

## References
[OID4VP 7]

## Profile applicability
claims path pointer when applied to a credential in ISO mdoc

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
{
  "org.iso.18013.5.1": {
    "first_name": "Alice"
  }
}

## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where the second component does not exist within the resolved namespace (e.g. path: ["org.iso.18013.5.1", "Bob"]).
3. Wallet parses the Authorization Request, resolves the namespace, and looks up the data element identifier within it.
4. Wallet detects that the data element is not present.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully resolves the namespace.
4. Wallet aborts processing and returns an error (e.g. invalid_request) because the data element identifier is not present in the namespace; presentation flow is not initiated.

