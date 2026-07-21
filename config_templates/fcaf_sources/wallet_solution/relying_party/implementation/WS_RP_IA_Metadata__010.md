# WS_RP_IA_Metadata_010

## Objective
Verify that when the Wallet receives an Authorization Request with response_mode = "direct_post.jwt" where both response_uri and redirect_uri are present, it rejects the request with an invalid_request error since the two parameters are mutually exclusive.

## References
[OIDF.OID4VP] section 5.2; [OIDF.OID4VP] section 8.2; [OIDF.OID4VP] section 8.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. The Wallet receives an Authorization Request with response_mode = 'direct_post.jwt' where redirect_uri is also present.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet returns an invalid_request error because redirect_uri and response_uri MUST NOT both be present.