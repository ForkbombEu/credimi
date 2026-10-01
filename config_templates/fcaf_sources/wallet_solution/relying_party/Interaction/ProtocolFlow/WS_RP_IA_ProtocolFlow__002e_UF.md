# WS_RP_IA_ProtocolFlow_002e_UF

## Objective

Verify that, in the presentation flow via Redirects, the Wallet rejects receiving a Signed Authorization Request using JWT-Secured Authorization Request (JAR, [RFC9101]) through the `request_uri` parameter, when the `request_uri_method` parameter has an invalid value.

## References

- [HAIP] section 5.1
- [OpenID4VP] section 5, 5.1, 8.5
- [RFC9101] section 5

## Profile applicability

Wallet does support the Request URI Method `post`

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Presentation via Redirects
Same-device flow

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. End-user interacts with Verifier using a User-agent.

## Test Scenario

1. The Verifier provides an invocation URL for a OID4VP request via redirects using Request Object by reference (i.e. `eu-eaap://` link), with
    1. a valid HTTPS link as value for the `request_uri` parameter,
    2. an invalid value, i.e. anything other than `get` or `post`, for the `request_uri_method` parameter.
2. End-user triggers the provided link to be followed.

## Expected results

1. User-agent presents option to End-user to engage presentation.
2. The Wallet is invoked and does not engage with the Verifier, i.e. no request is received at the endpoint stated in `request_uri`, informing the User on the invalid link provided, if applicable.
