# WS_AP_MS_ProtocolMessages_040b

## Objective

Verify that the Wallet sends a correct Pushed Authorization Request, when authentication to a Authorization Server for requesting issuance of a Credential.

## References

- [ETSI TS 119 472-3] section 4.4
- [HAIP] section 4.3
- [OpenID4VCI] section 5.1.2

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
7. Wallet send a syntactically correct Authorization Request Request Object.

## Test Scenario

1. Verify the `scope` parameter in the Pushed Authorization Request.

## Expected results

1. The value of the `scope` parameter matches with one or more values, with whitespace(s) as separator, where each value matches a `scope` of a `credential_configurations_supported` of the Credential Issuer as stated in the Credential Issuer's metadata.

    - NOTE: this assumes the Credential Issuer's Metadata does include a unique `scope` for each supported Credential Configuration.
