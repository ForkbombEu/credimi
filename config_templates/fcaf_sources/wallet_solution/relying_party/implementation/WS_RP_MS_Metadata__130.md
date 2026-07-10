# WS_RP_MS_Metadata_130

## Objective
Verify that when the Wallet receives a Request Object using the x509_hash Client Identifier Prefix where the Client Identifier does NOT match the base64url-encoded SHA-256 hash of the leaf certificate in x5c, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using x509_hash: prefix where the Client Identifier does NOT match the SHA-256 hash of the leaf certificate.
3. Wallet parses the Request Object and x5c.
4. Wallet computes and compares the leaf certificate hash.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and x5c.
4. Wallet rejects the Request Object and returns an invalid_request error due to Client Identifier / certificate hash mismatch; presentation flow is not initiated.

