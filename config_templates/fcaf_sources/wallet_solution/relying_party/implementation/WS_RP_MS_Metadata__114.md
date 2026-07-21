# WS_RP_MS_Metadata_114

## Objective
Verify that when the Wallet receives an Authorization Request using the openid_federation Client Identifier Prefix and client_metadata is present, the Wallet ignores the client_metadata parameter and resolves Verifier metadata via OpenID Federation.

## References
[OID4VP 5.9.3]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_undefined

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the openid_federation: prefix with a client_metadata parameter included.
3. Wallet parses the Authorization Request.
4. Wallet resolves Verifier metadata via OpenID Federation.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet ignores the client_metadata parameter and obtains all Verifier metadata via OpenID Federation; presentation flow proceeds.

