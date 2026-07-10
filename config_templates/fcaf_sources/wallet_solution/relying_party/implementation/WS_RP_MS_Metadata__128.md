# WS_RP_MS_Metadata_128

## Objective
Verify that when the Wallet receives a non-DC-API Request Object using the x509_san_dns Client Identifier Prefix and the FQDN of redirect_uri does NOT match the Client Identifier hostname (and trust in the Client Identifier is not otherwise established), the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
A. Wallet has a configured trust anchor for X.509 certificate chain validation.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a non-DC-API Request Object using x509_san_dns: prefix where the FQDN of redirect_uri does NOT match the Client Identifier hostname.
3. Wallet parses the Request Object.
4. Wallet validates the FQDN of redirect_uri against the Client Identifier hostname.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet rejects the Request Object and returns an invalid_request error due to FQDN / Client Identifier hostname mismatch; presentation flow is not initiated.

