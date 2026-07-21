# WS_RP_MS_ProtocolMessages_035

## Objective
Verify that the Wallet accepts vp_token as a valid response_type value in an Authorization Request and does not reject the request.

## References
[OIDF.OID4VP] section 5.6

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends an Authorization Request with response_type = "vp_token".
3. Wallet parses and validates the Authorization Request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet accepts the response_type value and proceeds with the presentation flow without rejecting the request.

