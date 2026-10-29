# WS_AP_IA_Supportive_010a

## Objective

Verify that the Wallet sends a Pushed Authorization Request (PAR) to the Authorization Server's Pushed Authentication Request Endpoint as a HTTP POST request to the endpoint.

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
3. Wallet successfully obtained the Credential Issuer Metadata.
4. Wallet successfully obtained the Authorization Server Metadata, including `pushed_authorization_request_endpoint`.

## Test Scenario

1. End-user selects a Credential in the Wallet (based on `credential_configuration_ids` and Credential Issuer Metadata) to obtain such Credential from the Credential Issuer.

## Expected results

1. Wallet connects to the Authorization Server's Pushed Authorization Request (PAR) Endpoint using TLS, making a HTTP request:
    1. Uses the POST method.
    2. Has the 'hostname', and 'port-number' if applicable, of the `pushed_authorization_request_endpoint` in the Authorization Server's Metadata.
    3. Requests the 'path' of the `pushed_authorization_request_endpoint`, without a query or fragment part.
