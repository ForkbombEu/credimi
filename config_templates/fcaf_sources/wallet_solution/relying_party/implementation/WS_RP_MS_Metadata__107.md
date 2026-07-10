# WS_RP_MS_Metadata_107

## Objective
 Verify that when the Verifier sets request_uri_method = "post" without client_metadata, the Wallet still proceeds by sending its full supported wallet_metadata in the POST request to the request_uri.

## References
[OIDF.OID4VP] Section 5.10, [OIDF.OID4VP] Section 5.1, [RFC9101]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with request_uri_method = "post" but no client_metadata parameter.
3. Wallet has no information about the Verifier's capabilities.
4. Wallet sends an HTTP POST request to the request_uri including a wallet_metadata parameter.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully parses the Authorization Request.
3. Wallet proceeds without Verifier capability information.
4. Wallet POSTs to the request_uri with wallet_metadata containing its full set of supported capabilities; Wallet retrieves the Request Object and proceeds with the flow.

