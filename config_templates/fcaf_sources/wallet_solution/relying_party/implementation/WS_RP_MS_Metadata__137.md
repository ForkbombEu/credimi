# WS_RP_MS_Metadata_137

## Objective
Verify that when the Wallet uses a Client Identifier Prefix that precludes signed Request Objects (e.g. redirect_uri:), the Wallet does NOT include the request_object_signing_alg_values_supported parameter in wallet_metadata.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post using a Client Identifier Prefix that precludes signed Request Objects (e.g. redirect_uri:).
3. Wallet prepares wallet_metadata for the POST request.
4. Wallet sends the POST request. Inspect wallet_metadata.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet correctly prepares wallet_metadata.
4. Wallet_metadata does NOT include the request_object_signing_alg_values_supported parameter.

