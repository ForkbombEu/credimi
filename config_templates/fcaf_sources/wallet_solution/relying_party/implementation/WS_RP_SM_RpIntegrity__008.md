# WS_RP_SM_RpIntegrity_008

## Objective
Verify that the Request Object is signed with the private key corresponding to the public key in the attestation cnf claim (proof of possession).

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix; the attestation JWT contains a cnf claim with the Verifier's public key; the Request Object signature is made using the corresponding private key.
3. Wallet parses the Request Object.
4. Wallet extracts the public key from the cnf claim of the attestation JWT.
5. Wallet verifies the Request Object signature using that public key.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet successfully extracts the public key from the cnf claim.
5. Wallet successfully verifies proof of possession; presentation flow proceeds.

