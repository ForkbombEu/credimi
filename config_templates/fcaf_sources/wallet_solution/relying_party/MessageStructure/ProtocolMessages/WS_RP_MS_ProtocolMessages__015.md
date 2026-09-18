# WS_RP_MS_ProtocolMessages_015

## Objective

Verify that the Wallet rejects the Authorization Request and returns an error when the dcql_query is malformed.

## References

- [OpenID4VP] Sections 5, 6.4
- [ISO/IEC 18013-7] Annex C

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

1. Wallet contains a credential matching the request.

## Test Scenario

1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request containing a malformed dcql_query.
3. Wallet attempts to parse the dcql_query.

## Expected results

1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives the Authorization Request.
3. Wallet rejects the Authorization Request and returns an invalid_request error; no credential is presented.
