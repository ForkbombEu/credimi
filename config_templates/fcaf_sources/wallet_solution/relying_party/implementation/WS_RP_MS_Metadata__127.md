# WS_RP_MS_Metadata_127

## Objective
Verify that when the Wallet receives a non-DC-API Request Object using the x509_san_dns Client Identifier Prefix and trust in the Client Identifier is established via the certificate, the Wallet allows the redirect_uri provided the FQDN of the redirect_uri matches the Client Identifier hostname; all non-key Verifier metadata is taken from client_metadata.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
A. Wallet has a configured trust anchor for X.509 certificate chain validation.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a non-DC-API Request Object using x509_san_dns: prefix where the FQDN of redirect_uri matches the Client Identifier hostname; non-key Verifier metadata is in client_metadata; trust is established via the leaf certificate.
3. Wallet parses the Request Object.
4. Wallet validates the FQDN of redirect_uri against the Client Identifier hostname.
5. Wallet extracts non-key Verifier metadata from client_metadata.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet verifies the FQDN match.
5. Wallet uses client_metadata for all non-key Verifier metadata; request is accepted; presentation flow proceeds.

