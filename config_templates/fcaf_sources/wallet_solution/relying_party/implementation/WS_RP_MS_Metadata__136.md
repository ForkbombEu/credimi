# WS_RP_MS_Metadata_136

## Objective
Verify that when the Wallet uses a Client Identifier Prefix that permits signed Request Objects, the Wallet lists supported Request Object signing algorithms in the request_object_signing_alg_values_supported parameter of wallet_metadata.

## References
[OIDF.OID4VP] section 5.10

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = post and a Client Identifier Prefix that permits signed Request Objects.
3. Wallet prepares wallet_metadata for the POST request.
4. Wallet sends the POST request to the request_uri endpoint. Inspect wallet_metadata for the request_object_signing_alg_values_supported parameter.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet correctly prepares wallet_metadata.
4. Wallet_metadata includes the request_object_signing_alg_values_supported parameter listing all supported Request Object signing algorithms.

