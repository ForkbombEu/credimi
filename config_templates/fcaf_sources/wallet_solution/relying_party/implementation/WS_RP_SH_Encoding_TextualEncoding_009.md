# WS_RP_SH_Encoding_TextualEncoding009

## Objective
Test that at step 2.2 of claims path processing if any of the currently selected element(s) is not an array, abort processing and return an error.

## References
[OID4VP 7] Section 7.2.1, 7.3

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
{
  "name": "Arthur Dent",
  "address": {
    "street_address": "42 Market Street",
    "locality": "Milliways",
    "postal_code": "12345"
  },
  "degrees": [
    {
      "type": "Bachelor of Science",
      "university": "University of Betelgeuse"
    },
    {
      "type": "Master of Science",
      "university": "University of Betelgeuse"
    }
  ],
  "nationalities": ["British", "Betelgeusian"]
}
## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where a null component is applied to a currently selected element that is NOT an array (e.g. claims path pointer: ["address", null, "street_address"] where address is a JSON object rather than an array).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer against the matching JSON-based Credential.
5. Wallet detects that the currently selected element is not an array when the null component is applied.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet evaluates the path pointer.
5. Wallet aborts processing and returns an error (e.g. invalid_request) due to the null component being applied to a non-array element; presentation flow is not initiated.

