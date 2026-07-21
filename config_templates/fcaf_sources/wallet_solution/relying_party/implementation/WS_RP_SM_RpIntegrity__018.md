# WS_RP_SM_RpIntegrity_018

## Objective
Verify that when the Wallet receives a Request Object using the x509_hash Client Identifier Prefix with a valid signature and a complete X.509 trust chain leading to a trusted root, the Wallet validates both the signature and the chain.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using x509_hash: prefix where the signature is valid and the certificate chain in x5c is complete and leads to a trusted root.
3. Wallet parses the Request Object and x5c.
4. Wallet validates the X.509 trust chain.
5. Wallet validates the Request Object signature.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and x5c.
4. Wallet successfully validates the X.509 trust chain.
5. Wallet successfully validates the signature; request is accepted; presentation flow proceeds.

