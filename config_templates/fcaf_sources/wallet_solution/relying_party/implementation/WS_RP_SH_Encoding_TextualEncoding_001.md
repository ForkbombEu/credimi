# WS_RP_SH_Encoding_TextualEncoding001

## Objective
Test that in a claims path pointer array, a string value indicates the respected key is to be selected.

## References
[OID4VP 7]

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has a credential:
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
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where one element is a string (e.g. path: ["name"]).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer against the matching JSON-based Credential.
5. Wallet selects the claim value at the key indicated by the string element.
6. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet correctly resolves the path pointer to the targeted key in the Credential.
5. Wallet selects the value associated with that key (e.g. "Arthur Dent" for path: ["name"]).
6. Wallet returns an Authorization Response containing only the selected claim value.

