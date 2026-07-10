# WS_RP_MS_ProtocolMessages_011

## Objective
Test that wallet continues with JAR when it receives a request_uri_method with the value post but does not support this feature.

## References
[RFC9101]; [OIDF.OID4VP] section 5, 6.4; DCQL specification

## Profile applicability
Wallet does not support POST method

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives Authorization Request with request_uri and request_uri_method = 'post'.
3. Wallet does not support the POST method for request_uri retrieval.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Authorization Request is processed by the Wallet.
3. Wallet falls back to JAR processing [RFC9101] using HTTP GET and continues with the flow.

