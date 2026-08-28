# WS_RP_SM_TrustMechanisms_015

## Objective
Test the Wallet will not return a credential if its X.509 Certificate trust chain cannot be verified against the specified etsi_tl or its cascading lists.

## References
[OID4VP 6.1.1.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions
A mock ETSI trusted List is active to be used

## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request containing a DCQL query with trusted_authorities of type etsi_tl, with values containing the identifier of the Trusted List, but no wallet credential's X.509 in the cascading Trusted List entries.
3. The Wallet returns a response to the Verifier.

## Expected results
1. Wallet and Verifier can interact.
2. The Wallet successfully receives the DCQL query.
3. The Wallet does not return any credential in the Authorization Response.