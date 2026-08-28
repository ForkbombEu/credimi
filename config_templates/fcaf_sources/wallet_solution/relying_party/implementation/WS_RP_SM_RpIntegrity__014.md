# WS_RP_SM_RpIntegrity_014

## Objective
Verify that when the Wallet receives a Request Object using an X.509-based Client Identifier Prefix where the x5c JOSE header is missing, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3; [RFC7515]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
A. X.509 certificate is configured 

## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a signed Request Object using an X.509-based prefix where the x5c JOSE header is missing.
3. Wallet parses the Request Object JOSE header.
4. Wallet attempts to extract the leaf certificate public key for signature verification.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object JOSE header.
4. Wallet rejects the Request Object and returns an invalid_request error due to missing x5c header; presentation flow is not initiated.

