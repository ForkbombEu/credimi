# WS_RP_MS_ProtocolMessages_120

## Objective
Verify negative case that the Wallet cannot accept a claims path pointer that is an array that contains a Boolean.

## References
[OID4VP] Sections 7, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. Engage Wallet-verifier interaction.
2. Verifier sends an Authorization Request with a DCQL query containing a claims path pointer that includes a boolean.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet rejects the request and returns an invalid_request error; presentation flow is not initiated.

