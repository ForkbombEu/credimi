# WS_RP_SM_RpIntegrity_001

## Objective
Verify that when the Wallet uses information from a verifier_info attestation and signature validation or binding verification fails, the Wallet rejects the Authorization Request.

## References
[OIDF.OID4VP] Section 5.1 and 5.11

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions
Wallet elected to use verifier_info


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a verifier_info array whose attestation objects include a data member matching the declared format, but with either an invalid signature or incorrect binding to the Verifier/purpose.
3. Wallet chooses to use the information from verifier_info, attempts to validate the signature and check binding.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives and parses the Authorization Request.
3. Wallet rejects the Authorization Request due to signature or binding failure; presentation flow is not initiated.

