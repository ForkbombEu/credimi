# WS_RP_SM_TrustMechanisms_002

## Objective
Test that the Wallet when processing trusted_authorities, the type "aki" is supported.

## References
[OID4VP 6.1.1.1]
Section 4.2.1.1 of [RFC5280]

## Profile applicability
Wallet supports trusted authorities query based on 'aki'.

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends a DCQL query containing a credential query with trusted_authorities of type "aki", and a specific KeyIdentifier value that matches the AuthorityKeyIdentifier of an X.509 certificate.
3. The Wallet returns a response to the Verifier.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet successfully receives the DCQL query.
3. The Wallet returns an Authorization Response containing the matching credential.

