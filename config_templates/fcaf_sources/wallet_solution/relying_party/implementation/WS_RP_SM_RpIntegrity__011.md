# WS_RP_SM_RpIntegrity_011

## Objective
Verify that when the Wallet receives a Request Object whose Verifier attestation JWT has an iss claim NOT listed among the Wallet's trusted issuers, the Wallet refuses the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object with a Verifier attestation JWT whose iss is NOT in the Wallet's trusted issuer list.
3. Wallet parses the Request Object and the attestation JWT.
4. Wallet checks the iss claim against its trusted issuer list.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the attestation JWT.
4. Wallet refuses the request and returns an invalid_client error due to untrusted attestation issuer; presentation flow is not initiated.

