# WS_RP_MS_Metadata_131

## Objective
Verify that when the Wallet receives a Request Object using the x509_hash Client Identifier Prefix signed with the private key corresponding to the public key in the leaf X.509 certificate (whose hash is in the Client Identifier), the Wallet successfully verifies the signature.

## References
[OIDF.OID4VP] section 5.9.3; [RFC7515]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a signed Request Object using x509_hash: prefix where the Client Identifier equals the SHA-256 hash of the leaf certificate and the signature is made with the leaf certificate's private key.
3. Wallet parses the Request Object and x5c.
4. Wallet extracts the leaf certificate's public key.
5. Wallet verifies the Request Object signature using that public key.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and x5c.
4. Wallet successfully extracts the leaf certificate public key.
5. Wallet successfully verifies the signature; request is processed; presentation flow proceeds.

