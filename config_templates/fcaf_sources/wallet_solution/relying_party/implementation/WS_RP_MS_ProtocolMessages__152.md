# WS_RP_MS_ProtocolMessages_152

## Objective
Test the Wallet returns the error message invalid_request_uri_method when the value of request_uri_method is invalid.

## References
[OID4VP Section 8.5]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with verifier.
2. The Verifier sends an Authorization request with request_uri, setting request_uri_method to an invalid value.
3. Wallet processes request.
4. Test the response returned by the Wallet to the Verifier.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The Wallet identifies it cannot fetch request object.
4. Verify the Wallet returns an error response, where the error parameter is exactly invalid_request_uri_method.

