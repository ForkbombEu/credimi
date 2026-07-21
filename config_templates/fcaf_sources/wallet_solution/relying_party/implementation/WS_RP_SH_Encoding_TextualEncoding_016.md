# WS_RP_SH_Encoding_TextualEncoding016

## Objective
Verify that if the first element of the path does not match any namespace in the mdoc, the Wallet aborts and returns an error.

## References
[OID4VP 7] Section 7.2.1 step 2, 7.4

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
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer path: ["org.iso.18013.5.1", "first_name"] whose first element does NOT match any namespace in the Credential.
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet attempts to locate the namespace identified by the first element in the mdoc Credential.
5. Wallet detects that no matching namespace exists.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet attempts to resolve the namespace.
5. Wallet aborts processing and returns an error (e.g. invalid_request) due to the namespace not being present in the mdoc Credential; presentation flow is not initiated.

