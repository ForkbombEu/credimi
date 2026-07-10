# WS_RP_MS_ProtocolMessages_101

## Objective
Test that the credential_sets property "required" is correctly supported by the wallet when all credentials exist.

## References
[OID4VP 6.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a "credential_sets" property, including "required" property set as true for a set containing a PID that the wallet has.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet identifies that the set containing the PID is marked as required: true.
    1. The Wallet confirms the PID exists and satisfies the query. It prepares the PID for presentation.
    2. The Wallet presents the PID to the user. Because it is marked as required, the UI should indicate it is mandatory (e.g., no "skip" option for this specific set).
    3. The Wallet generates the Verifiable Presentation containing the PID as required by the Verifier

