# WS_RP_SM_RpIntegrity_020

## Objective
Verify that when the Wallet receives a Request Object using the x509_hash Client Identifier Prefix with valid signature and trust chain, and all non-key Verifier metadata is provided within the client_metadata parameter, the Wallet uses client_metadata as the source of all non-key metadata.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using x509_hash: prefix where signature is valid, trust chain leads to a trusted root, and all non-key Verifier metadata is in client_metadata.
3. Wallet parses the Request Object.
4. Wallet validates signature and trust chain.
5. Wallet extracts non-key Verifier metadata from client_metadata.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet successfully validates signature and trust chain.
5. Wallet obtains all non-key Verifier metadata from client_metadata; request is accepted; presentation flow proceeds.

