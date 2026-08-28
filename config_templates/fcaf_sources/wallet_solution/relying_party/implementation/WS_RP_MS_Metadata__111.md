# WS_RP_MS_Metadata_111

## Objective
Verify that when the Wallet receives a signed Authorization Request using a Client Identifier Prefix that supports signing (e.g. x509_hash, openid_federation, verifier_attestation, decentralized_identifier, x509_san_dns), the Wallet accepts the request and proceeds with the presentation flow.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a signed Authorization Request using a Client Identifier Prefix that supports signing (e.g. x509_hash).
3. Wallet verifies the signature using the keys resolvable through the Client Identifier Prefix.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Authorization Request.
3. Wallet successfully verifies the signature and processes the signed request; presentation flow proceeds.

