# WS_RP_SM_TrustMechanisms_012

## Objective
The Wallet can parse the identifier of a Trusted List as specified in ETSI TS 119 612 [ETSI.TL], in a "trusted_authorities" request.

## References
[OID4VP 6.1.1.2]
ETSI TS 119 612 [ETSI.TL]

## Profile applicability
Wallet supports trusted authorities query based on ETSI Trust List.

## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions
A mock ETSI trusted List is active to be used

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query requesting a credential with a "trusted_authorities" property with its type being "etsi_tl".
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3a. The Wallet successfully extracts the URI string
3b. Wallet fetches the list, validates that it follows the ETSI TS 119 612 schema (correctly handling XML or JSON).

