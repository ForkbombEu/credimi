# WS_AP_MS_ProtocolMessages_050e

## Objective

Verify that the Wallet sends a syntactically correct `proofs` parameter in the Credential Request to the Credential Endpoint of the Credential Issuer, when requesting issuance of a Credential with Key Binding.

## References

- [ETSI TS 119 472-3] section 4.6.1
- [HAIP] section 4.5.1
- [OpenID4VCI] section 8.2, F

## Profile applicability

None

## EUDI-wallet relevancy

EUDI_generic | EUDI_required

## Technology

Issuance via Redirects.

## Preconditions

1. Wallet is set to 'default_configuration_1'
2. Wallet started engagement with Credential Issuer.
3. Wallet successfully authenticated to the (Authorization Server of the) Credential Issuer.
4. Wallet successfully obtained an Access Token from the Token Endpoint of the (Authorization Server of the) Credential Issuer.
5. Wallet send a HTTP Request to the Credential Endpoint.
6. Wallet send a Credential Request a containing syntactically correct `proofs` parameter to the Credential Endpoint.

## Test Scenario

1. Verify the presence of properties in the `proofs` parameter in the HTTP Request send to the Credential Endpoint.

## Expected results

1. The `proofs` parameter in the HTTP Request contains either a `jwt` or `attestation` property, not both.
