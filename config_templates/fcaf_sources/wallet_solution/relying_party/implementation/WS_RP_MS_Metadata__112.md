# WS_RP_MS_Metadata_112

## Objective
Verify that when the Wallet receives an Authorization Request using the openid_federation Client Identifier Prefix containing a valid trust_chain parameter, the Wallet processes the trust_chain and uses it for Verifier trust establishment.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the openid_federation: prefix containing a valid trust_chain parameter.
3. Wallet parses the Authorization Request.
4. Wallet processes the trust_chain parameter per OpenID Federation rules.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet successfully validates the trust_chain, establishes Verifier trust, and proceeds with the presentation flow.

