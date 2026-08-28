# WS_RP_MS_Metadata_110

## Objective
Verify that when the Wallet receives a signed Authorization Request using the redirect_uri Client Identifier Prefix, it rejects the request since the redirect_uri prefix cannot be used with signed requests.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a signed Authorization Request using the redirect_uri: Client Identifier Prefix.
3. Wallet parses the Authorization Request.
4. Wallet detects the combination of signed request and redirect_uri prefix.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet rejects the Authorization Request (cannot verify signature with no trusted key available for the redirect_uri prefix) and returns an invalid_request error; presentation flow is not initiated.

