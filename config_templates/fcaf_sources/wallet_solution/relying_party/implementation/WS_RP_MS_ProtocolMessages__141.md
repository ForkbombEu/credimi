# WS_RP_MS_ProtocolMessages_141

## Objective
Test the Wallet responds with invalid_request when the request contains both a dcql_query parameter and a scope parameter referencing a DCQL query.

## References
[OID4VP Section 8]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request that includes both a valid dcql_query object & a scope value intended to trigger a DCQL based credential request.
3. Wallet processes the request and identifies the conflicting parameters.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The Wallet does NOT proceed to credential selection or user authorization.
4. The Wallet returns an error response where the error parameter is exactly invalid_request.

