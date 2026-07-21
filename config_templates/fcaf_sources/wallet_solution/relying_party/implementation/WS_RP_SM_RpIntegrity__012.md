# WS_RP_SM_RpIntegrity_012

## Objective
Verify that when the Wallet receives a Request Object whose Verifier attestation JWT signature is invalid, the Wallet refuses the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object with a Verifier attestation JWT whose signature is invalid.
3. Wallet parses the Request Object and the attestation JWT.
4. Wallet attempts to validate the attestation JWT signature.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the attestation JWT.
4. Wallet refuses the request and returns an invalid_client error due to invalid attestation JWT signature; presentation flow is not initiated.

