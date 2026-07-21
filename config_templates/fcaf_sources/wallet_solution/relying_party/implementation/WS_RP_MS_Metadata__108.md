# WS_RP_MS_Metadata_108

## Objective
Verify that when the Wallet receives an Authorization Request where all Verifier metadata is provided within the client_metadata parameter, the Wallet processes the metadata exclusively from client_metadata.

## References
[OIDF.OID4VP] section 5.9.3; [OIDF.OID4VP] section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request where all Verifier metadata is contained within the client_metadata parameter.
3. Wallet parses the Authorization Request.
4. Wallet extracts Verifier metadata from the client_metadata parameter.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet processes all Verifier metadata exclusively from the client_metadata parameter and proceeds with the flow.

