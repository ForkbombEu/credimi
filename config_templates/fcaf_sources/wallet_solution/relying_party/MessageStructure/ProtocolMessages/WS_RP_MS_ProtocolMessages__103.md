# WS_RP_MS_ProtocolMessages_103

## Objective

Test that the credential_sets property "required" will default to True if omitted.

## References

- [OpenID4VP] Section 6.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. The Wallet engages with the Verifier.
2. Verifier sends a DCQL query with a "credential_sets" property "required" omitted
3. The Wallet evaluates the request.

## Expected results

1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet reacts identically to the test case where "required" was included and set to True.
