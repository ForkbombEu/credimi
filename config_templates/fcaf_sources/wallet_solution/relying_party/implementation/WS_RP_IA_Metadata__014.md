# WS_RP_IA_Metadata_014

## Objective
Verify that when the Wallet receives an Authorization Request using the openid_federation Client Identifier Prefix with a resolvable Entity Identifier, the Wallet resolves the Entity via OpenID Federation and processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request with client_id using the openid_federation: prefix containing a resolvable Entity Identifier.
3. Wallet parses the Authorization Request.
4. Wallet recognizes the openid_federation: prefix and resolves the Entity Identifier following OpenID Federation processing rules.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet successfully resolves the Entity via OpenID Federation and processes the request; presentation flow proceeds.

