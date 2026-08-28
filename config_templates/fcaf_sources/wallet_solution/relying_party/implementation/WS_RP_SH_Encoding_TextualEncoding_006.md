# WS_RP_SH_Encoding_TextualEncoding006

## Objective
Test that the Wallet handles DCQL-query for SD-JWT VC credential with a non-matching top-level claim path pointer.

## References
[OID4VP 7] Section 7.1.1

## Profile applicability
claims path pointer when applied to a JSON-based Credential

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet contains an attestation without a specific top-level attribute.

## Test Scenario
1. The Wallet engages with the Verifier
2. A Verifier sends a request With a DCQL-query with a claims for the attestation with `path` with value an array containing the name of the specified top-level attribute
3. Wallet handles request
4. Wallet responds to request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet successfully parses the request.
4. Wallet rejects the request by either:
a. answering with an error with details (`invalid_request`), 
b. answering with an error without providing details or,
c. discontinuing the interaction.

