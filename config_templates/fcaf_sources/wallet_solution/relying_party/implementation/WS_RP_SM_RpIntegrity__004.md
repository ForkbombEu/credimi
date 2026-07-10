# WS_RP_SM_RpIntegrity_004

## Objective
Verify that when the Wallet is configured to validate signatures on Request Objects received via the DC API and receives a Request Object with a valid signature, it successfully validates the signature and processes the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability
DC API

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction via the DC API.
2. Wallet receives a Request Object via the DC API carrying a valid signature.
3. Wallet parses the Request Object.
4. Wallet validates the signature according to its configured profile and trust anchors.
5. Wallet proceeds with request processing.

## Expected results
1. Wallet-verifier interaction via the DC API is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet successfully validates the signature.
5. Wallet processes the request and the presentation flow proceeds normally.

