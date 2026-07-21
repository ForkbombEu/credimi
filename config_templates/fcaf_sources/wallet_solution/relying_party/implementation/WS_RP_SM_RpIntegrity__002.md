# WS_RP_SM_RpIntegrity_002

## Objective
Verify that when the Wallet receives a signed Request Object via the DC API with an invalid signature and is configured to validate signatures, it rejects the request.

## References
[OIDF.OID4VP] Section 5.3, 5.9.3, Appendix A

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction via the DC API.
2. Wallet receives a Request Object via the DC API carrying an invalid signature.
3. Wallet attempts to validate the signature.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet processes the Request Object.
3. Wallet rejects the Authorization Request due to signature validation failure; presentation flow is not initiated.

