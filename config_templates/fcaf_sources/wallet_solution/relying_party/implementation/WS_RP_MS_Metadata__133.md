# WS_RP_MS_Metadata_133

## Objective
Verify that when the Wallet receives an Authorization Request using the reserved origin: Client Identifier Prefix outside a DC API context, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the origin: Client Identifier Prefix outside of a DC API context (e.g. via redirect or request_uri flow).
3. Wallet parses the Authorization Request.
4. Wallet detects the origin: prefix in a non-DC-API context.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet rejects the Authorization Request and returns an invalid_request error (origin: prefix MUST NOT be accepted outside DC API); presentation flow is not initiated.

