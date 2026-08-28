# WS_RP_SM_TrustMechanisms_010

## Objective
Test the Wallet can resolve a trusted_authorities request of type etsi_tl by matching the credential's issuer against a recognized ETSI Trusted List scenario when no Match.

## References
[OID4VP 6.1.1.2]

## Profile applicability
Wallet supports trusted authorities query based on ETSI Trust List.

## EUDI-wallet relevancy
EUDI_generic | EUDI_optional

## Preconditions
Credential Issuer is not on Trusted List
A mock ETSI trusted List is active to be used

## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query requesting a credential with a "trusted_authorities" property with its type being "etsi_tl".
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3a. The Wallet checks the issuer of its stored credential against the entries in the Trusted List.
3b. It confirms the Issuer's status according to the list's metadata.
3c. The Wallet does not use the credential if the Issuer is not on the list (even if the credential is valid otherwise).
3d. Wallet will make sure it only returns privacy-safe error response to verifier.

