# WS_RP_SM_RpIntegrity_006

## Objective
Verify that when the Wallet receives a Request Object signed using the decentralized_identifier Client Identifier Prefix where the signing key is identified by the kid JOSE header and is present in the verificationMethod of the resolved DID Document, the Wallet successfully verifies the signature.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object signed with the DID's private key and using the decentralized_identifier: Client Identifier Prefix; the kid header identifies the signing key.
3. Wallet parses the Request Object.
4. Wallet resolves the DID Document for the Verifier and locates the public key identified by kid in verificationMethod.
5. Wallet verifies the Request Object signature using that public key.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet successfully resolves the DID Document and locates the public key.
5. Wallet successfully verifies the signature using the public key from the DID Document verificationMethod; request is processed.

