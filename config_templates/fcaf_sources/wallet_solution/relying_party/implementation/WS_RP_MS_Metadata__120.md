# WS_RP_MS_Metadata_120

## Objective
Verify that when the Verifier attestation JWT contains a redirect_uris claim and the redirect_uri request parameter exactly matches one of its entries, the Wallet processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
A. Wallet has a configured list of trusted issuers for Verifier Attestation JWTs.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix where the redirect_uri request parameter exactly matches one entry in the attestation JWT's redirect_uris claim.
3. Wallet parses the Request Object and the attestation JWT.
4. Wallet compares the redirect_uri parameter against the redirect_uris claim entries.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the attestation JWT.
4. Wallet verifies the exact match; request is processed; presentation flow proceeds.

