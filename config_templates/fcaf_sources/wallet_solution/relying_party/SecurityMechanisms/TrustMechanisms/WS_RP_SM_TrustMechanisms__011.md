# WS_RP_SM_TrustMechanisms_011

## Objective

Test the wallet can resolve a trusted_authorities request of type etsi_tl by matching the credential's issuer against a recognized ETSI Trusted List scenario when invalid trusted list.

## References

- [OpenID4VP] Section 6.1.1.2

## Profile applicability

Wallet supports trusted authorities query based on ETSI Trust List.

## EUDI-wallet relevancy

EUDI_generic | EUDI_optional

## Preconditions

1. Scenario when the Trusted List signature is invalid or if the URL in the query is unreachable.
2. A mock ETSI trusted List is active to be used

## Test Scenario

1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query requesting a credential with a "trusted_authorities" property with its type being "etsi_tl".
3. The Wallet evaluates the request.

## Expected results

1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet checks the issuer of its stored credential against the entries in the Trusted List.
    - It confirms the Issuer's status according to the list's metadata.
    - The wallet will not use the credential and returns an error if the Trusted List signature is invalid or if the URL in the query is unreachable.
    - Wallet will make sure it only returns privacy-safe error response to verifier.
