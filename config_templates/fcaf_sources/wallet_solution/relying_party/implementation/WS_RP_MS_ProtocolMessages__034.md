# WS_RP_MS_ProtocolMessages_034

## Objective
Verify that when the Wallet receives an Authorization Request under the conditions of Section 5.3 (i.e. require_cryptographic_holder_binding = false) without a state parameter, it rejects the request.

## References
[OIDF.OID4VP] sections 5.3, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction.
2. Wallet receives an Authorization Request under Section 5.3 conditions without a state parameter. 
3. Wallet parses the request.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet successfully parses the Authorization Request.
3. Wallet rejects the Authorization Request and returns an invalid_request error; presentation flow is not initiated.

