# WS_RP_SH_Encoding_TextualEncoding013

## Objective
Test that at Step 3 of claims path processing, if the set of elements currently selected is empty, abort processing and return an error. 

## References
[OID4VP 7] Section 7.2.1, 7.3

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
{
  "address": {
    "street": {
      "road": {
        "house": "blue"
      }
    }
  }
}

## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query whose claims path pointer resolves to an empty set against the matching Credential (e.g. claims path pointer: ["address", "postal_code"] where the Credential has no postal_code field under address).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer against the matching JSON-based Credential.
5. Wallet observes that the resulting set of selected JSON elements is empty.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet evaluates the path pointer.
5. Wallet aborts processing and returns an error (e.g. invalid_request) because the set of selected JSON elements is empty; presentation flow is not initiated.

