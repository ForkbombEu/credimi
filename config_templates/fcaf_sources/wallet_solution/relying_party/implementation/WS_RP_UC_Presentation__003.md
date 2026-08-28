# WS_RP_UC_Presentation_003

## Objective
Verify that a Wallet supporting OAuth 2.0 scope-based Presentation requests processes an Authorization Request using a valid pre-defined scope value, by presenting a credential of the type the scope value identifies.
## References
[OID4VP 5.5]

## Profile applicability
Wallet supports OAuth 2.0

## EUDI-wallet relevancy
EUDI_generic | EUDI_forbidden

## Preconditions


## Test Scenario
1. Engage wallet-verifier interaction (e.g. click link / scan QR code).
2. Wallet receives an Authorization Request using a valid pre-defined scope value as defined by the applicable profile.
3. The Wallet returns a response to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. The Wallet successfully processes the Authorization Request.
3. The Wallet returns an Authorization Response containing a credential whose type matches the type identified by the scope value per the applicable profile.