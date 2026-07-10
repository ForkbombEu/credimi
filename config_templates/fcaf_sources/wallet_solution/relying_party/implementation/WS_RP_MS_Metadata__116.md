# WS_RP_MS_Metadata_116

## Objective
Verify that when the Wallet receives an Authorization Request using the verifier_attestation Client Identifier Prefix where the original Client Identifier (part after the prefix) equals the subclaim of the Verifier attestation JWT, the Wallet processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the verifier_attestation: prefix where the original Client Identifier matches the subclaim of the attestation JWT.
3. Wallet parses the Authorization Request.
4. Wallet validates that the original Client Identifier equals the subclaim of the Verifier attestation JWT.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet verifies the Client Identifier matches the subclaim; request is processed; presentation flow proceeds.

