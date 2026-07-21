# WS_RP_SM_DeviceBinding_003

## Objective
Verify that when the Wallet receives an Authorization Request containing a key-bound Verifier Info attestation whose signature object is missing the nonce or client_id parameter, the Wallet rejects the attestation.

## References
[OIDF.OID4VP] section 5.11.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a key-bound Verifier Info attestation whose signature object is MISSING the nonce or client_id parameter.
3. Wallet parses the Authorization Request and the attestation.
4. Wallet validates the signature object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the attestation.
4. Wallet rejects the attestation and returns an invalid_request error due to missing nonce or client_id in the signature object; presentation flow is not initiated.

