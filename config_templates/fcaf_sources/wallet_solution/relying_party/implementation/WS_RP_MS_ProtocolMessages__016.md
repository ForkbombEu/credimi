# WS_RP_MS_ProtocolMessages_016

## Objective
Verify that the Wallet ignores unrecognized parameters in the Authorization Request.

## References
[OIDF.OID4VP] section 5; [RFC6749]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The Authorization Request parameters under test apply to both regular Authorization Requests and Authorization Requests delivered via a Request Object (signed JWT, passed by value or by reference via request_uri).

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request containing one or more unrecognized parameters that are NOT transaction_data.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet ignores all unrecognized parameters and proceeds with the presentation flow as if those parameters were absent. No error is returned.

