# WS_RP_SM_SessionBinding_002

## Objective
 Test that the Wallet rejects Authorization Requests that do not contain a nonce parameter.

## References
[OIDF.OID4VP] section 5, section 14.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request where the nonce parameter is absent.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet rejects the Authorization Request and returns an invalid_request error; presentation flow is not initiated.

