# WS_RP_SM_RpIntegrity_016

## Objective
Verify that when the Wallet receives a Request Object using an X.509-based Client Identifier Prefix with a valid signature and a complete X.509 trust chain leading to a trusted root, the Wallet validates both the signature and the chain.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
x5c JOSE header contains the full certificate chain.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using an X.509-based prefix where the x5c chain is complete and leads to a trusted root, and the signature is valid.
3. Wallet parses the Request Object and x5c.
4. Wallet validates the X.509 trust chain.
5. Wallet validates the Request Object signature using the leaf certificate.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and x5c.
4. Wallet successfully validates the X.509 trust chain.
5. Wallet successfully validates the signature; request is processed; presentation flow proceeds.