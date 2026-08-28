# WS_RP_SH_Encoding_TextualEncoding008

## Objective
Test the following condition when processing claims path pointer array is at step 2.1: if the key does not exist in element currently selected, remove that element from the selection.

## References
[OID4VP] Section 7.2.1, 7.3

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
      "university": "University of Betelgeuse"
    }
  ],
  "nationalities": ["British", "Betelgeusian"]
}
## Test Scenario
1. Engage wallet-verifier interaction. 
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where a string component targets a key that is absent from one or more currently selected element(s) (e.g. claims path pointer: ["degrees", null, "type"] where one entry of degrees has no type field).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer against the matching JSON-based Credential.
5. Wallet identifies element(s) where the targeted key is absent and removes them from the selection.
6. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet evaluates the path pointer.
5. Wallet removes element(s) lacking the targeted key from the selection without aborting (e.g. a degrees entry without a type field is dropped, while "Bachelor of Science" is retained).
6. Wallet returns an Authorization Response containing only the retained selected values; presentation flow proceeds.

