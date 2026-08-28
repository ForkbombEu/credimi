# WS_RP_SM_RpIntegrity_010

## Objective
Verify that when the Wallet receives a Request Object whose Verifier attestation JWT has a valid signature and an iss claim identifying a trusted issuer of Verifier attestations, the Wallet establishes trust and processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object with a Verifier attestation JWT whose iss identifies a trusted issuer and whose signature is valid.
3. Wallet parses the Request Object and the attestation JWT.
4. Wallet validates the attestation JWT signature and checks the iss against its trusted issuer list.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the attestation JWT.
4. Wallet successfully validates the signature and trust; request is processed; presentation flow proceeds.

