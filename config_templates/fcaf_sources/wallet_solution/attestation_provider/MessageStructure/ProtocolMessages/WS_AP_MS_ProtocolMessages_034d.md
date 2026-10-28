# WS_AP_MS_ProtocolMessages_034d

## Objective

Verify that the Wallet sends a `request` parameter in the HTTP request is a correctly formatted JWT (Signed Request Object) when sending a Pushed Authorization Request, when using Pushed Authorization Requests for issuance.

## References

- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.4
- [RFC9126] section 3

## Profile applicability

Wallet that uses JAR during PAR for the Authorization Request to the Authorization Server on Credential Issuance.

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. End-user is engaging with a Credential Issuer using a User-agent.
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully obtained Credential Issuer Metadata.
4. The Wallet sends an HTTP POST Request to the selected Authorization Server's PAR endpoint.
5. The HTTP POST Request to the selected Authorization Server's PAR endpoint contains a `request` parameter.
6. The `request` parameter sent to the selected Authorization Server's PAR endpoint has the structure of a signed JWT.

## Test Scenario

1. Perform all Shared_JWT_JWS test cases on the value of the `request` parameter.

## Expected results

1. All test cases pass.
