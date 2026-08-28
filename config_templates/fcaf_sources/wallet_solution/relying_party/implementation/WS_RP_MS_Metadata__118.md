# WS_RP_MS_Metadata_118

## Objective
Verify that when the Wallet receives a Request Object using the verifier_attestation Client Identifier Prefix with the Verifier attestation JWT correctly placed in the jwt JOSE header, the Wallet locates and processes the attestation JWT.

## References
[OIDF.OID4VP] section 5.9.3; [OIDF.OID4VP] section 12

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix where the Verifier attestation JWT, containing a cnf claim with the Verifier's public key, is present in the jwt JOSE header.
3. Wallet parses the Request Object JOSE header.
4. Wallet locates and extracts the Verifier attestation JWT from the jwt header.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Request Object JOSE header.
4. Wallet successfully locates and processes the Verifier attestation JWT from the jwt JOSE header; presentation flow proceeds.