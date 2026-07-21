# WS_RP_MS_Metadata_122

## Objective
Verify that when the Verifier attestation JWT does NOT contain a redirect_uris claim, the Wallet does not enforce a redirect_uri match against the attestation and processes the request normally.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
A. Wallet has a configured list of trusted issuers for Verifier Attestation JWTs.

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix; the attestation JWT contains NO redirect_uris claim.
3. Wallet parses the Request Object and the attestation JWT.
4. Wallet checks for the redirect_uris claim and finds none.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and the attestation JWT.
4. Wallet does not enforce redirect_uri against the attestation; request is processed normally; presentation flow proceeds.

