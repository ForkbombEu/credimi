# WS_RP_SM_RpIntegrity_017

## Objective
Verify that when the Wallet receives a Request Object using an X.509-based Client Identifier Prefix where the X.509 trust chain is incomplete or leads to an untrusted root, the Wallet rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object using an X.509-based prefix where the x5c chain is incomplete or leads to an untrusted root.
3. Wallet parses the Request Object and x5c.
4. Wallet attempts to validate the X.509 trust chain.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object and x5c.
4. Wallet rejects the Request Object and returns an invalid_request error due to failed trust chain validation; presentation flow is not initiated.

