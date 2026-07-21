# WS_RP_MS_ProtocolMessages_028

## Objective
Verify that the Wallet successfully processes a transaction_data object when every string in the credential_ids array matches an id in the DCQL Credential Query.

## References
[OIDF.OID4VP] section 5.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet supports the transaction_data type; Authorization Request includes a valid dcql_query.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Verifier sends an Authorization Request containing a transaction_data object whose credential_ids array consists of strings all matching an id in the dcql_query.
3. Wallet parses the transaction_data and cross-references credential_ids against the dcql_query.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives and parses the Authorization Request.
3. Wallet validates that all credential_ids reference valid dcql_query ids and proceeds with the presentation flow using only the referenced Credentials for transaction authorization.

