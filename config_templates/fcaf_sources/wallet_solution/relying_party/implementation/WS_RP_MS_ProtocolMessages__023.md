# WS_RP_MS_ProtocolMessages_023

## Objective
Verify that when the Authorization Request contains a request_uri parameter and request_uri_method = "post", a Wallet that supports POST retrieves the Request Object using HTTP POST as defined in OID4VP Section 5.10.

## References
[OIDF.OID4VP] section 5.1; [OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing request_uri and request_uri_method = "post". 
3. Wallet retrieves the Request Object.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Request parsed successfully. 
3. Wallet sends an HTTP POST to the request_uri, retrieves the Request Object, and processes the Authorization Request successfully.

