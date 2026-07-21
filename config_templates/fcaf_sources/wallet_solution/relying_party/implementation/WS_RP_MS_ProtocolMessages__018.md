# WS_RP_MS_ProtocolMessages_018

## Objective
Verify that the Wallet accepts an Authorization Request that contains the transaction_data parameter when the wallet does support this parameter.

## References
[OIDF.OID4VP] section 5; [RFC6749]

## Profile applicability
Wallet does not support transaction_data

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Authorization Request parameters under test apply to both regular Authorization Requests and Authorization Requests delivered via a Request Object (signed JWT, passed by value or by reference via request_uri).

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request that includes a valid transaction_data parameter to a Wallet that DOES support it.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet processes transaction_data correctly and proceeds with the presentation flow.

