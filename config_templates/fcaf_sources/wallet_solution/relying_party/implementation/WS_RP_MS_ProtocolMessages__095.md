# WS_RP_MS_ProtocolMessages_095

## Objective
Test the Wallet when processing trusted_authorities, type "etsi_tl" is supported.

## References
[OID4VP 6.1.1]

## Profile applicability
Wallet supports trusted authorities query based on ETSI Trust List.

## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query requesting a credential with a "trusted_authorities" property with its type being "etsi_tl".
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. No errors returned, with the wallet using the Trusted List when performing credential matching.

