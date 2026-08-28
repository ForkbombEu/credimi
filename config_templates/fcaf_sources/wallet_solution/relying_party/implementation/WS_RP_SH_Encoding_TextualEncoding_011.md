# WS_RP_SH_Encoding_TextualEncoding011

## Objective
Test that at step 2.3 of claims path processing, if the index does not exist in a selected array, remove that array from the selection. 

## References
[OID4VP 7] Section 7.2.1, 7.3

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
{
  "degrees": [
    ["Bachelor of Science"], 
    ["Master of Science", "Ph.D."]
  ]
}

## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer where a non-negative integer index is applied to selected array(s), and the index is out of range for at least one of those arrays (e.g. path: ["degrees", null, 1] where some degrees entries have only one element).
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer against the matching JSON-based Credential.
5. Wallet identifies array(s) where the requested index is out of range and removes them from the selection.
6. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet evaluates the path pointer.
5. Wallet removes array(s) lacking an element at the requested index from the selection without aborting (e.g. a ["Bachelor of Science"] entry is dropped because index 1 is out of range, while ["Master of Science", "Ph.D."] retains index 1 = "Ph.D.").
6. Wallet returns an Authorization Response containing only the retained selected values; presentation flow proceeds.