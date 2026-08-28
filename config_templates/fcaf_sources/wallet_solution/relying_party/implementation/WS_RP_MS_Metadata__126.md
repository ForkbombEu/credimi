# WS_RP_MS_Metadata_126

## Objective
Verify that when the Wallet receives a Request Object using the x509_san_dns Client Identifier Prefix where the DNS name in the Client Identifier does NOT match any dNSName SAN entry in the leaf certificate, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3; [RFC5280]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using x509_san_dns: prefix where the DNS name does NOT match any dNSName SAN entry in the leaf certificate.
3. Wallet parses the Request Object and the x5c chain.
4. Wallet compares the DNS name in the Client Identifier against the SAN entries.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the x5c chain.
4. Wallet rejects the Request Object and returns an invalid_request error due to DNS name / SAN mismatch; presentation flow is not initiated.

