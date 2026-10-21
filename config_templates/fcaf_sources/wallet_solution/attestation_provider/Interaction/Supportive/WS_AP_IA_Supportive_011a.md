# WS_AP_IA_Supportive_011a

## Objective

Verify that the Wallet can handle an correct response when submitting a Pushed Authorization Request (PAR) to the Authorization Server's Pushed Authentication Request Endpoint.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4
- [OpenID4VCI] section 5.1.4
- [RFC9126] section 2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.
Credential Issuance using the Authorization Code Flow.

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully obtained Credential Issuer Metadata.
4. Wallet successfully obtained the Authorization Server Metadata, including `pushed_authorization_request_endpoint`.
5. The Wallet sent a valid Pushed Authorization Request to the Authorization Server's Pushed Authorization Request Endpoint.

## Test Scenario

1. Issuer responds with a valid PAR Response. That is a HTTP Response:
    1. with a HTTP Header `Content-Type` with value `application/json`, containing:
    2. with the HTTP Response Body in JSON format:
        1. contains a `request_uri`
        2. contains a `expires_in`

## Expected results

1. The subsequent Authorization Request sent through the User-agent contains the exact `request_uri` value returned by the Authorization Server.
