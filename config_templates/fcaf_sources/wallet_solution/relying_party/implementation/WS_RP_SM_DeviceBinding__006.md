# WS_RP_SM_DeviceBinding_006

## Objective
Verify that when the Wallet receives a Verifier Info attestation of a type NOT defined or supported by the active profile, the Wallet ignores the unrecognized attachment type and continues processing the rest of the request.

## References
[OIDF.OID4VP] section 5.11.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with a Verifier Info attestation of a type NOT defined by the active profile.
3. Wallet parses the Authorization Request.
4. Wallet inspects the attestation type.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet ignores the unrecognized attestation type; processing continues with remaining valid request elements; presentation flow proceeds.

