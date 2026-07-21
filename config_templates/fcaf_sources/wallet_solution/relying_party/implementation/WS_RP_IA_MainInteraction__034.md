# WS_RP_IA_MainInteraction_034

## Objective
Test when the Wallet cannot deliver all non-optional Credentials requested by the Verifier, it MUST NOT return any Credential(s).

## References
[OID4VP 6.4]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The verifier sends a DCQL query requesting 2 non-optional credentials; The wallet only has 1 of them
3. Wallet handles query

## Expected results
1. Wallet sees it can return 1 credential, it cannot return the other
2. Wallet does NOT return EITHER credential, sends a privacy keeping requirement not met response