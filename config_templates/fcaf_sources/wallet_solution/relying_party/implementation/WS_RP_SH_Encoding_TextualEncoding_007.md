# WS_RP_SH_Encoding_TextualEncoding007

## Objective
Test that when processing claims path pointer array, an element at the first level is selected. If the selected element is not an object, abort processing and return an error

## References
[OID4VP] Section 7.2.1, 7.3

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet contains the credentials
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
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where a string element is applied to a value that is NOT a JSON object (e.g. claims path pointer: ["name", "firstname"] where name is a string).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer against the matching JSON-based Credential.
5. Wallet detects that the currently selected element is not an object when the string path element is applied.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet evaluates the path pointer.
5. Wallet aborts processing and returns an error (e.g. invalid_request); presentation flow is not initiated.

