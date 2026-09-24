# WS_RP_IA_ProtocolFlow_002b

## Objective

Verify that, in the presentation flow via Redirects, a "Request URI method `post`"-supporting Wallet supports receiving a Signed Authorization Request using JWT-Secured Authorization Request (JAR, [RFC9101]) through the `request_uri` parameter, when the `request_uri_method` parameter has value `post`.

## References

- [HAIP] section 5.1
- [OpenID4VP] section 5, 5.1, 5.10
- [RFC9101] section 5

## Profile applicability

Wallet does support the Request URI Method `post`

## EUDI-wallet relevancy

EUDI_generic | EUDI_optional

## Technology

Presentation via Redirects
Same-device flow

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. End-user interacts with Verifier using a User-agent.

## Test Scenario

1. The Verifier provides an invocation URL for a OID4VP request via redirects using Request Object by reference (i.e. `eu-eaap://` link), with
    1. a valid HTTPS link as value for the `request_uri` parameter,
    2. the value `post` for the `request_uri_method` parameter.
2. End-user triggers the provided link to be followed.
3. Verify the request made to the `request_uri` endpoint.
4. The Verifier responds to the HTTP request at the `request_uri` endpoint with a valid Signed Request Object requesting 'default_credential_A'.

## Expected results

1. User-agent presents an option to the End-user to engage the presentation.
2. The Wallet is invoked.
3. Wallet makes a HTTP request at the Verifier's endpoint, using a TLS connection. The HTTP request:
    1. Uses the POST method.
    2. Has the 'hostname', and 'port-number' if applicable, of the provided `request_uri` as `Host` header.
    3. Requests the 'path' of the `request_uri`, without a query or fragment part.
4. Wallet responds to the request with a presentation of 'default_credential_A'.
