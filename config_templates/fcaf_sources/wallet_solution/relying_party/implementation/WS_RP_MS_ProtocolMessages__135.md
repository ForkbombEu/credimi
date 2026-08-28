# WS_RP_MS_ProtocolMessages_135

## Objective
Verify that when transaction_data is provided in the request, the Wallet includes a reference to it in the credential presentation, to ensure transaction integrity.

## References
[OID4VP Section 8.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request containing a transaction_data object.
3. Wallet displays the transaction data to the user, then processes request upon user approval.
4. Verify the JWT payload contains reference to the transaction_data.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet displays the transaction_data to the user and processes the request upon user approval.
4. The wallet submits a presentation whose JWT payload contains a transaction_data_hashes parameter referencing the transaction_data from the request; the verifier successfully identifies and validates the reference.

