# WS_RP_IA_Metadata_015

## Objective
Verify that when the Wallet receives a Request Object using the decentralized_identifier Client Identifier Prefix with a resolvable DID Document and all non-key Verifier metadata present in client_metadata, the Wallet performs DID Resolution for the public key and obtains all other metadata from client_metadata.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using decentralized_identifier: prefix; client_metadata contains all non-key Verifier metadata; DID Document is resolvable.
3. Wallet parses the Request Object.
4. Wallet performs DID Resolution to obtain the public key.
5. Wallet extracts all non-key Verifier metadata from client_metadata.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet successfully resolves the DID Document and obtains the public key.
5. Wallet uses client_metadata for all non-key Verifier metadata; presentation flow proceeds.

