# WS_RP_SM_RpIntegrity_022

## Objective
Verify that when the Wallet receives a Request Object via the DC API where the audience is the origin value prefixed by origin:, all non-key Verifier metadata is taken from client_metadata, and the Wallet processes the request within the DC API context.

## References
[OIDF.OID4VP] section 5.9.3, Appendix A.2.

## Profile applicability
DC API

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives a Request Object via DC API where audience uses the origin: prefix and all non-key Verifier metadata is in client_metadata.
3. Wallet parses the Request Object.
4. Wallet processes the audience as origin:<origin> per DC API rules.
5. Wallet uses client_metadata for non-key Verifier metadata.

## Expected results
1. Wallet-verifier interaction via the DC API is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet processes the audience correctly per DC API rules.
5. Wallet uses client_metadata for all non-key Verifier metadata; presentation flow proceeds within the DC API context.

