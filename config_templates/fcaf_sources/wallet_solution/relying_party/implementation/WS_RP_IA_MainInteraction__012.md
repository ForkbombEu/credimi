# WS_RP_IA_MainInteraction_012

## Objective
Verify the Wallet sends only the specific claims requested by the Verifier

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
Wallet has claims (Name, address, phone)

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query asking only for name claim.
3. Wallet sends only specific claim requested

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet hides "Address" and "Phone." Presents user with name claim on UI.