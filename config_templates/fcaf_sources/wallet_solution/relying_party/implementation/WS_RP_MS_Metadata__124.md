# WS_RP_MS_Metadata_124

## Objective
Verify that when the Wallet receives a Request Object using the verifier_attestation Client Identifier Prefix where all non-key Verifier metadata is provided within client_metadata, the Wallet uses client_metadata as the source of all non-key metadata and processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix; all non-key Verifier metadata is included in client_metadata.
3. Wallet parses the Request Object.
4. Wallet extracts non-key Verifier metadata from client_metadata.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet obtains all Verifier metadata (except public key) exclusively from client_metadata; presentation flow proceeds.

