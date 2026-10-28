# WS_AP_MS_ProtocolMessages_040a

## Objective

Verify that the Wallet sends a syntactically correct Pushed Authorization Request, when authentication to a Authorization Server for requesting issuance of a Credential.

## References

- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.2, 5.1.3
- [RFC6749] section 4.1.1
- [RFC7636] section 4.3

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. End-user is engaging with a Credential Issuer using a User-agent.
3. Wallet started engagement with Credential Issuer.
4. Wallet successfully obtained Credential Issuer Metadata.
5. Wallet send a valid and trusted PAR request the Authorization Server's PAR Endpoint.
6. Wallet send a valid Authorization Request to the Authorization Server's Authorization Endpoint.

## Test Scenario

1. Verify the Pushed Authorization Request.

## Expected results

1. The Pushed Authorization Request  contains the following top-level request parameters:
    1. `scope`
    2. `response_type`
    3. `client_id`
    4. `redirect_uri`
    5. `code_challenge`
    6. `code_challenge_method`
    7. (optionally) `issuer_state`
    8. (optionally) `resource`
