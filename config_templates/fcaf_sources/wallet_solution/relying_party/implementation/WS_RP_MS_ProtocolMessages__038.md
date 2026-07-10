# WS_RP_MS_ProtocolMessages_038

## Objective
Verify that when the Wallet receives an Authorization Request using the redirect_uri Client Identifier Prefix with a valid HTTPS URI, the Wallet recognizes the prefix, uses the embedded URI as the redirect/response target, and processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an unsigned Authorization Request with client_id = 'redirect_uri:https://client.example.org/cb'.
3. Wallet parses the Authorization Request.
4. Wallet recognizes the redirect_uri: prefix and extracts the embedded URI.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet recognizes the redirect_uri: prefix, uses the embedded URI as the redirect/response target, and processes the request successfully.

