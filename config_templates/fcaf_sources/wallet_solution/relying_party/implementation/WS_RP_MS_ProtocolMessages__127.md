# WS_RP_MS_ProtocolMessages_127

## Objective
Test the Wallet ignores an unrecognized parameter provided by the Verifier in its JSON response, returned from a response_uri.

## References
[OID4VP Section 8.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request, with response_type=vp_token, and parameter response_uri.
3. The Wallet performs HTTP POST to the response_uri.
4. The Verifier responds with HTTP 200 and a JSON body (containing an unrecognized parameter).
5. Wallet processes JSON and triggers User agent.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. The wallet submits the Authorization Response to the verifier's response_uri.
4. The wallet receives the HTTP 200 JSON response.
5. Wallet opens redirect_uri, shows user success page.

