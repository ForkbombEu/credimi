# WS_AP_IA_MainInteraction_010c

## Objective

Verify that the Wallet sends a reference to a successfully Pushed Authorization Request to the Authorization Server's Authorization Endpoint, when requesting a Credential.

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4
- [OpenID4VCI] section 5.1
- [RFC9126] section 2.2, 4

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
4. Wallet successfully obtained the Authorization Server Metadata.
5. Wallet successfully pushed an Authentication Request to the Authorization Server's `pushed_authorization_request_endpoint`.
6. Wallet send an Authorization Request to the Authorization Server's `authorization_endpoint`.
7. Wallet send a reference to a Pushed Authorization Request in the Authorization Request to the Authorization Server's Authorization Endpoint.

## Test Scenario

1. Verify the contents of the Authorization Request.

## Expected results

1. The value of the `request_uri` is the same value as the value of the request_uri send in response to the Pushed Authentication Request submission, by the PAR endpoint of the Authorization Server.
