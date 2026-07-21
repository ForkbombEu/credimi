# WS_RP_SH_Encoding_TextualEncoding004

## Objective
Test that the claims path pointer array is processed left to right

## References
[OID4VP 7] Section 7.1.1

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
1. The Wallet engages with the Verifier
2. A Verifier sends a request With a DCQL-query with `path` array: ["street_address", "address"].
3. Wallet handles request
4. Wallet responds to request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet successfully parses the request.
4. Verify the wallet responds with an error

