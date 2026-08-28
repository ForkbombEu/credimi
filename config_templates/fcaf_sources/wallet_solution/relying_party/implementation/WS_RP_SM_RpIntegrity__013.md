# WS_RP_SM_RpIntegrity_013

## Objective
Verify that when the Wallet receives a Request Object using an X.509-based Client Identifier Prefix (x509_san_dns or x509_hash) signed with the private key corresponding to the public key in the leaf certificate of the x5c JOSE header chain, the Wallet successfully verifies the signature.

## References
[OIDF.OID4VP] section 5.9.3; [RFC7515]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
A. X.509 certificate is configured 

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a signed Request Object using an X.509-based prefix where the signature is made with the leaf certificate's private key and x5c contains the certificate chain.
3. Wallet parses the Request Object and the x5c JOSE header.
4. Wallet extracts the leaf certificate's public key from x5c.
5. Wallet verifies the Request Object signature using the leaf certificate public key.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the x5c JOSE header.
4. Wallet successfully extracts the leaf certificate's public key.
5. Wallet successfully verifies the signature; request is accepted; presentation flow proceeds.

