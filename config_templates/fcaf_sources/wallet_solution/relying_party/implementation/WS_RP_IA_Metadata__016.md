# WS_RP_IA_Metadata_016

## Objective
Verify that when the Wallet receives a Request Object using the decentralized_identifier Client Identifier Prefix with an empty client_metadata parameter, the Wallet rejects the request since non-key Verifier metadata MUST be obtained from client_metadata.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using decentralized_identifier: prefix with an empty client_metadata parameter.
3. Wallet parses the Request Object.
4. Wallet attempts to extract non-key Verifier metadata.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet rejects the Authorization Request and returns an invalid_request error due to empty client_metadata (required for non-key Verifier metadata); presentation flow is not initiated.