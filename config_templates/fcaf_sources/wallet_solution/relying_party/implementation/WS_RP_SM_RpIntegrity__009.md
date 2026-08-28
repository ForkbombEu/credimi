# WS_RP_SM_RpIntegrity_009

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
2. Wallet receives a Request Object using verifier_attestation: prefix where the Request Object signature is made with a key that does NOT match the cnf claim public key.
3. Wallet parses the Request Object.
4. Wallet extracts the public key from the cnf claim.
5. Wallet attempts signature verification.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet extracts the cnf claim public key.
5. Wallet rejects the Request Object and returns an invalid_request error due to failed proof of possession; presentation flow is not initiated.

