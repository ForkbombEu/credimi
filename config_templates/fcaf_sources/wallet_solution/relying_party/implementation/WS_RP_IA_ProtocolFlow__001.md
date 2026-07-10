# WS_RP_IA_ProtocolFlow_001

## Objective
Verify that when the Wallet receives a Request Object via the Request URI flow, it extracts the set of Authorization Request parameters from the Request Object and uses them for further processing.

## References
[OIDF.OID4VP] section 5.10.1

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post.
3. Wallet sends a POST request and receives a Request Object containing all required Authorization Request parameters.
4. The Wallet processes the request and sends an Authorization Response to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully receives the Request Object.
4. The Authorization Response sent by the Wallet is consistent with the parameter values from the Request Object.