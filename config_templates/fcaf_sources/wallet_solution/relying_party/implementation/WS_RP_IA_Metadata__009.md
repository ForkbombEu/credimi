# WS_RP_IA_Metadata_009

## Objective
Verify that when the Wallet receives an Authorization Request with response_mode = "direct_post.jwt", it sends the Authorization Response to the response_uri as an encrypted JWE over HTTPS.

## References
[OIDF.OID4VP] section 5.2; [OIDF.OID4VP] section 8.2; [OIDF.OID4VP] section 8.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request with response_mode = 'direct_post.jwt' requesting an encrypted response.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet sends an encrypted Authorization Response (JWE) to the response_uri via HTTPS.

