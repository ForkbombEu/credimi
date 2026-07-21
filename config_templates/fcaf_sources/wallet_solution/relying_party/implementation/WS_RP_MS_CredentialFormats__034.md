# WS_RP_MS_CredentialFormats_034

## Objective
Test that when a wallet matches a mdoc-based Credential, the CBOR values are matched by first converting to JSON.

## References
[OID4VP 6.3]
Section 6.1 of [RFC8949]

## Profile applicability
value matching with ISO mdoc is implemented by wallet

## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends a DCQL query requesting a specific element from a mdoc
3. Wallet handles Query

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3a. The Wallet retrieves the mdoc from secure storage, noting CBOR format.
3b.  The Wallet converts the CBOR value into a JSON-compatible representation to compare it against the query filter - it uses the guidance from Section 6.1 of [RFC8949] to avoid issues like Data Type fidelity
3c. The Wallet performs the matching logic

