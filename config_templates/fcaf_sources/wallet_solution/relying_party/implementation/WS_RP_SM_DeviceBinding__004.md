# WS_RP_SM_DeviceBinding_004

## Objective
Verify that when the Wallet receives a Verifier Info attestation whose proof is valid per the active profile, the Wallet successfully validates the proof and accepts the attestation.

## References
[OIDF.OID4VP] section 5.11.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with a Verifier Info attestation whose proof is valid per the active profile.
3. Wallet parses the Authorization Request and the attestation.
4. Wallet validates the proof per the active profile's rules.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request and the attestation.
4. Wallet validates the proof successfully and accepts the attestation; processing continues.

