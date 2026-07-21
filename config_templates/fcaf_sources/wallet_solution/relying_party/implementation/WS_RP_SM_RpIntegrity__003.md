# WS_RP_SM_RpIntegrity_003

## Objective
Verify that when the Wallet receives a Request Object via the DC API and the Wallet is configured NOT to validate signatures, it processes the request without signature validation in accordance with its configured profile.

## References
[OIDF.OID4VP] Section 5.3, 5.9.3, Appendix A

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction via the DC API.
2. Wallet receives a Request Object via the DC API.
3. Wallet proceeds without signature validation.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet processes the Request Object.
3. Wallet processes the request without signature validation in accordance with its configured profile; presentation flow proceeds.

