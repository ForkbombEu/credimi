# WS_RP_MS_Metadata_129

## Objective
Verify that when the Wallet receives a Request Object using the x509_hash Client Identifier Prefix where the original Client Identifier equals the base64url-encoded SHA-256 hash of the DER-encoded leaf certificate provided in x5c, the Wallet accepts the binding.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using x509_hash: prefix where the Client Identifier equals the base64url-encoded SHA-256 hash of the encoded leaf certificate in x5c.
3. Wallet parses the Request Object and the x5c chain.
4. Wallet computes the base64url-encoded SHA-256 hash of the leaf certificate and compares against the Client Identifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the x5c chain.
4. Wallet verifies that the Client Identifier matches the leaf certificate hash; request is accepted; presentation flow proceeds.

