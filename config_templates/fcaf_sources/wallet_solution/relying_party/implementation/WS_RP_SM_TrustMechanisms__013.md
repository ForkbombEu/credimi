# WS_RP_SM_TrustMechanisms_013

## Objective
Verify the Wallet can navigate an ETSI Trusted List (List of Trusted Lists) to identify and validate a Trust Service Provider’s (TSP) certificate.

## References
[OID4VP 6.1.1.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions
The Wallet contains a credential issued by a valid Trust Service Provider (TSP).
A mock ETSI trusted List is active to be used

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request containing a DCQL query with trusted_authorities of type etsi_tl whose value is the URL of the ETSI Trusted List.
3. The Wallet returns a response to the Verifier.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet successfully receives the DCQL query.
3. The Wallet returns an Authorization Response containing the credential whose issuing Trust Service Provider is listed in the linked Trusted List or its cascading Trusted Lists.