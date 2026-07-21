# WS_RP_SM_TrustMechanisms_008

## Objective
Test the Wallet processes a DCQL-query with an aki trusted_authorities value correctly when the Wallet does NOT contain a credential whose X.509 certificate chain matches the specified AuthorityKeyIdentifier.

## References
[OID4VP 6.1.1.1]

## Profile applicability
Wallet supports trusted authorities query based on 'aki'.

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends a DCQL query containing a credential query with trusted_authorities of type aki and a KeyIdentifier value that does NOT match the AuthorityKeyIdentifier of any X.509 certificate in the chain of any credential the Wallet holds.
3. The Wallet returns a response to the Verifier.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet successfully receives the DCQL query.
3. The Wallet does not return any credential in the Authorization Response (no credential satisfies the aki constraint).