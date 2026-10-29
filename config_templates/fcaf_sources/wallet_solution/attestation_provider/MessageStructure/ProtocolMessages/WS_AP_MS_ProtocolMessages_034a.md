# WS_AP_MS_ProtocolMessages_034a

## Objective

Verify that the Wallet sends the HTTP Request Body in the correct format when sending a Pushed Authorization Request, when using Pushed Authorization Requests for issuance.

## References

- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.4
- [RFC9126] section 2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. End-user is engaging with a Credential Issuer using a User-agent.
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully obtained Credential Issuer Metadata.
4. The Wallet sends an HTTP POST Request to the selected Authorization Server's PAR endpoint.

## Test Scenario

1. Verify the `Content-Type` header of the received HTTP Request.

## Expected results

1. The HTTP POST Request has the header `Content-Type` with value `application/x-www-form-urlencoded`.
