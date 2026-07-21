# WS_RP_SM_RpIntegrity_007

## Objective
Verify that when the Wallet receives a Request Object using the decentralized_identifier Client Identifier Prefix where the signing key is NOT listed in the verificationMethod of the resolved DID Document, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using decentralized_identifier: prefix where the signing key is NOT listed in the resolved DID Document verificationMethod.
3. Wallet parses the Request Object.
4. Wallet resolves the DID Document and attempts to locate the signing key.
5. Wallet attempts signature verification.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet resolves the DID Document; signing key is not found in verificationMethod.
5. Wallet rejects the Request Object and returns an invalid_request error due to signature verification failure; presentation flow is not initiated.

