# WS_RP_MS_ProtocolMessages_067

## Objective
Test the Wallet accepts a DCQL-query with credential `format` values conform to those defined in [OID4VP] Appendix B.

## References
[OID4VP] sections 6.1
[HAIP]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the Verifier.
2. The Verifier sends an Authorization Request with a DCQL-query with a credential with a "format" value that is one of `mso_mdoc` or `dc+sd-jwt`.
3. The Wallet evaluates the request.

## Expected results
1. Wallet and Verifier can interact.
2. Wallet receives the request.
3. Wallet accepts and process the request, continuing to respond with matching response without any error.
