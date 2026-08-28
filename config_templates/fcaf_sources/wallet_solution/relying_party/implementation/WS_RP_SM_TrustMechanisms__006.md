# WS_RP_SM_TrustMechanisms_006

## Objective
Test the Wallet processes a DCQL-query with an "aki" correctly when it does contain a matching credential, with the match in a sub-CA certificate.

## References
[OID4VP 6.1.1.1]
[RFC5280]

## Profile applicability
Wallet supports trusted authorities query based on 'aki'

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions
The stored credential in the Wallet used in test is issued under a certificate chain containing a specified AuthorityKeyIdentifier in a sub-CA certificate, with the chain having a depth of at least 3.

## Test Scenario

1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a "trusted_authorities" property with its type being "aki", and its value contains a specific keyIdentifier of a sub-CA certificate in the certificate chain of the credential issuer
3. The Wallet evaluates the request and allows the user to continue with presenting matching credentials.
4. User selects the available credential and approves to present it.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet allows the user to select the credential matching the certificate chain.
4. The Wallet presents the selected credential.

