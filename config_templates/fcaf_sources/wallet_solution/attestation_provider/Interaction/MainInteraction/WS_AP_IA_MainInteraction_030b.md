# WS_AP_IA_MainInteraction_030b

## Objective

Verify that the Wallet sends a Credential Request using a correctly formatted HTTP Request to the Credential Endpoint of the Credential Issuer.

## References

- [ETSI TS 119 472-3] section 4.6
- [HAIP] section 4
- [OpenID4VCI] section 8.2

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

OpenID4VCI Credential Endpoint requests.

## Preconditions

1. Wallet is set to 'default_configuration_1'.
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully authenticated to the (Authorization Server of the) Credential Issuer.
4. Wallet successfully obtained an Access Token from the Token Endpoint of the (Authorization Server of the) Credential Issuer.
5. Wallet send a HTTP Request to the Credential Endpoint.

## Test Scenario

1. Verify the header of the HTTP Request of the Credential Request.
2. Perform all Shared_JSON test cases on the contents of the Body of the HTTP Request.

## Expected results

1. The Wallet sends a HTTP Request with a `Content-Type` header with the value `application/json`.
2. All test cases pass.

    - TODO Distinguish Credential Request encryption applicable or not?
