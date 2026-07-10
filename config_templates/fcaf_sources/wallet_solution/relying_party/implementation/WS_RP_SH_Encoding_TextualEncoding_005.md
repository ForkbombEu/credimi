# WS_RP_SH_Encoding_TextualEncoding005

## Objective
Test that the Wallet handles DCQL-query for SD-JWT VC credential with a matching top-level claim path pointer.

## References
[OID4VP 7] Section 7.1.1

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet contains an attestation with a specific top-level attribute.

## Test Scenario
1. The Wallet engages with the Verifier
2. A Verifier sends a request With a DCQL-query with a claims for the attestation with `path` with value an array containing the named attribute.
3. Wallet handles request
4. Wallet responds to request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet successfully parses the request.
4. Verify the wallet responds with the credential, including selective disclosure of the named attribute.

