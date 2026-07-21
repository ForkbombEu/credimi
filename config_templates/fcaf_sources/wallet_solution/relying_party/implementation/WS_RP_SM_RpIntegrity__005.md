# WS_RP_SM_RpIntegrity_005

## Objective
Verify that when the Wallet is configured to validate signatures on Request Objects received via the DC API and receives a Request Object with an invalid signature, it rejects the request.

## References
[OIDF.OID4VP] section 5.9.3

## Profile applicability
DC API

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction via the DC API.
2. Wallet receives a Request Object via the DC API carrying an invalid signature.
3. Wallet parses the Request Object.
4. Wallet attempts to validate the signature.

## Expected results
1. Wallet-verifier interaction via the DC API is successfully initiated.
2. Wallet successfully receives the Request Object.
3. Wallet successfully parses the Request Object.
4. Wallet rejects the Authorization Request due to signature validation failure; presentation flow is not initiated.

