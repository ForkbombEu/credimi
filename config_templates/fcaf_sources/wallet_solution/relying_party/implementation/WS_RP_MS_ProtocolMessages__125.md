# WS_RP_MS_ProtocolMessages_125

## Objective
Test that Wallet returns an error after an HTTP 200 response not of form json.

## References
[OID4VP] Sections 8.2, 8.5

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends a request, with parameter 'response_mode=direct_post.jwt'.
3. The Wallet processes the request.
4. The Wallet responds with request object.
5. Verifier sends an HTTP status code of 200 + a plain text body.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet evaluates the request and proceeds without returning an error.
4. The wallet submits the Authorization Response to the verifier's response_uri.
5. The wallet detects that the verifier's HTTP 200 response body is not valid JSON and returns an error.
