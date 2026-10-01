# WS_RP_MS_ProtocolMessages_064

## Objective

Test the Wallet accepts DCQL-query credential object property "id" when it contains valid characters (consists of alphanumeric, underscore (_), or hyphen (-)).

## References

- [OpenID4VP] Section 6.1

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Preconditions

None

## Test Scenario

1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with valid "id" property containing characters of the form alphanumeric, underscore (_) and hyphen (-).
3. The Wallet evaluates the request.

## Expected results

1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet accepts and process the request, continuing to respond with matching response.
