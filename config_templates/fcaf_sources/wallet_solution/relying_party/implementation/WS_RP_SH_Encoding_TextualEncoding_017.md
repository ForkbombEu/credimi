# WS_RP_SH_Encoding_TextualEncoding017

## Objective
Test that the second element of mdoc claims path pointer refers to a data element identifier.

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
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer path: ["org.iso.18013.5.1", "first_name"].
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet resolves the namespace from the first element.
5. Wallet looks up the data element identifier (second element) within that namespace and selects its value.6. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet correctly resolves the namespace.
5. Wallet correctly resolves the first_name data element identifier and selects its value (e.g. "Alice").
6. Wallet returns an Authorization Response containing the value "Alice" CBOR-encoded; presentation flow proceeds.