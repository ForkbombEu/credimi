# WS_RP_MS_ProtocolMessages_154

## Objective
Test the Wallet returns the invalid_transaction_data error message, when one object in the transaction_data structure is an object of a known type but containing an unknown field.

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with verifier.
2. The Verifier sends an Authorization request including a transaction_data array where one object has a supported type but includes an unknown field.
3. Wallet processes request and the transaction_data structure.
4. Test the response returned by the Wallet to the verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Wallet identifies unknown field.
4. Verify the Wallet does NOT proceed to the user consent screen, instead it must return an error response: invalid_transaction_data.

