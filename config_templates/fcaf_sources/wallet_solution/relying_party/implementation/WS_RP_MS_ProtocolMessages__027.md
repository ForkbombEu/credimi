# WS_RP_MS_ProtocolMessages_027

## Objective
Verify that the Wallet returns an invalid_transaction_data error when a string in the credential_ids array inside a transaction_data object does NOT match any id in the DCQL Credential Query.

## References
[OIDF.OID4VP] sections 5.1; 8.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Applicable only if tested in combination with a specific transaction data type.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends an Authorization Request containing a transaction_data object whose credential_ids array includes at least one string that does not match any id in the dcql_query.
3. Wallet parses the transaction_data and cross-references credential_ids against the dcql_query.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives and parses the Authorization Request.
3. Wallet returns an invalid_transaction_data error; presentation flow is not initiated.

