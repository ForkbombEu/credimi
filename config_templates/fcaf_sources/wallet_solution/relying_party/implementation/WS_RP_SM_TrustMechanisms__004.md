# WS_RP_SM_TrustMechanisms_004

## Objective
Test the Wallet processes a DCQL-query with an "aki" correctly when it does not contain a matching credential.

## References
[OID4VP 6.1.1.1]
[RFC5280]

## Profile applicability
Wallet supports trusted authorities query based on 'aki'

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The stored credential(s) in the Wallet used in test is/are issued under a certificate chain with none of the certificates matching a specified AuthorityKeyIdentifier.

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a "trusted_authorities" property with its type being "aki", and its 'value' contains only a value not matching any AuthorizationKeyIdentifier in the credential issuer's certificate chain.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet informs the user it has no matching credential available, and does not (allow to) continue to present any credential to the verifier.

