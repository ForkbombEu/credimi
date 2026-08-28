# WS_RP_MS_Metadata_132

## Objective
Verify that when the Wallet receives a Request Object using the x509_hash Client Identifier Prefix where the signature key does NOT correspond to the leaf certificate in x5c, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3; [RFC7515]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a signed Request Object using x509_hash: prefix where the signature was made with a key NOT corresponding to the leaf certificate in x5c.
3. Wallet parses the Request Object and x5c.
4. Wallet extracts the leaf certificate public key.
5. Wallet attempts signature verification.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and x5c.
4. Wallet successfully extracts the leaf certificate public key.
5. Wallet rejects the Request Object and returns an invalid_request error due to signature verification failure; presentation flow is not initiated.

