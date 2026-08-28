# WS_RP_MS_ProtocolMessages_134

## Objective
Verify that for direct_post.jwt, the Wallet encapsulates the response in a JWT and delivers it via an HTTP POST body with the correct Content-Type.

## References
[OID4VP Section 8.2]

## Profile applicability


## EUDI-wallet relevancy
EUDI_generic | EUDI_required

## Preconditions


## Test Scenario
1. The Wallet engages with the verifier.
2. The Verifier sends an Authorization request to the wallet, with response mode=direct_post.jwt.
3. Wallet processes request, returns encrypted response.
4. Verify the HTTP POST request content-type header shows it is application/x-www-form-urlencoded, and the HTTP POST request body contains only the Wallet response parameter holding the JWT.

## Expected results
1. Wallet-verifier interaction is successfully initiated.
2. Wallet receives request.
3. Verify the wallet sends an HTTP POST request.
4. Body is form-encoded and contains the response parameter holding the JWT.

