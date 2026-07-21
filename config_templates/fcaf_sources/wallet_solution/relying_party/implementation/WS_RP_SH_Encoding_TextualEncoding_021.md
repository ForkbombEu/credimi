# WS_RP_SH_Encoding_TextualEncoding021

## Objective
Verify that the wallet can successfully select the data element references by the second component of a claims path pointer.

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
3. Wallet parses the Authorization Request, resolves the namespace, and selects the data element value within it.
4. Wallet CBOR-encodes the value and returns it in the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully selects the first_name data element value ("Alice").
4. Wallet returns an Authorization Response containing "Alice" CBOR-encoded; presentation flow proceeds.

