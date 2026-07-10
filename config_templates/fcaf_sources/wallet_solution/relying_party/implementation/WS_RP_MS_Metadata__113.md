# WS_RP_MS_Metadata_113

## Objective
Verify that when the Wallet receives an Authorization Request using the openid_federation Client Identifier Prefix containing a trust_chain parameter that cannot be validated, the Wallet rejects the request in accordance with its trust policy.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using the openid_federation: prefix containing a trust_chain parameter that cannot be validated (malformed, unresolvable, or not trusted).
3. Wallet parses the Authorization Request.
4. Wallet attempts to validate the trust_chain.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully parses the Authorization Request.
4. Wallet rejects the Authorization Request (trust chain validation failure) and returns an appropriate error per its trust policy; presentation flow is not initiated.

