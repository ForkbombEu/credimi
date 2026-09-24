# WS_RP_MS_ProtocolMessages_012

## Objective

Verify that the Wallet successfully evaluates a valid dcql_query that matches exactly one credential and proceeds with the presentation flow.

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
2. Wallet receives an Authorization Request containing a valid dcql_query that matches exactly one credential held by the Wallet.
3. Wallet evaluates the DCQL query and identifies the matching credential.

## Expected results

1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives and parses the Authorization Request successfully.
3. Wallet selects the single matching credential and proceeds with the presentation flow.
