# WS_RP_MS_Metadata_109

## Objective

Verify that when the Wallet receives an Authorization Request where some Verifier metadata is provided outside the client_metadata parameter, the Wallet ignores such metadata and processes only what is contained within client_metadata.

## References

- [OpenID4VP] Sections 5.9.3, 5.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request where some Verifier metadata is provided outside the client_metadata parameter.
3. Wallet parses the Authorization Request.
4. Wallet extracts Verifier metadata from the Authorization Request.

## Expected results

1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet ignores Verifier metadata provided outside the client_metadata parameter and processes only the metadata within client_metadata; flow proceeds accordingly.
