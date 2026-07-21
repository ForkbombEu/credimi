# WS_RP_MS_Metadata_119

## Objective
Verify that when the Wallet receives a Request Object using the verifier_attestation Client Identifier Prefix where the jwt JOSE header is absent or does not contain the Verifier attestation JWT, the Wallet rejects the request.

## References
[OID4VP 5.9.3]; [OID4VP 12]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using verifier_attestation: prefix where the jwt JOSE header is absent.
3. Wallet parses the Request Object JOSE header.
4. Wallet attempts to locate the Verifier attestation JWT.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object JOSE header.
4. Wallet rejects the Authorization Request and returns an invalid_request error due to missing jwt JOSE header / Verifier attestation JWT; presentation flow is not initiated.