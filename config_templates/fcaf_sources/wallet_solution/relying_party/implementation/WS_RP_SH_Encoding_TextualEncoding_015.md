# WS_RP_SH_Encoding_TextualEncoding015

## Objective
Test that the first element of mdoc claims path pointer refers to a namespace.

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
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer path: ["org.iso.18013.5.1", "first_name"].
3. Wallet parses the Authorization Request and the DCQL query.
4. Wallet interprets the first element of the path as a namespace and looks it up in the mdoc Credential.
5. Wallet selects the data element identified by the second element within that namespace.
6. Wallet generates and returns the Authorization Response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the DCQL query.
4. Wallet correctly resolves org.iso.18013.5.1 as the namespace.
5. Wallet selects the first_name data element and CBOR-encodes its value (e.g. "Alice").
6. Wallet returns an Authorization Response containing the first_name value CBOR-encoded; presentation flow proceeds.

