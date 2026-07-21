# WS_RP_MS_ProtocolMessages_102

## Objective
Test that the credential_sets property "required" is correctly supported by the wallet when a PID is missing.

## References
[OID4VP 6.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a "credential_sets" property, including "required" property set as true for a set containing a PID that the wallet is missing. No other sets available.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. The Wallet realises it doesn't have PID and sees "required": true, hence it must NOT offer this set. No other sets are available, so the transaction stops, making sure it sends a privacy preserving response.

