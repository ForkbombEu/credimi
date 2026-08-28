# WS_RP_MS_ProtocolMessages_051

## Objective
Verify that when the Wallet receives an Authorization Request where the client_id in the query parameter and the Request Object client_id claim differ, the Wallet terminates request processing.

## References
[OIDF.OID4VP] sections 8.5, 5.10.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request where client_id in the query and client_id in the Request Object differ.
3. Wallet parses the query string and the Request Object.
4. Wallet compares the two client_id values.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses both sources.
4. Wallet terminates processing and returns an invalid_request error due to client_id mismatch; presentation flow is not initiated.

