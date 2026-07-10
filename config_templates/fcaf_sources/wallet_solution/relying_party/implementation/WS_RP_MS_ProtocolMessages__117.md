# WS_RP_MS_ProtocolMessages_117

## Objective
Test that if credential_sets is not provided, the wallet interprets this as the Verifier requesting presentations for all Credentials in credentials to be returned.

## References
[OID4VP 6.4.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query, with "credentials" asking for 2 credentials, "credential_sets" is NOT included.
3. Wallet handles query.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet received query.
3. The Wallet interprets the missing credential_sets as a mandatory request for all defined credentials without any "credential_sets" rules.
    1. The Wallet offers both credentials to the user as a single, required bundle.
    2. Upon consent, the Wallet generates a Verifiable Presentation containing both credentials.