# WS_RP_MS_ProtocolMessages_156

## Objective
Test the Wallet returns the invalid_transaction_data error message, when one object in the transaction_data structure contains fields with invalid values for the transaction data type.

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Wallet has a defined range for a known transaction data type.

## Test Scenario
1. The Wallet engages with Verifier.
2. The Verifier sends an Authorization request including a transaction_data array where one object uses a known type, but with a field of value of same type but outside defined range.
3. Wallet processes request and the transaction_data structure.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Wallet identifies value outside acceptable range.
4. Verify the Wallet does NOT proceed to the user consent screen, instead it must return an error response: invalid_transaction_data.

