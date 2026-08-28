# WS_RP_MS_ProtocolMessages_026

## Objective
Verify that the Wallet returns an invalid_transaction_data error when the credential_ids field inside a transaction_data object is not a non-empty array of strings.

## References
[OIDF.OID4VP] sections 5.1, 8.5;

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Applicable only if tested in combination with a specific transaction data type.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends an Authorization Request containing a transaction_data object where credential_ids is set to a non-array value (e.g. a string, integer, or object).

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet returns an invalid_transaction_data error; presentation flow is not initiated.

