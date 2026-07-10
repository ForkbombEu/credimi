# WS_RP_MS_ProtocolMessages_025

## Objective
Verify that the Wallet returns an invalid_transaction_data error when the type field inside a transaction_data object is not a string.

## References
[OIDF.OID4VP] sections 5.1, 8.3, 8.4

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Applicable only if tested in combination with a Verifier that sends a transaction_data array.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends an Authorization Request containing a transaction_data array where at least one entry has a type field set to a non-string value (e.g. an integer, boolean, or object).

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet returns an invalid_transaction_data error; presentation flow is not initiated.

