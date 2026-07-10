# WS_RP_IA_Metadata_008

## Objective
Verify that when the Wallet receives an Authorization Request with response_mode = "direct_post.jwt" and a valid HTTPS response_uri, it sends the Authorization Response (vp_token) to the response_uri via an HTTP POST request using HTTPS

## References
[OIDF.OID4VP] section 5.2; [OIDF.OID4VP] section 8.2; [OIDF.OID4VP] section 8.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request with response_mode = 'direct_post.jwt' and a valid HTTPS response_uri.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet sends the vp_token Authorization Response to the response_uri via an HTTP POST request using HTTPS; response body is encoded as application/x-www-form-urlencoded.