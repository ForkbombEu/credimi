# WS_RP_MS_Metadata_106

## Objective
VVerify that the Wallet rejects the Authorization Request when the client_metadata parameter is not a valid UTF-8 encoded JSON object, returning an invalid_request error.

## References
[OIDF.OID4VP] section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends Authorization Request with a client_metadata value that is not a valid UTF-8 encoded JSON object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet rejects the Authorization Request due to invalid encoding or structure and returns an invalid_request error.

