# WS_RP_MS_Metadata_123

## Objective
Verify that when the Wallet receives a Request Object using the verifier_attestation Client Identifier Prefix where non-key Verifier metadata is provided outside the client_metadata parameter, the Wallet ignores such metadata and processes only what is contained within client_metadata.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix where some non-key Verifier metadata is provided outside the client_metadata parameter.
3. Wallet parses the Request Object.
4. Wallet extracts Verifier metadata from the Request Object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet ignores non-key Verifier metadata provided outside client_metadata and processes only the metadata within client_metadata; presentation flow proceeds.

