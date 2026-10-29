# WS_AP_IA_MainInteraction_010a

## Objective

Verify that the Wallet sends an Authorization Request to the Authorization Server's Authorization Endpoint, after successfully pushing the Authorization Request (PAR).

## References

- [ETSI TS 119 472-3] section 4.1
- [HAIP] section 4
- [OpenID4VCI] section 5.1
- [RFC8414] section 2

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
3. Wallet successfully obtained the Credential Issuer Metadata.
4. Wallet successfully obtained the Authorization Server Metadata.
5. Wallet successfully pushed an Authentication Request to the Authorization Server's `pushed_authorization_request_endpoint`.

## Test Scenario

1. Verify the Authorization Request.

## Expected results

1. The Wallet sends a HTTP Request to the Authorization Endpoint of the Authorization Server as listed in the Credential Issuer Metadata. The HTTP Request:
    1. Uses the GET or POST method.
    2. Has the 'hostname', and 'port-number' if applicable, of the `authorization_endpoint` in the Authorization Server's Metadata.
    3. Requests the 'path' and any 'query' component of the `authorization_endpoint`, without a fragment component.
