# WS_RP_MS_ProtocolMessages_017

## Objective
Verify that the Wallet rejects an Authorization Request that contains the transaction_data parameter when the wallet does not support this parameter.

## References
[OIDF.OID4VP] sections 5.8, 8.5; [RFC6749]

## Profile applicability
Wallet does not support transaction_data

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Authorization Request parameters under test apply to both regular Authorization Requests and Authorization Requests delivered via a Request Object (signed JWT, passed by value or by reference via request_uri).

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request that includes the transaction_data parameter the Wallet does NOT support.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet MUST reject the Authorization Request and return an invalid_transaction_data error; the presentation flow is NOT initiated.

