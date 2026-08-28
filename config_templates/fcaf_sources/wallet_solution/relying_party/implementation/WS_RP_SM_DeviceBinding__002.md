# WS_RP_SM_DeviceBinding_002

## Objective
Verify that when the Wallet receives an Authorization Request containing a key-bound Verifier Info attestation whose signature object correctly includes both the nonce and client_id request parameters, the Wallet validates the signature and accepts the attestation.

## References
[OIDF.OID4VP] section 5.11.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a key-bound Verifier Info attestation whose signature object includes both the nonce and client_id parameters.
3. Wallet parses the Authorization Request and the attestation.
4. Wallet validates the signature object using the attestation's bound key.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the attestation.
4. Wallet validates the signature object successfully and accepts the attestation; processing continues.

