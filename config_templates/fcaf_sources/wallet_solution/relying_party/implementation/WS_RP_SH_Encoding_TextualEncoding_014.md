# WS_RP_SH_Encoding_TextualEncoding014

## Objective
Verify that when the Wallet receives a DCQL query with a claims path pointer applied to an ISO mdoc Credential, where the path is a two-element array containing a valid namespace and a valid data element identifier, the Wallet selects the corresponding data element and returns it CBOR-encoded in the Authorization Response.

## References
[OID4VP 7] Section 7.2, 7.4

## Profile applicability
claims path pointer when applied to a credential in ISO mdoc

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The wallet has an ISO mdoc credential with data element specified in verifier request

## Test Scenario
1. Engage wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer path: ["org.iso.18013.5.1", "first_name"] applied to an ISO mdoc Credential.
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet evaluates the claims path pointer: locates the namespace, then the data element identifier within it.
5. Wallet selects the matching data element value and CBOR-encodes it.
6. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet correctly resolves the namespace and the data element identifier within it.
5. Wallet selects the data element value (e.g. "Alice" for first_name) and CBOR-encodes it.
6. Wallet returns an Authorization Response containing the selected first_name value CBOR-encoded; presentation flow proceeds.