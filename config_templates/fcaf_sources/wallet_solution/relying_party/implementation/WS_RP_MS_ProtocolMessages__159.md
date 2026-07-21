# WS_RP_MS_ProtocolMessages_159

## Objective
Test the Wallet returns the invalid_transaction_data error message, when one object in the transaction_data structure has referenced Credential(s) that are not available in the Wallet.

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with Verifier.
2. The Verifier sends an Authorization request including a transaction_data object that contains credential_ids for a specific credential the Wallet does not have.
3. Wallet processes request and the transaction_data structure.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Wallet identifies that it does not hold credentials of credential_ids in transaction_data object.
4. Verify the Wallet does NOT proceed to the user consent screen, instead it must return an error response: invalid_transaction_data.

